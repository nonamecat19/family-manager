package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/nnc/notes-manager-backend/internal/config"
)

const vercelBlobBaseURL = "https://blob.vercel-storage.com"
const vercelBlobAPIVersion = "7"

type VercelBlobStorage struct {
	token string
}

type blobUploadResponse struct {
	URL string `json:"url"`
}

func NewVercelBlobStorage(cfg config.VercelBlobConfig) *VercelBlobStorage {
	return &VercelBlobStorage{token: cfg.Token}
}

func (s *VercelBlobStorage) Upload(ctx context.Context, reader io.Reader, contentType string, size int64) (string, error) {
	pathname := uuid.NewString()
	endpoint := fmt.Sprintf("%s/%s", vercelBlobBaseURL, pathname)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, reader)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("x-api-version", vercelBlobAPIVersion)
	req.Header.Set("Content-Type", contentType)
	if size > 0 {
		req.ContentLength = size
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, body)
	}

	var result blobUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.URL, nil
}

func (s *VercelBlobStorage) GetURL(_ context.Context, key string) (string, error) {
	// The key is already the public URL returned by Vercel Blob on upload.
	return key, nil
}

func (s *VercelBlobStorage) Delete(ctx context.Context, key string) error {
	body, err := json.Marshal(map[string][]string{"urls": {key}})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, vercelBlobBaseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("x-api-version", vercelBlobAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, b)
	}
	return nil
}
