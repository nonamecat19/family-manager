// Package storage is the S3-compatible object storage client shared by every Go service that
// needs to hold user-uploaded files (currently just recipe images). Services never build
// their own client.
//
// Two providers are supported: self-hosted MinIO in development and Cloudflare R2 in
// production. They are not interchangeable at runtime — R2 omits S3 APIs MinIO implements and
// serves objects from a different URL shape — so the provider is an explicit config field
// rather than something guessed from the endpoint.
package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Provider selects which object store the client is talking to.
type Provider string

const (
	// ProviderMinIO is a self-hosted MinIO, the local dev backend. The service owns the
	// bucket: EnsureBucket creates it and sets an anonymous-read policy, and MinIO serves
	// the objects itself at <PublicURL>/<bucket>/<key>.
	ProviderMinIO Provider = "minio"

	// ProviderR2 is Cloudflare R2, the production backend. The bucket and the public
	// hostname in front of it are provisioned out of band, and the service's token is
	// scoped to object read/write — so EnsureBucket must neither create the bucket nor set
	// a policy on it: R2 does not implement PutBucketPolicy at all, and a scoped token
	// cannot CreateBucket. The public hostname is bound to one bucket, so objects are at
	// <PublicURL>/<key> with no bucket segment.
	ProviderR2 Provider = "r2"
)

// Config describes an object store. Endpoint and credentials are always required — a client
// with a blank endpoint or empty credentials must fail at construction, not on first upload.
type Config struct {
	// Provider defaults to ProviderMinIO when empty.
	Provider  Provider
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// PublicURL is the base URL clients (the app, a browser) use to fetch objects — always
	// different from Endpoint, which is the address the service itself uses to reach the
	// S3 API ("minio:9000" in dev, "<account>.r2.cloudflarestorage.com" in prod). Optional
	// for MinIO, where it falls back to Endpoint. Required for R2, whose S3 endpoint
	// serves no public reads at all.
	PublicURL string
}

// Client wraps a minio.Client with the one operation this repo needs: upload-and-serve.
// Objects are publicly readable — via a bucket policy on MinIO, via a public custom domain
// on R2 — rather than presigned per object: recipe images aren't sensitive, and public reads
// mean the app renders an <Image> straight off image_url with no signing round trip.
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
		// R2 has one global region named "auto" and does not answer GetBucketLocation;
		// without this minio-go probes for a region and the probe fails. The S3 endpoint
		// is account-scoped rather than a per-bucket virtual host, so addressing is
		// path-style.
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

// EnsureBucket creates the bucket if missing and sets an anonymous-read policy on it. Safe to
// call on every boot — both operations are idempotent.
//
// On R2 it is a deliberate no-op. The bucket is provisioned out of band and the token is
// scoped to objects, so there is nothing to create and nothing safe to probe: a scoped token
// can be denied HeadBucket on a bucket that is perfectly healthy, which would turn a correct
// deployment into a boot failure. A genuinely missing bucket surfaces on the first upload,
// which the caller already degrades on.
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

// Put uploads an object and returns the public URL it will be reachable at.
func (c *Client) Put(
	ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string,
) (string, error) {
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("storage: put object: %w", err)
	}
	return c.objectURL(bucket, key), nil
}

// objectURL is where a stored object is publicly readable. The two providers differ: MinIO
// serves every bucket off one host, so the bucket is a path segment; an R2 public domain is
// bound to a single bucket, so repeating it would 404.
func (c *Client) objectURL(bucket, key string) string {
	if c.provider == ProviderR2 {
		return fmt.Sprintf("%s/%s", c.publicURL, key)
	}
	return fmt.Sprintf("%s/%s/%s", c.publicURL, bucket, key)
}
