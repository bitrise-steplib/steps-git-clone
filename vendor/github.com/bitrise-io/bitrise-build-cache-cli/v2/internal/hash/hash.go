package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func ChecksumOfFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	return Checksum(file)
}

func Checksum(source io.Reader) (string, error) {
	hash := sha256.New()

	_, err := io.Copy(hash, source)
	if err != nil {
		return "", fmt.Errorf("reading file to hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
