package filegroup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/dustin/go-humanize"
	"github.com/pkg/xattr"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/hash"
)

type Info struct {
	Files       []*FileInfo      `json:"files"`
	Directories []*DirectoryInfo `json:"directories"`
	Symlinks    []*SymlinkInfo   `json:"symlinks,omitempty"`
}

type DirectoryInfo struct {
	Path    string    `json:"path"`
	ModTime time.Time `json:"modTime"`
}

type SymlinkInfo struct {
	Path    string    `json:"path"`
	Target  string    `json:"target"`
	ModTime time.Time `json:"modTime"`
}

type FileInfo struct {
	Path       string            `json:"path"`
	Size       int64             `json:"size"`
	Hash       string            `json:"hash"`
	ModTime    time.Time         `json:"modTime"`
	Mode       os.FileMode       `json:"mode"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type fileGroupInfoCollector struct {
	Files           []*FileInfo
	Dirs            []*DirectoryInfo
	Symlinks        []*SymlinkInfo
	LargestFileSize int64
	seen            map[string]bool

	mu sync.Mutex
}

func (mc *fileGroupInfoCollector) AddFile(fileInfo *FileInfo) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.Files = append(mc.Files, fileInfo)
	if fileInfo.Size > mc.LargestFileSize {
		mc.LargestFileSize = fileInfo.Size
	}
	mc.seen[fileInfo.Path] = true
}

func (mc *fileGroupInfoCollector) AddDir(dirInfo *DirectoryInfo) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.Dirs = append(mc.Dirs, dirInfo)
	mc.seen[dirInfo.Path] = true
}

func (mc *fileGroupInfoCollector) AddSymlink(symlink *SymlinkInfo) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.Symlinks = append(mc.Symlinks, symlink)
	mc.seen[symlink.Path] = true
}

func (mc *fileGroupInfoCollector) isSeen(path string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return mc.seen[path]
}

func CollectFileGroupInfo(cacheDirPath string,
	collectAttributes,
	followSymlinks bool,
	skipSPM bool,
	logger log.Logger,
) (Info, error) {
	var dd Info

	fgi := fileGroupInfoCollector{
		Files:    make([]*FileInfo, 0),
		Dirs:     make([]*DirectoryInfo, 0),
		Symlinks: make([]*SymlinkInfo, 0),
		seen:     make(map[string]bool),
	}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit parallelization

	err := filepath.WalkDir(cacheDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		wg.Add(1)
		semaphore <- struct{}{} // Block if there are too many goroutines are running

		go func(d fs.DirEntry) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release a slot in the semaphore

			inf, err := d.Info()
			if err != nil {
				logger.Errorf("get file info: %v", err)

				return
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(cacheDirPath, path)
			}
			if err := CollectFileMetadata(cacheDirPath, path, inf, inf.IsDir(), &fgi, collectAttributes, followSymlinks, skipSPM, logger); err != nil {
				logger.Errorf("Failed to collect metadata: %s", err)
			}
		}(d)

		return nil
	})

	wg.Wait()

	if err != nil {
		return Info{}, fmt.Errorf("walk dir: %w", err)
	}

	dd.Files = fgi.Files
	dd.Directories = fgi.Dirs
	dd.Symlinks = fgi.Symlinks

	logger.Infof("(i) Collected %d files and %d directories ", len(dd.Files), len(dd.Directories))
	//nolint: gosec
	logger.Debugf("(i) Largest processed file size: %s", humanize.Bytes(uint64(fgi.LargestFileSize)))

	return dd, nil
}

// nolint:wrapcheck
func followSymlink(rootPath, path, target string,
	fgi *fileGroupInfoCollector,
	followSymlinks,
	skipSPM bool,
	logger log.Logger,
) error {
	if !followSymlinks {
		logger.Debugf("Skipping symbolic link: %s", path)

		return nil
	}
	if fgi.isSeen(target) {
		logger.Debugf("Skipping symbolic link target: %s, already seen", target)

		return nil
	}

	// Dont save symlink if target doesn't exist
	stat, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}

	logger.Debugf("Resolved symlink %s to target: %s", path, target)

	fgi.AddSymlink(&SymlinkInfo{
		Path:    path,
		Target:  target,
		ModTime: stat.ModTime(),
	})

	if !stat.IsDir() {
		return CollectFileMetadata(rootPath, target, stat, false, fgi, false, followSymlinks, skipSPM, logger)
	}

	logger.Debugf("Symlink target is a directory, walking it: %s", target)
	// Recursively walk the target directory, as it will not be included in this walk
	return filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk target dir: %w", err)
		}

		inf, err := d.Info()
		if err != nil {
			return fmt.Errorf("get file info: %w", err)
		}

		if !filepath.IsAbs(path) {
			path = filepath.Join(target, path)
		}

		return CollectFileMetadata(target, path, inf, inf.IsDir(), fgi, false, followSymlinks, skipSPM, logger)
	})
}

func CollectFileMetadata(
	rootPath, path string,
	fileInfo fs.FileInfo,
	isDirectory bool,
	fgi *fileGroupInfoCollector,
	collectAttributes, followSymlinks, skipSPM bool,
	logger log.Logger,
) error {
	if fgi.isSeen(path) {
		logger.Debugf("Skipping path %s, already seen", path)

		return nil
	}

	if isDirectory {
		fgi.AddDir(&DirectoryInfo{
			Path:    path,
			ModTime: fileInfo.ModTime(),
		})

		return nil
	}

	if skipSPM {
		parts := strings.Split(filepath.ToSlash(path), "/")
		for _, part := range parts {
			if part == "SourcePackages" {
				return nil
			}
		}
	}

	isSymlink := fileInfo.Mode()&os.ModeSymlink != 0

	if isSymlink {
		var target string
		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("read symlink: %w", err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}

		return followSymlink(rootPath, path, target, fgi, followSymlinks, skipSPM, logger)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash copy file content: %w", err)
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	var attrs map[string]string
	if collectAttributes {
		attrs, err = getAttributes(path)
		if err != nil {
			return fmt.Errorf("getting attributes: %w", err)
		}
	}

	fgi.AddFile(&FileInfo{
		Path:       path,
		Size:       fileInfo.Size(),
		Hash:       hash,
		ModTime:    fileInfo.ModTime(),
		Mode:       fileInfo.Mode(),
		Attributes: attrs,
	})

	return nil
}

func getAttributes(path string) (map[string]string, error) {
	attributes := make(map[string]string)
	attrNames, err := xattr.List(path)
	if err != nil {
		return nil, fmt.Errorf("list attributes: %w", err)
	}

	for _, attr := range attrNames {
		value, err := xattr.Get(path, attr)
		if err != nil {
			return nil, fmt.Errorf("xattr get: %w", err)
		}
		attributes[attr] = string(value)
	}

	return attributes, nil
}

func SetAttributes(path string, attributes map[string]string) error {
	for attr, value := range attributes {
		if err := xattr.Set(path, attr, []byte(value)); err != nil {
			return fmt.Errorf("xattr set: %w", err)
		}
	}

	return nil
}

func RestoreSymlink(symlink SymlinkInfo, logger log.Logger) bool {
	fileInfo, err := os.Lstat(symlink.Path)
	if err == nil && fileInfo.Mode()&os.ModeSymlink != 0 {
		logger.Debugf("Symlink %s already exists, overwriting...", symlink.Path)

		err := os.Remove(symlink.Path)
		if err != nil {
			logger.Infof("Error removing existing symlink %s: %v", symlink.Path, err)

			return false
		}
	}

	err = os.Symlink(symlink.Target, symlink.Path)
	if err != nil {
		logger.Debugf("Error creating symlink %s -> %s: %v", symlink.Path, symlink.Target, err)

		return false
	}

	// Set times
	mtimeSpec := syscall.NsecToTimespec(symlink.ModTime.UnixNano())
	err = syscall.UtimesNano(symlink.Path, []syscall.Timespec{mtimeSpec, mtimeSpec})
	if err != nil {
		logger.Debugf("Error setting symlink times for %s: %v", symlink.Path, err)

		return false
	}

	return true
}

func RestoreFileInfo(fi FileInfo, rootDir string, logger log.Logger) bool {
	var path string
	if filepath.IsAbs(fi.Path) || rootDir == "" {
		path = fi.Path
	} else {
		path = filepath.Join(rootDir, fi.Path)
	}

	// Skip if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Debugf("File %s doesn't exist", fi.Path)

		return false
	}

	h, err := hash.ChecksumOfFile(path)
	if err != nil {
		logger.Infof("Error hashing file %s: %v", fi.Path, err)

		return false
	}

	if h != fi.Hash {
		return false
	}

	if err := os.Chtimes(path, fi.ModTime, fi.ModTime); err != nil {
		logger.Debugf("Error setting modification time for %s: %v", fi.Path, err)

		return false
	}

	if err = os.Chmod(fi.Path, fi.Mode); err != nil {
		logger.Debugf("Error setting file mode for %s: %v", fi.Path, err)

		return false
	}

	if len(fi.Attributes) > 0 {
		err = SetAttributes(fi.Path, fi.Attributes)
		if err != nil {
			logger.Debugf("Error setting file attributes for %s: %v", fi.Path, err)

			return false
		}
	}

	return true
}

func RestoreDirectoryInfo(dir DirectoryInfo, rootDir string) error {
	var path string
	if filepath.IsAbs(dir.Path) {
		path = dir.Path
	} else {
		path = filepath.Join(rootDir, dir.Path)
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.Chtimes(path, dir.ModTime, dir.ModTime); err != nil {
		return fmt.Errorf("set directory mod time: %w", err)
	}

	return nil
}
