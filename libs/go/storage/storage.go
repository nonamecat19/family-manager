// Package storage is the MinIO (S3-compatible) object storage client shared by every Go
// service that needs to hold user-uploaded files (currently just recipe images). Services
// never build their own MinIO client.
package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config describes a MinIO endpoint. All fields are required — a client with a blank
// endpoint or empty credentials must fail at construction, not on the first upload.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// PublicURL is the base URL clients (the app, a browser) use to fetch objects —
	// typically different from Endpoint, which is the in-network address the service
	// itself uses to talk to MinIO (e.g. "minio:9000" vs a public host behind Caddy).
	PublicURL string
}

// Client wraps a minio.Client with the one operation this repo needs: upload-and-serve.
// Objects are made publicly readable at the bucket level (see EnsureBucket) rather than
// per-object or via presigned URLs — recipe images aren't sensitive, and a public bucket
// means the app can render an <Image> straight off image_url with no signing round trip.
type Client struct {
	mc        *minio.Client
	publicURL string
}

func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage: endpoint, access key and secret key are required")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: new client: %w", err)
	}
	publicURL := cfg.PublicURL
	if publicURL == "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)
	}
	return &Client{mc: mc, publicURL: strings.TrimSuffix(publicURL, "/")}, nil
}

// EnsureBucket creates the bucket if missing and sets an anonymous-read policy on it. Safe
// to call on every boot — both operations are idempotent.
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("storage: bucket exists: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("storage: make bucket: %w", err)
		}
	}
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"AWS": ["*"]},
			"Action": ["s3:GetObject"],
			"Resource": ["arn:aws:s3:::%s/*"]
		}]
	}`, bucket)
	if err := c.mc.SetBucketPolicy(ctx, bucket, policy); err != nil {
		return fmt.Errorf("storage: set bucket policy: %w", err)
	}
	return nil
}

// Put uploads an object and returns the public URL it will be reachable at.
func (c *Client) Put(
	ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string,
) (string, error) {
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("storage: put object: %w", err)
	}
	return fmt.Sprintf("%s/%s/%s", c.publicURL, bucket, key), nil
}
