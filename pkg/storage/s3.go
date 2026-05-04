package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config is the minimal config needed for S3 uploads.
type S3Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional; for MinIO, DigitalOcean Spaces, etc. (where the SDK sends requests)
	// PublicURL is optional. If set, Upload() returns URLs using this base (for browsers/clients).
	// Use when Endpoint is host.docker.internal (Docker → host MinIO) but clients must use localhost.
	PublicURL string
}

// S3Storage implements Storage using AWS S3 or an S3-compatible endpoint.
// PutObject never sends object ACLs (AWS buckets with "ACLs disabled" reject x-amz-acl).
// Use bucket policy, CloudFront, or presigned URLs for read access.
type S3Storage struct {
	client  *s3.Client
	bucket  string
	region  string
	baseURL string // base for public URLs (e.g. https://bucket.s3.region.amazonaws.com)
}

// NewS3 returns an S3 storage client, or nil if bucket is not set.
func NewS3(cfg S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, nil
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.Region = region
		},
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, func(o *s3.Options) {
			o.Credentials = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
		})
	}
	if cfg.Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(strings.TrimSuffix(cfg.Endpoint, "/"))
			o.UsePathStyle = true
		})
	}
	client := s3.New(s3.Options{}, opts...)
	baseURL := buildBaseURL(cfg)
	if cfg.PublicURL != "" {
		baseURL = strings.TrimSuffix(cfg.PublicURL, "/")
	}
	return &S3Storage{
		client:  client,
		bucket:  cfg.Bucket,
		region:  region,
		baseURL: baseURL,
	}, nil
}

func buildBaseURL(cfg S3Config) string {
	if cfg.Endpoint != "" {
		// S3-compatible (MinIO, Spaces): base is endpoint + bucket
		e := strings.TrimSuffix(cfg.Endpoint, "/")
		return e + "/" + cfg.Bucket
	}
	// AWS: https://bucket.s3.region.amazonaws.com
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
}

// Upload uploads body to S3 at key and returns the public URL.
func (s *S3Storage) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", err
	}
	// Escape each path segment for the public URL
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	publicURL := s.baseURL + "/" + strings.Join(segments, "/")
	return publicURL, nil
}
