package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// ExportFileStore keeps finished export files where every API instance can
// read them (T128). Production uses object storage (S3-compatible: R2, MinIO
// locally); the local-directory store is for development without storage and
// for tests only — on Cloud Run a local file is invisible to other instances
// and lost on scale-down.
type ExportFileStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// NewObjectExportStore stores export files in object storage.
func NewObjectExportStore(client storageclient.Client) ExportFileStore {
	return &objectExportStore{client: client}
}

type objectExportStore struct {
	client storageclient.Client
}

func (s *objectExportStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.UploadBytes(ctx, data, &storageclient.UploadInput{
		Key:         key,
		FileName:    filepath.Base(key),
		Size:        int64(len(data)),
		ContentType: contentType,
	})
	return err
}

func (s *objectExportStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.Download(ctx, &storageclient.DownloadInput{Key: key})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *objectExportStore) Delete(ctx context.Context, key string) error {
	return s.client.Delete(ctx, key)
}

// NewLocalExportStore stores export files under dir (development/tests).
func NewLocalExportStore(dir string) ExportFileStore {
	return &localExportStore{dir: dir}
}

type localExportStore struct {
	dir string
}

func (s *localExportStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid export key")
	}
	return filepath.Join(s.dir, clean), nil
}

func (s *localExportStore) Put(_ context.Context, key string, data []byte, _ string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (s *localExportStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *localExportStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
