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
	"os"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

type EatCommand struct {
	cli.BaseCommand
}

func (c *EatCommand) Desc() string {
	return "Eat some food"
}

func (c *EatCommand) Help() string {
	return strings.Trim(`
The eat command eats food.
`+c.Flags().Help(), "\n")
}

func (c *EatCommand) Flags() *cli.FlagSet {
	return cli.NewFlagSet()
}

func (c *EatCommand) Run(ctx context.Context, args []string) error {
	// TODO: implement
	return nil
}

type DrinkCommand struct {
	cli.BaseCommand
}

func (c *DrinkCommand) Desc() string {
	return "Drink some water"
}

func (c *DrinkCommand) Help() string {
	return strings.Trim(`
The drink command drinks water.
`+c.Flags().Help(), "\n")
}

func (c *DrinkCommand) Flags() *cli.FlagSet {
	return cli.NewFlagSet()
}

func (c *DrinkCommand) Run(ctx context.Context, args []string) error {
	// TODO: implement
	return nil
}

func Example_commandGroup() {
	ctx := context.Background()

	rootCmd := func() cli.Command {
		return &cli.RootCommand{
			Name:    "my-tool",
			Version: "1.2.3",
			Commands: map[string]cli.CommandFactory{
				"eat": func() cli.Command {
					return &EatCommand{}
				},
				"drink": func() cli.Command {
					return &DrinkCommand{}
				},
			},
		}
	}

	cmd := rootCmd()

	// Help output is written to stderr by default. Redirect to stdout so the
	// "Output" assertion works.
	cmd.SetStderr(os.Stdout)

	if err := cmd.Run(ctx, []string{"-h"}); err != nil {
		// TODO: handle error
		panic(err)
	}

	// Output:
	// Usage: my-tool COMMAND
	//
	//   drink    Drink some water
	//   eat      Eat some food
}
