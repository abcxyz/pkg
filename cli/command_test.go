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

package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
)

func TestRootCommand_Help(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cmd  Command
		exp  string
	}{
		{
			name: "no_commands",
			cmd: &RootCommand{
				Name: "test",
			},
			exp: `Usage: test COMMAND`,
		},
		{
			name: "nil_command",
			cmd: &RootCommand{
				Name: "test",
				Commands: map[string]CommandFactory{
					"nil": func() Command {
						return nil
					},
				},
			},
			exp: `Usage: test COMMAND`,
		},
		{
			name: "single",
			cmd: &RootCommand{
				Name: "test",
				Commands: map[string]CommandFactory{
					"one": func() Command { return &TestCommand{} },
				},
			},
			exp: `
Usage: test COMMAND

  one    Test command
`,
		},
		{
			name: "multiple",
			cmd: &RootCommand{
				Name: "test",
				Commands: map[string]CommandFactory{
					"one":   func() Command { return &TestCommand{} },
					"two":   func() Command { return &TestCommand{} },
					"three": func() Command { return &TestCommand{} },
				},
			},
			exp: `
Usage: test COMMAND

  one      Test command
  three    Test command
  two      Test command
`,
		},
		{
			name: "hidden",
			cmd: &RootCommand{
				Name: "test",
				Commands: map[string]CommandFactory{
					"one": func() Command { return &TestCommand{} },
					"two": func() Command {
						return &TestCommand{
							Hide: true,
						}
					},
				},
			},
			exp: `
Usage: test COMMAND

  one    Test command
`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, want := strings.TrimSpace(tc.cmd.Help()), strings.TrimSpace(tc.exp); got != want {
				t.Errorf("expected\n\n%s\n\nto be\n\n%s\n\n", got, want)
			}
		})
	}
}

func TestRootCommand_Run(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	rootCmd := func() Command {
		return &RootCommand{
			Name:    "test",
			Version: "1.2.3",
			Commands: map[string]CommandFactory{
				"default": func() Command {
					return &TestCommand{
						Output: "output from default command",
					}
				},
				"error": func() Command {
					return &TestCommand{
						Error: fmt.Errorf("a bad thing happened"),
					}
				},
				"hidden": func() Command {
					return &TestCommand{
						Hide:   true,
						Output: "you found me",
					}
				},
				"child": func() Command {
					return &RootCommand{
						Name:        "child",
						Description: "This is a child command",
						Commands: map[string]CommandFactory{
							"default": func() Command {
								return &TestCommand{
									Output: "output from child",
								}
							},
						},
					}
				},
			},
		}
	}

	cases := []struct {
		name      string
		cmd       Command
		args      []string
		err       string
		expStdout string
		expStderr string
	}{
		{
			name:      "nothing",
			args:      nil,
			expStderr: `Usage: test COMMAND`,
		},
		{
			name:      "-h",
			args:      []string{"-h"},
			expStderr: `Usage: test COMMAND`,
		},
		{
			name:      "-help",
			args:      []string{"-help"},
			expStderr: `Usage: test COMMAND`,
		},
		{
			name:      "--help",
			args:      []string{"-help"},
			expStderr: `Usage: test COMMAND`,
		},
		{
			name:      "-v",
			args:      []string{"-v"},
			expStderr: `1.2.3`,
		},
		{
			name:      "-version",
			args:      []string{"-version"},
			expStderr: `1.2.3`,
		},
		{
			name:      "--version",
			args:      []string{"--version"},
			expStderr: `1.2.3`,
		},
		{
			name: "unknown_command",
			args: []string{"nope"},
			err:  `unknown command "nope": run "test -help" for a list of commands`,
		},
		{
			name:      "runs_parent_command",
			args:      []string{"default"},
			expStdout: `output from default command`,
		},
		{
			name: "handles_error",
			args: []string{"error"},
			err:  `a bad thing happened`,
		},
		{
			name:      "runs_hidden",
			args:      []string{"hidden"},
			expStdout: `you found me`,
		},
		{
			name:      "runs_child",
			args:      []string{"child", "default"},
			expStdout: `output from child`,
		},
		{
			name:      "child_version",
			args:      []string{"child", "-v"},
			expStderr: `1.2.3`,
		},
		{
			name:      "child_help",
			args:      []string{"child", "-h"},
			expStderr: `Usage: test child COMMAND`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := rootCmd()
			_, stdout, stderr := cmd.Pipe()

			err := cmd.Run(ctx, tc.args)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}

			if got, want := strings.TrimSpace(stdout.String()), strings.TrimSpace(tc.expStdout); !strings.Contains(got, want) {
				t.Errorf("expected\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
			if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
				t.Errorf("expected\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
		})
	}
}

type TestCommand struct {
	BaseCommand

	Hide   bool
	Output string
	Error  error
}

func (c *TestCommand) Desc() string    { return "Test command" }
func (c *TestCommand) Help() string    { return "Test command help" }
func (c *TestCommand) Flags() *FlagSet { return NewFlagSet() }
func (c *TestCommand) Hidden() bool    { return c.Hide }
func (c *TestCommand) Run(ctx context.Context, args []string) error {
	if err := c.Error; err != nil {
		return err
	}

	if v := c.Output; v != "" {
		fmt.Fprint(c.Stdout(), v)
	}

	return nil
}