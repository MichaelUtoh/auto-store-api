package storage

import (
	"context"
	"io"
)

// Storage uploads files and returns a public URL.
type Storage interface {
	// Upload uploads body to the given key and returns the public URL for the object.
	// key should be a path like "products/{id}/{filename}".
	Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) (publicURL string, err error)
}
