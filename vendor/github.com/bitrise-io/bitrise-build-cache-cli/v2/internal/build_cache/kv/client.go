package kv

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	remoteexecution "github.com/bitrise-io/bitrise-build-cache-cli/v2/proto/build/bazel/remote/execution/v2"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/proto/kv_storage"
)

//go:generate moq -rm -stub -pkg mocks -out ./mocks/kv_storage.go ./../../../proto/kv_storage KVStorageClient

type Client struct {
	bitriseKVClient     kv_storage.KVStorageClient
	capabilitiesClient  remoteexecution.CapabilitiesClient
	casClient           remoteexecution.ContentAddressableStorageClient
	clientName          string
	authConfig          common.CacheAuthConfig
	cacheConfigMetadata common.CacheConfigMetadata
	logger              log.Logger
	cacheOperationID    string
	invocationID        string
	sessionMutex        sync.Mutex
	downloadRetry       uint
	downloadRetryWait   time.Duration
	uploadRetry         uint
	uploadRetryWait     time.Duration
}

type NewClientParams struct {
	UseInsecure         bool
	Host                string
	DialTimeout         time.Duration
	ClientName          string
	AuthConfig          common.CacheAuthConfig
	CacheConfigMetadata common.CacheConfigMetadata
	Logger              log.Logger
	CacheOperationID    string
	BitriseKVClient     kv_storage.KVStorageClient
	CapabilitiesClient  remoteexecution.CapabilitiesClient
	InvocationID        string
	DownloadRetry       uint
	DownloadRetryWait   time.Duration
	UploadRetry         uint
	UploadRetryWait     time.Duration
}

func NewClient(p NewClientParams) (*Client, error) {
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	if p.UseInsecure {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(p.Host, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", p.Host, err)
	}

	bitriseKVClient := p.BitriseKVClient
	if bitriseKVClient == nil {
		bitriseKVClient = kv_storage.NewKVStorageClient(conn)
	}
	capabilitiesClient := p.CapabilitiesClient
	if capabilitiesClient == nil {
		capabilitiesClient = remoteexecution.NewCapabilitiesClient(conn)
	}

	if p.DownloadRetry == 0 {
		p.DownloadRetry = 3
	}
	if p.DownloadRetryWait == 0 {
		p.DownloadRetryWait = 1 * time.Second
	}
	if p.UploadRetry == 0 {
		p.UploadRetry = 3
	}
	if p.UploadRetryWait == 0 {
		p.UploadRetryWait = 1 * time.Second
	}

	return &Client{
		bitriseKVClient:     bitriseKVClient,
		capabilitiesClient:  capabilitiesClient,
		casClient:           remoteexecution.NewContentAddressableStorageClient(conn),
		clientName:          p.ClientName,
		authConfig:          p.AuthConfig,
		logger:              p.Logger,
		cacheConfigMetadata: p.CacheConfigMetadata,
		cacheOperationID:    p.CacheOperationID,
		invocationID:        p.InvocationID,
		downloadRetry:       p.DownloadRetry,
		downloadRetryWait:   p.DownloadRetryWait,
		uploadRetry:         p.UploadRetry,
		uploadRetryWait:     p.UploadRetryWait,
	}, nil
}

func (c *Client) SetLogger(logger log.Logger) {
	c.logger = logger
}

type writer struct {
	stream       bytestream.ByteStream_WriteClient
	resourceName string
	offset       int64
	fileSize     int64
	response     *bytestream.WriteResponse
}

func (w *writer) Response() *bytestream.WriteResponse {
	return w.response
}

func (w *writer) Write(p []byte) (int, error) {
	req := &bytestream.WriteRequest{
		ResourceName: w.resourceName,
		WriteOffset:  w.offset,
		Data:         p,
		FinishWrite:  w.offset+int64(len(p)) >= w.fileSize,
	}
	err := w.stream.Send(req)
	switch {
	case errors.Is(err, io.EOF):
		return 0, io.EOF
	case err != nil:
		return 0, fmt.Errorf("send data: %w", err)
	}
	w.offset += int64(len(p))

	return len(p), nil
}

func (w *writer) Close() error {
	var err error
	w.response, err = w.stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close stream: %w", err)
	}

	return nil
}

type reader struct {
	logger   log.Logger
	stream   bytestream.ByteStream_ReadClient
	metadata sync.Map
	buf      bytes.Buffer

	metadataReady chan struct{}
}

func (r *reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	bufLen := r.buf.Len()
	if bufLen > 0 {
		n, _ := r.buf.Read(p) // this will never fail

		return n, nil
	}
	r.buf.Reset()

	resp, err := r.stream.Recv()
	switch {
	case errors.Is(err, io.EOF):
		r.readTrailerMetadata()

		return 0, io.EOF
	case err != nil:
		r.readTrailerMetadata()

		return 0, fmt.Errorf("stream receive: %w", err)
	}

	n := copy(p, resp.GetData())
	if n == len(resp.GetData()) {
		return n, nil
	}

	unwrittenData := resp.GetData()[n:]
	_, _ = r.buf.Write(unwrittenData) // this will never fail

	return n, nil
}

func (r *reader) readStreamMetadata() {
	if header, err := r.stream.Header(); err == nil {
		for k, v := range header {
			if len(v) > 0 {
				r.metadata.Store(k, v[0])
			}
		}
	} else {
		r.logger.Errorf("Failed to read stream header: %v", err)
	}

	go func() {
		close(r.metadataReady)
	}()
}

func (r *reader) readTrailerMetadata() {
	if trailer := r.stream.Trailer(); trailer != nil {
		for k, v := range trailer {
			if len(v) > 0 {
				r.metadata.Store(k, v[0])
			}
		}
	}
}

func (r *reader) Metadata() map[string]string {
	<-r.metadataReady
	m := make(map[string]string)
	r.metadata.Range(func(key, value any) bool {
		k, ok1 := key.(string)
		v, ok2 := value.(string)
		if ok1 && ok2 {
			m[k] = v
		}

		return true
	})

	return m
}

func (r *reader) Close() error {
	r.buf.Reset()

	return nil
}
