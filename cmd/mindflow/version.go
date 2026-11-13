package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func applicationVersion() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate application build: %w", err)
	}
	binary, err := os.Open(path) //nolint:gosec // G304: the path is the running executable, not request input.
	if err != nil {
		return "", fmt.Errorf("open application build: %w", err)
	}
	defer func() { _ = binary.Close() }()
	digest := sha256.New()
	_, err = io.Copy(digest, binary)
	if err != nil {
		return "", fmt.Errorf("identify application build: %w", err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}
