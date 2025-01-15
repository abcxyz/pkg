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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

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

	rootCmd := func() *RootCommand {
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
		{
			name:      "child_help_flags",
			args:      []string{"child", "default", "-h"},
			expStderr: `-string="my-string"`,
		},
	}

	for _, tc := range cases {
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

//nolint:thelper // These aren't actually helpers, and we want the failures to show the correct line
func TestBaseCommand_Prompt(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name    string
		execute func(t *testing.T, cmd *BaseCommand)
	}{
		{
			name: "reads_line",
			execute: func(t *testing.T, cmd *BaseCommand) {
				var stdin bytes.Buffer
				cmd.SetStdin(&stdin)

				stdin.WriteString("hello ")
				stdin.WriteString("world\n")
				stdin.WriteString("🌎")

				got, err := cmd.Prompt(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "hello world"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "reads_multiple",
			execute: func(t *testing.T, cmd *BaseCommand) {
				var stdin bytes.Buffer
				cmd.SetStdin(&stdin)

				stdin.WriteString("hello")
				got, err := cmd.Prompt(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "hello"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}

				stdin.WriteString("world")
				got, err = cmd.Prompt(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "world"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "prompts",
			execute: func(t *testing.T, cmd *BaseCommand) {
				stdinR, stdinW := io.Pipe()
				stdoutR, stdoutW := io.Pipe()
				_, stderrW := io.Pipe()

				cmd.SetStdin(stdinR)
				cmd.SetStdout(stdoutW)
				cmd.SetStderr(stderrW)

				var gotName, gotAge string
				var err error
				errCh := make(chan error)
				go func() {
					defer close(errCh)

					gotName, err = cmd.Prompt(ctx, "name: ")
					if err != nil {
						errCh <- err
						return
					}

					gotAge, err = cmd.Prompt(ctx, "age: ")
					if err != nil {
						errCh <- err
						return
					}
				}()

				readWithTimeout(t, stdoutR, "name: ")
				writeWithTimeout(t, stdinW, "turing\n")

				readWithTimeout(t, stdoutR, "age: ")
				writeWithTimeout(t, stdinW, "41\n")

				select {
				case err := <-errCh:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(time.Second):
					t.Fatal("timed out waiting for prompt to stop")
				}

				if got, want := gotName, "turing"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
				if got, want := gotAge, "41"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd BaseCommand
			cmd.Pipe()

			tc.execute(t, &cmd)
		})
	}
}

//nolint:thelper // These aren't actually helpers, and we want the failures to show the correct line
func TestBaseCommand_PromptAll(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name    string
		execute func(t *testing.T, cmd *BaseCommand)
	}{
		{
			name: "reads_all",
			execute: func(t *testing.T, cmd *BaseCommand) {
				var stdin bytes.Buffer
				cmd.SetStdin(&stdin)

				stdin.WriteString("hello ")
				stdin.WriteString("world\n")
				stdin.WriteString("🌎")

				got, err := cmd.PromptAll(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "hello world\n🌎"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "reads_multiple",
			execute: func(t *testing.T, cmd *BaseCommand) {
				var stdin bytes.Buffer
				cmd.SetStdin(&stdin)

				stdin.WriteString("hello")
				got, err := cmd.PromptAll(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "hello"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}

				stdin.WriteString("world")
				got, err = cmd.PromptAll(ctx, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, "world"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "prompts",
			execute: func(t *testing.T, cmd *BaseCommand) {
				stdinR, stdinW := io.Pipe()
				stdoutR, stdoutW := io.Pipe()
				_, stderrW := io.Pipe()

				cmd.SetStdin(stdinR)
				cmd.SetStdout(stdoutW)
				cmd.SetStderr(stderrW)

				var gotStory string
				var err error
				errCh := make(chan error)
				go func() {
					defer close(errCh)

					gotStory, err = cmd.PromptAll(ctx, "story: ")
					if err != nil {
						errCh <- err
						return
					}
				}()

				readWithTimeout(t, stdoutR, "story: ")
				writeWithTimeout(t, stdinW, "hello world\n")
				writeWithTimeout(t, stdinW, "my name is: ")
				writeWithTimeout(t, stdinW, "🌎")
				if err := stdinW.Close(); err != nil {
					t.Fatal(err)
				}

				select {
				case err := <-errCh:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(time.Second):
					t.Fatal("timed out waiting for prompt to stop")
				}

				if got, want := gotStory, "hello world\nmy name is: 🌎"; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd BaseCommand
			cmd.Pipe()

			tc.execute(t, &cmd)
		})
	}
}

//nolint:thelper // These aren't actually helpers, and we want the failures to show the correct line
func TestBaseCommand_PromptTo(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name    string
		execute func(t *testing.T, cmd *BaseCommand)
	}{
		{
			name: "context_canceled",
			execute: func(t *testing.T, cmd *BaseCommand) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()

				got, err := cmd.PromptTo(ctx, bufio.ScanLines, "")
				if diff := testutil.DiffErrString(err, "context canceled"); diff != "" {
					t.Errorf("unexpected err: %s", diff)
				}
				if got, want := got, ""; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "context_timeout",
			execute: func(t *testing.T, cmd *BaseCommand) {
				ctx, cancel := context.WithDeadline(ctx, time.Unix(0, 0))
				defer cancel()

				got, err := cmd.PromptTo(ctx, bufio.ScanLines, "")
				if diff := testutil.DiffErrString(err, "context deadline exceeded"); diff != "" {
					t.Errorf("unexpected err: %s", diff)
				}
				if got, want := got, ""; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
		{
			name: "eof",
			execute: func(t *testing.T, cmd *BaseCommand) {
				got, err := cmd.PromptTo(ctx, bufio.ScanLines, "")
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if got, want := got, ""; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd BaseCommand
			cmd.Pipe()

			tc.execute(t, &cmd)
		})
	}
}

func TestShouldPrompt(t *testing.T) {
	t.Parallel()

	t.Run("io_pipe", func(t *testing.T) {
		t.Parallel()

		stdinReader, _ := io.Pipe()
		_, stdoutWriter := io.Pipe()
		_, stderrWriter := io.Pipe()

		if !shouldPrompt(stdinReader, stdoutWriter, stderrWriter) {
			t.Error("expected shouldPrompt to be true")
		}
	})

	t.Run("bytes_buffer", func(t *testing.T) {
		t.Parallel()

		if shouldPrompt(&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}) {
			t.Error("expected shouldPrompt to be false")
		}
	})
}

// readWithTimeout does a single read from the given reader. It calls Fatal if
// that read fails or the returned string doesn't contain wantSubStr. May leak a
// goroutine on timeout.
func readWithTimeout(tb testing.TB, r io.Reader, msg string) {
	tb.Helper()

	var got string
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		buf := make([]byte, 64*1_000)
		n, err := r.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		got = string(buf[:n])
	}()

	select {
	case err := <-errCh:
		if err != nil {
			tb.Fatal(err)
		}
	case <-time.After(time.Second):
		tb.Fatalf("timed out waiting to read")
	}

	if got, want := got, msg; got != want {
		tb.Errorf("expected %q to be %q", got, want)
	}
}

// writeWithTimeout does a single write to the given writer. It calls Fatal
// if that read doesn't contain wantSubStr. May leak a goroutine on timeout.
func writeWithTimeout(tb testing.TB, w io.Writer, msg string) {
	tb.Helper()

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		n, err := w.Write([]byte(msg))
		if n < len(msg) {
			errCh <- fmt.Errorf("only wrote %d bytes of %d in message %q",
				n, len(msg), msg)
			return
		}

		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			tb.Fatal(err)
		}
	case <-time.After(time.Second):
		tb.Fatalf("timed out waiting to write %q", msg)
	}
}

func TestBaseCommand_LookupEnv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		lookupEnv LookupEnvFunc
		expValue  string
		expFound  bool
	}{
		{
			name: "uses_override",
			lookupEnv: MapLookuper(map[string]string{
				"PATH": "value",
			}),
			expValue: "value",
			expFound: true,
		},
		{
			name: "uses_override_not_present",
			lookupEnv: MapLookuper(map[string]string{
				"ZIP": "zap",
			}),
			expValue: "",
			expFound: false,
		},
		{
			name:     "uses_os_no_override",
			expValue: os.Getenv("PATH"),
			expFound: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd RootCommand
			cmd.lookupEnv = tc.lookupEnv

			got, found := cmd.LookupEnv("PATH")
			if got, want := got, tc.expValue; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
			if got, want := found, tc.expFound; got != want {
				t.Errorf("expected %t to be %t", got, want)
			}
		})
	}
}

func TestBaseCommand_SetLookupEnv(t *testing.T) {
	t.Parallel()

	var cmd RootCommand

	if got := cmd.lookupEnv; got != nil {
		t.Errorf("expected func to be nil")
	}

	fn := func(_ string) (string, bool) {
		return "banana", false
	}
	cmd.SetLookupEnv(fn)

	got, _ := cmd.lookupEnv("")
	if got, want := got, "banana"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestBaseCommand_GetEnv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		lookupEnv LookupEnvFunc
		expValue  string
	}{
		{
			name: "uses_override",
			lookupEnv: MapLookuper(map[string]string{
				"PATH": "value",
			}),
			expValue: "value",
		},
		{
			name: "uses_override_not_present",
			lookupEnv: MapLookuper(map[string]string{
				"ZIP": "zap",
			}),
			expValue: "",
		},
		{
			name:     "uses_os_no_override",
			expValue: os.Getenv("PATH"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd RootCommand
			cmd.lookupEnv = tc.lookupEnv

			if got, want := cmd.GetEnv("PATH"), tc.expValue; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestBaseCommand_WorkingDir(t *testing.T) {
	t.Parallel()

	var cmd RootCommand
	dir, err := cmd.WorkingDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Errorf("expected working dir to be defined")
	}
}

func TestBaseCommand_ExecutablePath(t *testing.T) {
	t.Parallel()

	var cmd RootCommand
	pth, err := cmd.ExecutablePath()
	if err != nil {
		t.Fatal(err)
	}
	if pth == "" {
		t.Errorf("expected executable path to be defined")
	}
}

type TestCommand struct {
	BaseCommand

	Hide   bool
	Output string
	Error  error

	flagString string
}

func (c *TestCommand) Desc() string {
	return "Test command"
}

func (c *TestCommand) Help() string {
	return "Usage: {{ COMMAND }}"
}

func (c *TestCommand) Flags() *FlagSet {
	set := c.NewFlagSet()

	f := set.NewSection("OPTIONS")

	f.StringVar(&StringVar{
		Name:    "string",
		Example: "my-string",
		Target:  &c.flagString,
		Usage:   "A literal string.",
	})

	return set
}

func (c *TestCommand) Hidden() bool { return c.Hide }

func (c *TestCommand) Run(ctx context.Context, args []string) error {
	if err := c.Flags().Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if err := c.Error; err != nil {
		return err
	}

	if v := c.Output; v != "" {
		c.Outf(v)
	}

	return nil
}
