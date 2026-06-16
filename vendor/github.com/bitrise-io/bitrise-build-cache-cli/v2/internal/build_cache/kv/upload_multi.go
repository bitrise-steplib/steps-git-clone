package kv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bitrise-io/go-utils/v2/retry"
	"github.com/dustin/go-humanize"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/filegroup"
)

type UploadFilesStats struct {
	FilesToUpload       int
	FilesUploaded       int
	FilesFailedToUpload int
	TotalFiles          int
	UploadSize          int64
	LargestFileSize     int64
}

func (c *Client) uploadFileToBuildCache(ctx context.Context, file *filegroup.FileInfo, mutex *sync.Mutex, stats *UploadFilesStats) {
	const retries = 2
	err := retry.Times(retries).Wait(3 * time.Second).TryWithAbort(func(attempt uint) (error, bool) {
		if attempt > 0 {
			c.logger.Debugf("Retrying upload... (attempt %d)", attempt)
		}

		_, err := c.uploadFile(ctx, file.Path, file.Hash, file.Hash)
		if err != nil {
			c.logger.Errorf("Error in upload file attempt %d: %s", attempt, err)
			if errors.Is(err, ErrCacheUnauthenticated) {
				return ErrCacheUnauthenticated, true
			}

			return fmt.Errorf("upload file %s: %w", file.Path, err), false
		}

		return nil, false
	})

	mutex.Lock()
	if err != nil {
		c.logger.Errorf("Failed to upload file %s with error: %v", file.Path, err)
		stats.FilesFailedToUpload++
	} else {
		stats.FilesUploaded++
		stats.UploadSize += file.Size
		if file.Size > stats.LargestFileSize {
			stats.LargestFileSize = file.Size
		}
	}
	mutex.Unlock()
}

func (c *Client) UploadFileGroupToBuildCache(ctx context.Context, dd filegroup.Info) (UploadFilesStats, error) {
	missingBlobs, err := c.findMissingBlobs(ctx, dd)
	if err != nil {
		return UploadFilesStats{}, fmt.Errorf("failed to check for missing blobs: %w", err)
	}

	stats := UploadFilesStats{
		TotalFiles:    len(dd.Files),
		FilesToUpload: len(missingBlobs),
	}

	c.logger.TInfof("(i) Uploading missing blobs...")

	var wg sync.WaitGroup
	var mutex sync.Mutex
	semaphore := make(chan struct{}, 20) // Limit parallelization
	for _, file := range dd.Files {
		mutex.Lock()
		_, ok := missingBlobs[file.Hash]
		delete(missingBlobs, file.Hash) // Remove the blob from the list of missing blobs as it's being uploaded
		mutex.Unlock()
		if !ok {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Block if there are too many goroutines are running

		go func(file *filegroup.FileInfo) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release a slot in the semaphore

			c.uploadFileToBuildCache(ctx, file, &mutex, &stats)
		}(file)
	}

	wg.Wait()

	//nolint: gosec
	c.logger.TInfof("(i) Uploaded %s in %d keys", humanize.Bytes(uint64(stats.UploadSize)), stats.FilesUploaded)

	if stats.FilesFailedToUpload > 0 {
		return stats, fmt.Errorf("failed to upload some files")
	}

	return stats, nil
}

func (c *Client) findMissingBlobs(ctx context.Context, dd filegroup.Info) (map[string]bool, error) {
	c.logger.TInfof("(i) Checking for missing blobs in the cache of %d files", len(dd.Files))

	blobs := make(map[string]bool)

	allDigests := make([]*FileDigest, 0, len(dd.Files))
	for _, file := range dd.Files {
		if _, ok := blobs[file.Hash]; !ok {
			allDigests = append(allDigests, &FileDigest{
				Sha256Sum:   file.Hash,
				SizeInBytes: file.Size,
			})

			blobs[file.Hash] = true
		}
	}

	c.logger.Infof("(i) The files are stored in %d different blobs", len(allDigests))
	missingDigests, err := c.FindMissing(ctx, allDigests)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing files in the cache: %w", err)
	}

	missingBlobs := make(map[string]bool)
	for _, d := range missingDigests {
		missingBlobs[d.Sha256Sum] = true
	}

	c.logger.TInfof("(i) %d of %d blobs are missing", len(missingBlobs), len(blobs))

	return missingBlobs, nil
}
