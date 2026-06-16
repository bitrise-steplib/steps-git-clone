package kv

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/v2/retry"
	"github.com/dustin/go-humanize"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/hash"
)

func (c *Client) UploadFileToBuildCache(ctx context.Context, filePath, key string) error {
	c.logger.Debugf("Uploading %s", filePath)

	checksum, err := hash.ChecksumOfFile(filePath)
	if err != nil {
		return fmt.Errorf("checksum of %s: %w", filePath, err)
	}

	fileSize, err := c.uploadFile(ctx, filePath, key, checksum)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	//nolint: gosec
	c.logger.Infof("(i) Uploaded: %s", humanize.Bytes(uint64(fileSize)))

	return nil
}

func (c *Client) UploadStreamToBuildCache(ctx context.Context, source io.ReadSeeker, key string, size int64) error {
	// Always seek to start before checksum
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek to start: %w", err)
	}
	checksum, err := hash.Checksum(source)
	if err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	// Seek to start before upload
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek to start: %w", err)
	}
	if err := c.uploadStream(ctx, source, key, checksum, size); err != nil {
		return fmt.Errorf("upload stream: %w", err)
	}

	return nil
}

func (c *Client) uploadFile(ctx context.Context, filePath, key, checksum string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open %q: %w", filePath, err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat %q: %w", filePath, err)
	}

	if err = c.uploadStream(ctx, file, key, checksum, stat.Size()); err != nil {
		return 0, fmt.Errorf("upload %q: %w", filePath, err)
	}

	return stat.Size(), nil
}

// nolint: gocognit
func (c *Client) uploadStream(ctx context.Context, source io.ReadSeeker, key, checksum string, size int64) error {
	const divisor = 10 * 1024 * 1024 // 10 MB

	// give each 10 MB a second, min 20s max 2m
	timeout := min(
		max(
			time.Duration(size/divisor)*time.Second,
			20*time.Second,
		),
		2*time.Minute,
	)

	lastCommittedSize := int64(0)
	hasAlreadyExists := false

	//nolint:wrapcheck
	return retry.Times(c.uploadRetry).Wait(c.uploadRetryWait).TryWithAbort(func(attempt uint) (error, bool) {
		if attempt == 0 {
			c.logger.TDebugf("Uploading %s (size: %d, timeout: %s)", key, size, timeout.String())
		} else {
			c.logger.TInfof("%d. attempt to upload %s (size: %d, previously uploaded: %d, timeout: %s)", attempt+1, key, size, lastCommittedSize, timeout.String())
		}

		if attempt > 0 {
			writeStatus, err := c.QueryWriteStatus(ctx, key)
			switch {
			case err != nil:
				if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
					// ignore not found errors
				} else {
					c.logger.Warnf("Failed to query write status for %s: %s", key, err)
				}
			case writeStatus.Complete:
				c.logger.Infof("Upload %s already complete, skipping", key)

				return nil, true
			default:
				lastCommittedSize = writeStatus.CommittedSize
				c.logger.Infof("Last committed %s by server: %d", key, lastCommittedSize)
			}
		}

		if lastCommittedSize >= size {
			c.logger.Infof("Already written %s by server", key)

			return nil, true
		}

		select {
		case <-ctx.Done():
			return ctx.Err(), true
		default:
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		kvWriter, err := c.initiatePut(timeoutCtx, PutParams{
			Name:            key,
			Sha256Sum:       checksum,
			FileSize:        size,
			Offset:          lastCommittedSize,
			DeleteOnRewrite: hasAlreadyExists,
		})
		if err != nil {
			c.logger.Warnf("Failed to upload stream %s: attempt %d: initiate put: %s", key, attempt+1, err)

			return fmt.Errorf("create kv put client (with key %s): %w", key, err), false
		}

		bytesSent := int64(0)
		if size > 0 {
			if _, err := source.Seek(lastCommittedSize, io.SeekStart); err != nil {
				return fmt.Errorf("seek source to start: %w", err), true
			}

			// io.Copy does not write if there was no read
			bytesSent, err = io.Copy(kvWriter, source)
		} else {
			// io.Copy does not write if there was no read
			_, err = kvWriter.Write([]byte{})
		}

		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			c.logger.Infof("blob %s already exists, deleting and restarting...", key)
			hasAlreadyExists = true
			lastCommittedSize = 0

			return nil, false
		}
		if ok && st.Code() == codes.Unauthenticated {
			return ErrCacheUnauthenticated, true
		}
		if err != nil {
			c.logger.TWarnf("Failed to upload stream %s: attempt %d: %s", key, attempt+1, err)

			return fmt.Errorf("upload archive: %w", err), false
		}

		if err := kvWriter.Close(); err != nil {
			c.logger.TWarnf("Failed to upload stream %s: attempt %d: %s", key, attempt+1, err)

			return fmt.Errorf("close upload: %w", err), false
		}

		if kvWriter.Response().GetCommittedSize() != bytesSent {
			return fmt.Errorf("uploaded size mismatch: expected %d, got %d", bytesSent, kvWriter.Response().GetCommittedSize()), false
		}

		if lastCommittedSize > 0 && attempt > 0 {
			c.logger.Infof("Upload %s success (size: %d) in %d attempts", key, size, attempt+1)
		}

		return nil, false
	})
}
