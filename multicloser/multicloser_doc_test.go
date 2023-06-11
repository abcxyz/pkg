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

package multicloser_test

import (
	"fmt"
	"log"

	"github.com/abcxyz/pkg/multicloser"
)

func setup() (*multicloser.Closer, error) {
	var closer *multicloser.Closer

	client1, err := newClient()
	if err != nil {
		return closer, fmt.Errorf("failed to create client1: %w", err)
	}
	closer = multicloser.Append(closer, client1.Close)

	client2, err := newClient()
	if err != nil {
		return closer, fmt.Errorf("failed to create client2: %w", err)
	}
	closer = multicloser.Append(closer, client2.Close)

	return closer, nil
}

// client is just a stub to demonstrate something that needs to be closed.
type client struct{}

func (c *client) Close() error {
	return nil
}

func newClient() (*client, error) {
	return &client{}, nil
}

func Example() {
	closer, err := setup()
	defer func() {
		if err := closer.Close(); err != nil {
			log.Printf("failed to close: %s\n", err)
		}
	}()
	if err != nil {
		// handle err
	}

	// Output:
}
