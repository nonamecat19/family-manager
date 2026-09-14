package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Provider string

const (
	ProviderMinIO Provider = "minio"

	ProviderR2 Provider = "r2"
)

type Config struct {
	Provider  Provider
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	PublicURL string
}

type Client struct {
	mc        *minio.Client
	provider  Provider
	publicURL string
}

func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage: endpoint, access key and secret key are required")
	}

	provider := cfg.Provider
	if provider == "" {
		provider = ProviderMinIO
	}
	if provider != ProviderMinIO && provider != ProviderR2 {
		return nil, fmt.Errorf("storage: unknown provider %q (want %q or %q)", provider, ProviderMinIO, ProviderR2)
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	}
	if provider == ProviderR2 {
		opts.Region = "auto"
		opts.BucketLookup = minio.BucketLookupPath
	}

	mc, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("storage: new client: %w", err)
	}

	publicURL := strings.TrimSuffix(cfg.PublicURL, "/")
	if publicURL == "" {
		if provider == ProviderR2 {
			return nil, fmt.Errorf("storage: PublicURL is required for the %q provider: the R2 S3 endpoint does not serve public reads", ProviderR2)
		}
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)
	}

	return &Client{mc: mc, provider: provider, publicURL: publicURL}, nil
}

func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	if c.provider == ProviderR2 {
		return nil
	}

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

func (c *Client) Put(
	ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string,
) (string, error) {
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("storage: put object: %w", err)
	}
	return c.objectURL(bucket, key), nil
}

func (c *Client) objectURL(bucket, key string) string {
	if c.provider == ProviderR2 {
		return fmt.Sprintf("%s/%s", c.publicURL, key)
	}
	return fmt.Sprintf("%s/%s/%s", c.publicURL, bucket, key)
}
