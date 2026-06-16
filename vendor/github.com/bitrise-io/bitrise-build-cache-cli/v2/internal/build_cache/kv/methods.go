package kv

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retry"
	"github.com/dustin/go-humanize"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	remoteexecution "github.com/bitrise-io/bitrise-build-cache-cli/v2/proto/build/bazel/remote/execution/v2"
)

type PutParams struct {
	Name            string
	Sha256Sum       string
	FileSize        int64
	Offset          int64
	DeleteOnRewrite bool
}

type WriteStatus struct {
	Complete      bool
	CommittedSize int64
}

type FileDigest struct {
	Sha256Sum   string
	SizeInBytes int64
}

func (c *Client) GetCapabilities(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	callCtx := metadata.NewOutgoingContext(timeoutCtx, c.getMethodCallMetadata(true))

	_, err := c.capabilitiesClient.GetCapabilities(callCtx, &remoteexecution.GetCapabilitiesRequest{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return ErrCacheUnauthenticated
		}

		return fmt.Errorf("get capabilities: %w", err)
	}

	return nil
}

func (c *Client) GetCapabilitiesWithRetry(ctx context.Context) error {
	//nolint:wrapcheck
	return retry.Times(10).Wait(3 * time.Second).TryWithAbort(func(attempt uint) (error, bool) {
		if attempt > 0 {
			c.logger.Debugf("Retrying GetCapabilities... (attempt %d)", attempt)
		}

		if err := c.GetCapabilities(ctx); err != nil {
			c.logger.Errorf("Error in GetCapabilities attempt %d: %s", attempt, err)
			if errors.Is(err, ErrCacheUnauthenticated) {
				return ErrCacheUnauthenticated, true
			}

			return err, false
		}

		return nil, false
	})
}

func (c *Client) initiatePut(ctx context.Context, params PutParams) (*writer, error) {
	md := metadata.Join(c.getMethodCallMetadata(false), metadata.Pairs(
		"x-flare-blob-validation-sha256", params.Sha256Sum,
		"x-flare-blob-validation-level", "error",
		"x-flare-no-skip-duplicate-writes", "true",
	))
	if params.DeleteOnRewrite {
		md.Set("x-cache-delete-on-rewrite", "true")
	}
	// Timeout is the responsibility of the caller
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := c.bitriseKVClient.Put(ctx)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return nil, ErrCacheUnauthenticated
		}

		return nil, fmt.Errorf("initiate put: %w", err)
	}

	resourceName := fmt.Sprintf("kv/%s", params.Name)

	return &writer{
		stream:       stream,
		resourceName: resourceName,
		offset:       params.Offset,
		fileSize:     params.FileSize,
	}, nil
}

func (c *Client) initiateGet(ctx context.Context, logger log.Logger, name string, offset int64) (*reader, error) {
	resourceName := fmt.Sprintf("kv/%s", name)

	// Timeout is the responsibility of the caller
	ctx = metadata.NewOutgoingContext(ctx, c.getMethodCallMetadata(false))

	readReq := &bytestream.ReadRequest{
		ResourceName: resourceName,
		ReadOffset:   offset,
		ReadLimit:    0,
	}
	stream, err := c.bitriseKVClient.Get(ctx, readReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return nil, ErrCacheUnauthenticated
		}

		return nil, fmt.Errorf("initiate get: %w", err)
	}

	r := &reader{
		logger:        logger,
		stream:        stream,
		metadataReady: make(chan struct{}),
	}
	go r.readStreamMetadata()

	return r, nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	resourceName := fmt.Sprintf("kv/%s", name)

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	callCtx := metadata.NewOutgoingContext(timeoutCtx, c.getMethodCallMetadata(false))

	readReq := &bytestream.ReadRequest{
		ResourceName: resourceName,
		ReadOffset:   0,
		ReadLimit:    0,
	}
	_, err := c.bitriseKVClient.Delete(callCtx, readReq)
	if err != nil {
		return fmt.Errorf("initiate delete: %w", err)
	}

	return nil
}

func (c *Client) findMissing(ctx context.Context,
	req *remoteexecution.FindMissingBlobsRequest,
) ([]*FileDigest, error) {
	var resp *remoteexecution.FindMissingBlobsResponse
	err := retry.Times(3).Wait(3 * time.Second).TryWithAbort(func(attempt uint) (error, bool) {
		if attempt > 0 {
			c.logger.Debugf("Retrying FindMissingBlobs... (attempt %d)", attempt)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		callCtx := metadata.NewOutgoingContext(timeoutCtx, c.getMethodCallMetadata(false))

		var err error
		resp, err = c.casClient.FindMissingBlobs(callCtx, req)

		cancel()

		if err != nil {
			c.logger.Errorf("Error in FindMissingBlobs attempt %d: %s", attempt, err)

			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Unauthenticated {
				return ErrCacheUnauthenticated, false
			}

			return fmt.Errorf("find missing blobs: %w", err), true
		}

		return nil, false
	})
	if err != nil {
		return nil, fmt.Errorf("with retries: %w", err)
	}

	return convertToFileDigests(resp.GetMissingBlobDigests()), nil
}

func (c *Client) findMissingChunked(ctx context.Context,
	req *remoteexecution.FindMissingBlobsRequest,
	digests []*FileDigest,
	blobDigests []*remoteexecution.Digest,
	gRPCLimitBytes int,
) ([]*FileDigest, error) {
	var missingBlobs []*FileDigest
	// Chunk up request blobs to fit into gRPC limits
	// Calculate the unit size of a blob (in practice can differ to the theoretical sha256(32 bytes) + size(8 bytes) = 40 bytes)
	digestUnitSize := float64(len(req.String())) / float64(len(digests))
	maxDigests := int(float64(gRPCLimitBytes) / digestUnitSize)
	for startIndex := 0; startIndex < len(digests); startIndex += maxDigests {
		endIndex := startIndex + maxDigests
		if endIndex > len(digests) {
			endIndex = len(digests)
		}
		req.BlobDigests = blobDigests[startIndex:endIndex]
		c.logger.Debugf("Calling FindMissingBlobs for chunk: digests[%d:%d]", startIndex, endIndex)

		var resp []*FileDigest
		var err error
		if resp, err = c.findMissing(ctx, req); err != nil {
			return nil, fmt.Errorf("find missing blobs: %w", err)
		}

		missingBlobs = append(missingBlobs, resp...)
	}

	return missingBlobs, nil
}

func (c *Client) FindMissing(ctx context.Context, digests []*FileDigest) ([]*FileDigest, error) {
	blobDigests := convertToBlobDigests(digests)
	req := &remoteexecution.FindMissingBlobsRequest{
		BlobDigests: blobDigests,
	}
	c.logger.Debugf("Size of FindMissingBlobs request for %d blobs is %s", len(digests), humanize.Bytes(uint64(len(req.String()))))
	gRPCLimitBytes := 4 * 1024 * 1024 // gRPC limit is 4 MiB
	if len(req.String()) > gRPCLimitBytes {
		return c.findMissingChunked(ctx, req, digests, blobDigests, gRPCLimitBytes)
	}

	return c.findMissing(ctx, req)
}

func convertToBlobDigests(digests []*FileDigest) []*remoteexecution.Digest {
	out := make([]*remoteexecution.Digest, 0, len(digests))

	for _, d := range digests {
		out = append(out, &remoteexecution.Digest{
			Hash:      d.Sha256Sum,
			SizeBytes: d.SizeInBytes,
		})
	}

	return out
}

func convertToFileDigests(digests []*remoteexecution.Digest) []*FileDigest {
	out := make([]*FileDigest, 0, len(digests))

	for _, d := range digests {
		out = append(out, &FileDigest{
			Sha256Sum:   d.GetHash(),
			SizeInBytes: d.GetSizeBytes(),
		})
	}

	return out
}

func (c *Client) getMethodCallMetadata(logMD bool) metadata.MD {
	md := metadata.Pairs(
		"authorization", fmt.Sprintf("bearer %s", c.authConfig.AuthToken),
		"x-flare-buildtool", c.clientName)

	if c.cacheOperationID != "" {
		md.Set("x-cache-operation-id", c.cacheOperationID)
	}

	if c.authConfig.WorkspaceID != "" {
		md.Set("x-org-id", c.authConfig.WorkspaceID)
	}
	if c.cacheConfigMetadata.BitriseAppID != "" {
		md.Set("x-app-id", c.cacheConfigMetadata.BitriseAppID)
	}
	if c.cacheConfigMetadata.BitriseBuildID != "" {
		md.Set("x-flare-build-id", c.cacheConfigMetadata.BitriseBuildID)
	}
	if c.cacheConfigMetadata.BitriseWorkflowName != "" {
		md.Set("x-workflow-name", c.cacheConfigMetadata.BitriseWorkflowName)
	}
	if c.cacheConfigMetadata.BitriseStepExecutionID != "" {
		md.Set("x-flare-step-id", c.cacheConfigMetadata.BitriseStepExecutionID)
	}
	if c.cacheConfigMetadata.GitMetadata.RepoURL != "" {
		md.Set("x-repository-url", c.cacheConfigMetadata.GitMetadata.RepoURL)
	}
	if c.cacheConfigMetadata.CIProvider != "" {
		md.Set("x-ci-provider", c.cacheConfigMetadata.CIProvider)
	}

	md.Set("x-flare-blob-validation-level", "WARN")
	md.Set("x-flare-ac-validation-mode", "fast")

	rmd := &remoteexecution.RequestMetadata{
		ToolInvocationId: c.invocationID,
		ToolDetails: &remoteexecution.ToolDetails{
			ToolName: c.clientName,
		},
	}
	serializedRMD, err := proto.Marshal(rmd)
	if err != nil {
		c.logger.Errorf("Failed to marshal RequestMetadata: %v", err)
	} else {
		md.Set("build.bazel.remote.execution.v2.requestmetadata-bin", string(serializedRMD))
	}

	if logMD {
		logMd := md.Copy()
		logMd.Delete("authorization")
		logMd.Set("build.bazel.remote.execution.v2.requestmetadata-bin", rmd.String())
		c.logger.TDebugf("metadata: %+v", logMd)
	}

	return md
}

func (c *Client) QueryWriteStatus(ctx context.Context, name string) (WriteStatus, error) {
	resourceName := fmt.Sprintf("kv/%s", name)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	callCtx := metadata.NewOutgoingContext(timeoutCtx, c.getMethodCallMetadata(false))
	resp, err := c.bitriseKVClient.WriteStatus(callCtx, &bytestream.QueryWriteStatusRequest{
		ResourceName: resourceName,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return WriteStatus{}, ErrCacheUnauthenticated
		}

		return WriteStatus{}, fmt.Errorf("query write status: %w", err)
	}

	return WriteStatus{
		Complete:      resp.GetComplete(),
		CommittedSize: resp.GetCommittedSize(),
	}, nil
}
