// Copyright 2023 The Authors (see AUTHORS file)
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

// Package reflectionclient wraps other libraries to make client-less grpc calls easily.
package reflectionclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client ingests config and creates persistent dialed connection.
type Client struct {
	config *ClientConfig
	conn   *grpc.ClientConn
	ts     oauth2.TokenSource
}

// ClientConfig host requires address:port format, ssl option, and request timeout.
type ClientConfig struct {
	Host     string
	Insecure bool
	Audience string
	Timeout  time.Duration
}

// NewClient creates and returns a reflection client.
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	var d grpc.DialOption
	var ts oauth2.TokenSource

	if !config.Insecure {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("x509.SystemCertPool failed: %w", err)
		}
		if certPool == nil {
			return nil, fmt.Errorf("x509.SystemCertPool failed: nil certpool")
		}
		c := &tls.Config{RootCAs: certPool, MinVersion: tls.VersionTLS12}
		tlsCreds := credentials.NewTLS(c)
		d = grpc.WithTransportCredentials(tlsCreds)

		// https://pkg.go.dev/golang.org/x/oauth2/google#FindDefaultCredentialsWithParams
		// In a non-GCP environment, we need to place the JSON key to where
		// GOOGLE_APPLICATION_CREDENTIALS env var is pointed to.
		cred, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate default credentials: %w", err)
		}

		aud := config.Audience
		if aud == "" {
			aud = config.Host
		}
		t, err := idtoken.NewTokenSource(ctx, aud, option.WithCredentials(cred))
		if err != nil {
			return nil, fmt.Errorf("failed to create NewTokenSource: %w", err)
		}
		ts = t
	} else {
		d = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.DialContext(ctx, config.Host, d)
	if err != nil {
		return nil, fmt.Errorf("grpc.DialContext failed: %w", err)
	}
	return &Client{
		conn:   conn,
		config: config,
		ts:     ts,
	}, nil
}

// CallMethod provides a payload to specified method, and returns the results as txt or json.
func (c *Client) CallMethod(ctx context.Context, method, payload string, textFormat bool) (*bytes.Buffer, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var headers []string
	// If the token source is present, generate an ID token to authenticate with
	// the server.
	if c.ts != nil {
		t, err := c.ts.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID token: %w", err)
		}
		headers = append(headers, "authorization: Bearer "+t.AccessToken)
	}

	refCtx := metadata.NewOutgoingContext(ctx, grpcurl.MetadataFromHeaders(headers))
	cli := grpcreflect.NewClientAuto(refCtx, c.conn)
	defer cli.Reset()

	descSource := grpcurl.DescriptorSourceFromServer(ctx, cli)
	in := strings.NewReader(payload)
	options := grpcurl.FormatOptions{IncludeTextSeparator: true}
	format := "text"
	if !textFormat {
		format = "json"
	}
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.Format(format), descSource, in, options)
	if err != nil {
		return nil, fmt.Errorf("grpcurl.RequestParserAndFormatter failed: %w", err)
	}

	o := new(bytes.Buffer)
	h := &grpcurl.DefaultEventHandler{Out: o, Formatter: formatter}

	if err := grpcurl.InvokeRPC(timeoutCtx, descSource, c.conn, method, headers, h, rf.Next); err != nil {
		return nil, fmt.Errorf("grpcurl.InvokeRPC(%q) failed: %w", method, err)
	}
	if h.Status != nil && h.Status.Err() != nil {
		return nil, fmt.Errorf("grpcurl.DefaultEventHandler error: %w", h.Status.Err())
	}
	return o, nil
}

// Close the connection on the client.
func (c *Client) Close() {
	c.conn.Close()
}
