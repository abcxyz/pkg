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

// ServerFlags represent the shared flags among all server commands. Embed this
// struct into any commands that interact with a server.
type ServerFlags struct {
	flagAddress       string
	flagTLSSkipVerify bool
}

func (sf *ServerFlags) Register(set *cli.FlagSet) {
	f := set.NewSection("SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "server-address",
		Example: "https://my.corp.server:8145",
		Default: "http://localhost:8145",
		EnvVar:  "CLI_SERVER_ADDRESS",
		Target:  &sf.flagAddress,
		Usage:   "Endpoint, including protocol and port, the server.",
	})

	f.BoolVar(&cli.BoolVar{
		Name:    "insecure",
		Default: false,
		EnvVar:  "CLI_SERVER_TLS_SKIP_VERIFY",
		Target:  &sf.flagTLSSkipVerify,
		Usage:   "Skip TLS verification. This is bad, please don't do it.",
	})
}

type UploadCommand struct {
	cli.BaseCommand
	serverFlags ServerFlags
}

func (c *UploadCommand) Desc() string {
	return "Upload a file"
}

func (c *UploadCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Upload a file to the server.
`
}

func (c *UploadCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()
	c.serverFlags.Register(set)
	return set
}

func (c *UploadCommand) Run(ctx context.Context, args []string) error {
	if err := c.Flags().Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	_ = c.serverFlags.flagAddress
	_ = c.serverFlags.flagTLSSkipVerify

	// TODO: implement
	return nil
}

type DownloadCommand struct {
	cli.BaseCommand
	serverFlags ServerFlags
}

func (c *DownloadCommand) Desc() string {
	return "Download a file"
}

func (c *DownloadCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Download a file from the server.
`
}

func (c *DownloadCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()
	c.serverFlags.Register(set)
	return set
}

func (c *DownloadCommand) Run(ctx context.Context, args []string) error {
	if err := c.Flags().Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	_ = c.serverFlags.flagAddress
	_ = c.serverFlags.flagTLSSkipVerify

	// TODO: implement
	return nil
}

func Example_persistentFlags() {
	ctx := context.Background()

	rootCmd := func() cli.Command {
		return &cli.RootCommand{
			Name:    "my-tool",
			Version: "1.2.3",
			Commands: map[string]cli.CommandFactory{
				"download": func() cli.Command {
					return &DownloadCommand{}
				},
				"upload": func() cli.Command {
					return &UploadCommand{}
				},
			},
		}
	}

	cmd := rootCmd()

	// Help output is written to stderr by default. Redirect to stdout so the
	// "Output" assertion works.
	cmd.SetStderr(os.Stdout)

	cmd.Outf("\nTop-level help:")
	if err := cmd.Run(ctx, []string{"-h"}); err != nil {
		panic(err)
	}

	cmd.Outf("\nCommand-level help:")
	if err := cmd.Run(ctx, []string{"download", "-h"}); err != nil {
		panic(err)
	}

	// Output:
	// Top-level help:
	// Usage: my-tool COMMAND
	//
	//   download    Download a file
	//   upload      Upload a file
	//
	// Command-level help:
	// Usage: my-tool download [options]
	//
	//   Download a file from the server.
	//
	// SERVER OPTIONS
	//
	//     -insecure
	//         Skip TLS verification. This is bad, please don't do it. The default
	//         value is "false". This option can also be specified with the
	//         CLI_SERVER_TLS_SKIP_VERIFY environment variable.
	//
	//     -server-address="https://my.corp.server:8145"
	//         Endpoint, including protocol and port, the server. The default value
	//         is "http://localhost:8145". This option can also be specified with the
	//         CLI_SERVER_ADDRESS environment variable.
}
