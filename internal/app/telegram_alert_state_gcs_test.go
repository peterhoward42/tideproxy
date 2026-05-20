package app

import (
	"testing"

	"cloud.google.com/go/storage"
)

func TestNewGCSTelegramAlertStateStore_validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  *storage.Client
		bucket  string
		path    string
		wantErr error
	}{
		{name: "nil client", bucket: "b", path: "p", wantErr: ErrNilStorageClient},
		{name: "empty bucket", client: &storage.Client{}, bucket: "", path: "p", wantErr: ErrEmptyTelegramAlertStateBucket},
		{name: "empty path", client: &storage.Client{}, bucket: "b", path: "", wantErr: ErrEmptyTelegramAlertStatePath},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewGCSTelegramAlertStateStore(tt.client, tt.bucket, tt.path)
			if err == nil {
				t.Fatal("NewGCSTelegramAlertStateStore: got nil want error")
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Fatalf("NewGCSTelegramAlertStateStore: got %v want %v", err, tt.wantErr)
			}
		})
	}
}
