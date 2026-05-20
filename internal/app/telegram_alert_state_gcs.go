package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

var (
	// ErrNilStorageClient is returned by [NewGCSTelegramAlertStateStore] when client is nil.
	ErrNilStorageClient = errors.New("app: storage client is required")
	// ErrEmptyTelegramAlertStateBucket is returned when bucket is empty.
	ErrEmptyTelegramAlertStateBucket = errors.New("app: TELEGRAM_ALERT_STATE_BUCKET is required")
	// ErrEmptyTelegramAlertStatePath is returned when object path is empty.
	ErrEmptyTelegramAlertStatePath = errors.New("app: TELEGRAM_ALERT_STATE_PATH is required")
)

// GCSTelegramAlertStateStore reads and writes the last-sent hour in a single GCS object.
type GCSTelegramAlertStateStore struct {
	client *storage.Client
	bucket string
	path   string
}

// NewGCSTelegramAlertStateStore returns a store backed by one object in bucket at path.
func NewGCSTelegramAlertStateStore(client *storage.Client, bucket, path string) (GCSTelegramAlertStateStore, error) {
	if client == nil {
		return GCSTelegramAlertStateStore{}, ErrNilStorageClient
	}
	if bucket == "" {
		return GCSTelegramAlertStateStore{}, ErrEmptyTelegramAlertStateBucket
	}
	if path == "" {
		return GCSTelegramAlertStateStore{}, ErrEmptyTelegramAlertStatePath
	}
	return GCSTelegramAlertStateStore{
		client: client,
		bucket: bucket,
		path:   path,
	}, nil
}

// ReadLastSentHour returns the stored hour, or "" if the object does not exist.
func (s GCSTelegramAlertStateStore) ReadLastSentHour(ctx context.Context) (string, error) {
	r, err := s.client.Bucket(s.bucket).Object(s.path).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("gcs read %s/%s: %w", s.bucket, s.path, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("gcs read %s/%s body: %w", s.bucket, s.path, err)
	}
	return normalizeStoredHour(data), nil
}

// WriteLastSentHour overwrites the object with the given UTC calendar hour.
func (s GCSTelegramAlertStateStore) WriteLastSentHour(ctx context.Context, hour string) error {
	w := s.client.Bucket(s.bucket).Object(s.path).NewWriter(ctx)
	w.ContentType = "text/plain; charset=utf-8"
	if _, err := io.WriteString(w, hour); err != nil {
		_ = w.Close()
		return fmt.Errorf("gcs write %s/%s body: %w", s.bucket, s.path, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("gcs write %s/%s: %w", s.bucket, s.path, err)
	}
	return nil
}
