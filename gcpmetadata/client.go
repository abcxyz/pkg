// Copyright 2024 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcpmetadata

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
)

const (
	// metadataIP is the documented metadata server IP address.
	metadataIP = "169.254.169.254"

	// metadataHostEnv is the environment variable specifying the GCE metadata
	// hostname; borrowed from the google-cloud-go package.
	metadataHostEnv = "GCE_METADATA_HOST"

	// userAgent is the HTTP user agent to use for HTTP calls.
	userAgent = "abcxyz:pkg/1.0 (+https://github.com/abcxyz/pkg)"
)

// Option is a configuration for the client.
type Option func(c *Client) *Client

// WithHost is an option that injects a custom metadata server host.
func WithHost(host string) Option {
	return func(c *Client) *Client {
		c.host = host
		return c
	}
}

// WithHTTPClient is an option that injects a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) *Client {
		c.httpClient = client
		return c
	}
}

// Client is an HTTP client for interacting with the Google Cloud metadata
// server. No results are cached and each invocation will result in HTTP
// requests.
type Client struct {
	host       string
	httpClient *http.Client
}

// NewClient creates a new HTTP metadata client for interacting with the Google Cloud
// metadata server.
func NewClient(opts ...Option) *Client {
	c := &Client{}

	for _, opt := range opts {
		if opt != nil {
			c = opt(c)
		}
	}

	// Default host
	if c.host == "" {
		c.host = os.Getenv(metadataHostEnv)
	}
	if c.host == "" {
		c.host = metadataIP
	}
	c.host = "http://" + c.host + "/computeMetadata/v1/"

	// Default httpClient
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				IdleConnTimeout: 60 * time.Second,
			},
			Timeout: 5 * time.Second,
		}
	}

	return c
}

// ProjectID returns the project ID from the metadata server.
func (c *Client) ProjectID(ctx context.Context) (string, error) {
	return c.Get(ctx, "project/project-id")
}

// ProjectNumber returns the project number from the metadata server.
func (c *Client) ProjectNumber(ctx context.Context) (string, error) {
	return c.Get(ctx, "project/numeric-project-id")
}

// Get fetches the metadata server response at the given path.
func (c *Client) Get(ctx context.Context, pth string) (string, error) {
	u := c.host + strings.TrimLeft(pth, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")
	req.Header.Set("User-Agent", userAgent)

	b := retry.NewFibonacci(50 * time.Millisecond)
	b = retry.WithCappedDuration(5*time.Second, b)
	b = retry.WithMaxDuration(30*time.Second, b)

	var bodyStr string
	if err := retry.Do(ctx, b, func(ctx context.Context) error {
		var err error
		resp, err := c.httpClient.Do(req)

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}

		// Handle retries
		if shouldRetry(statusCode, err) {
			return retry.RetryableError(err)
		}

		// 404 are immediate errors
		if statusCode == http.StatusNotFound {
			return fmt.Errorf("metadata does not exist for %q", pth)
		}

		// Read the entire response.
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2mb
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		bodyStr = string(bytes.TrimSpace(body))

		// Ensure we got a 200 response
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-200 response: %s", bodyStr)
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to get metadata: %w", err)
	}

	return bodyStr, nil
}

// temporaryError is an interface for an error that declares itself as
// temporary. Some of the standard library errors do this.
type temporaryError interface {
	Temporary() bool
}

// unwrappableError is an error that wraps another error.
type unwrappableError interface {
	Unwrap() error
}

// unwrappableErrors is an error that wraps multiple other errors.
type unwrappableErrors interface {
	Unwrap() []error
}

func shouldRetry(status int, err error) bool {
	// Do not retry success, that's weird.
	if status == http.StatusOK {
		return false
	}

	// Retry server-side errors.
	if status >= 500 && status <= 599 {
		return true
	}

	// Retry on EOF.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Retry temporary errors.
	var terr temporaryError
	if ok := errors.As(err, &terr); ok && terr.Temporary() {
		return true
	}

	// If this is a wrapped error, do everything above for the inner error(s).
	var uerr unwrappableError
	if ok := errors.As(err, &uerr); ok {
		return shouldRetry(status, uerr.Unwrap())
	}

	var uerrs unwrappableErrors
	if ok := errors.As(err, &uerrs); ok {
		for _, err := range uerrs.Unwrap() {
			if shouldRetry(status, err) {
				return true
			}
		}
	}

	// If we got this far, don't retry.
	return false
}
