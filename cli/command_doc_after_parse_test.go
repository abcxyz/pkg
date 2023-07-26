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

package cli_test

import (
	"context"
	"fmt"
	"os"

	"github.com/abcxyz/pkg/cli"
)

type StreamCommand struct {
	cli.BaseCommand

	flagOldAddress string
	flagAddress    string
}

func (c *StreamCommand) Desc() string {
	return "Stream a data stream"
}

func (c *StreamCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Stream a data stream.
`
}

func (c *StreamCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	f := set.NewSection("SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "server-address",
		Example: "https://my.corp.server:8145",
		Default: "http://localhost:8145",
		EnvVar:  "CLI_SERVER_ADDRESS",
		Target:  &c.flagAddress,
		Usage:   "Endpoint, including protocol and port, the server.",
	})

	// Deprecated - use -server-address instead.
	f.StringVar(&cli.StringVar{
		Name:    "address",
		Default: "http://localhost:8145",
		Target:  &c.flagOldAddress,
		Hidden:  true,
	})

	// Each AfterParse will be invoked after flags have been parsed.
	set.AfterParse(func(existingErr error) error {
		// Example of deferred defaulting. At this point, it is safe to set values
		// of flags to other values.
		if c.flagOldAddress != "" {
			c.Errf("WARNING: -address is deprecated, use -server-address instead")
		}
		if c.flagAddress == "" {
			c.flagAddress = c.flagOldAddress
		}

		return nil
	})

	return set
}

func (c *StreamCommand) Run(ctx context.Context, args []string) error {
	if err := c.Flags().Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	c.Outf("address: %s", c.flagAddress)

	// TODO: implement
	return nil
}

func Example_afterParse() {
	ctx := context.Background()

	rootCmd := func() cli.Command {
		return &cli.RootCommand{
			Name:    "my-tool",
			Version: "1.2.3",
			Commands: map[string]cli.CommandFactory{
				"stream": func() cli.Command {
					return &StreamCommand{}
				},
			},
		}
	}

	cmd := rootCmd()

	// Help output is written to stderr by default. Redirect to stdout so the
	// "Output" assertion works.
	cmd.SetStderr(os.Stdout)

	if err := cmd.Run(ctx, []string{"stream", "-address", "1.2.3"}); err != nil {
		panic(err)
	}

	// Output:
	// WARNING: -address is deprecated, use -server-address instead
	// address: http://localhost:8145
}
