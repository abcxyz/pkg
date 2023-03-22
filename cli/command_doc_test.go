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
	"strconv"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

type CountCommand struct {
	cli.BaseCommand

	flagStep int64
}

func (c *CountCommand) Desc() string {
	return "Counts from 0 up to a number"
}

func (c *CountCommand) Help() string {
	return strings.Trim(`
Usage: my-tool count [options] MAX

  The count command prints out a list of numbers starting from 0 up to and
  including the provided MAX.

      $ my-tool count 50

  The value for MAX must be a positive integer.

`+c.Flags().Help(), "\n")
}

func (c *CountCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	f := set.NewSection("Number options")

	f.Int64Var(&cli.Int64Var{
		Name:    "step",
		Aliases: []string{"s"},
		Example: "1",
		Default: 1,
		Target:  &c.flagStep,
		Usage:   "Numeric value by which to increment between each number.",
	})

	return set
}

func (c *CountCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	args = f.Args()
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument, got %q", args)
	}

	maxStr := args[0]
	max, err := strconv.ParseInt(maxStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse max: %w", err)
	}

	for i := int64(0); i <= max; i += c.flagStep {
		fmt.Fprintln(c.Stdout(), i)
	}

	return nil
}

func Example_commandWithFlags() {
	ctx := context.Background()

	// Create the command.
	rootCmd := func() cli.Command {
		return &cli.RootCommand{
			Name:    "my-tool",
			Version: "1.2.3",
			Commands: map[string]cli.CommandFactory{
				"count": func() cli.Command {
					return &CountCommand{}
				},
			},
		}
	}

	cmd := rootCmd()

	// Help output is written to stderr by default. Redirect to stdout so the
	// "Output" assertion works.
	cmd.SetStderr(os.Stdout)

	fmt.Fprintln(cmd.Stdout(), "\nUp to 3:")
	if err := cmd.Run(ctx, []string{"count", "3"}); err != nil {
		panic(err)
	}

	fmt.Fprintln(cmd.Stdout(), "\nUp to 10, stepping 2")
	if err := cmd.Run(ctx, []string{"count", "-step=2", "10"}); err != nil {
		panic(err)
	}

	// Output:
	//
	// Up to 3:
	// 0
	// 1
	// 2
	// 3
	//
	// Up to 10, stepping 2
	// 0
	// 2
	// 4
	// 6
	// 8
	// 10
}
