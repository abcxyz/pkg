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

func TestBaseCommand_Prompt_Values(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name      string
		args      []string
		msg       string
		inputs    []string
		err       string
		expStderr string
	}{
		{
			name:   "sets_input_value",
			args:   []string{"prompt"},
			msg:    "enter input value:",
			inputs: []string{"input"},
		}, {
			name:   "handles_multiple_prompts",
			args:   []string{"prompt"},
			msg:    "enter input value:",
			inputs: []string{"input1", "input2"},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd RootCommand
			stdin, _, stderr := cmd.Pipe()

			for _, input := range tc.inputs {
				stdin.WriteString(input)

				v, err := cmd.Prompt(ctx, tc.msg)
				if diff := testutil.DiffErrString(err, tc.err); diff != "" {
					t.Errorf("unexpected err: %s", diff)
				}
				if got, want := v, input; got != want {
					t.Errorf("expected\n\n%s\n\nto equal\n\n%s\n\n", got, want)
				}
				if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
					t.Errorf("expected\n\n%s\n\nto contain\n\n%s\n\n", got, want)
				}
			}
		})
	}
}

func TestBaseCommand_Prompt_Cancels(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name      string
		args      []string
		msg       string
		inputs    []string
		exp       string
		err       string
		expStderr string
	}{
		{
			name:   "context_cancels",
			args:   []string{"prompt"},
			msg:    "enter value:",
			inputs: []string{"input1", "input2"},
			err:    "context canceled",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd RootCommand
			stdin, _, stderr := cmd.Pipe()

			for _, input := range tc.inputs {
				stdin.WriteString(input)

				_, err := cmd.Prompt(ctx, tc.msg)
				if diff := testutil.DiffErrString(err, ""); diff != "" {
					t.Errorf("unexpected err: %s", diff)
				}
				if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
					t.Errorf("expected\n\n%s\n\nto contain\n\n%s\n\n", got, want)
				}
			}

			cancelCtx, cancel := context.WithCancel(ctx)
			cancel()

			_, err := cmd.Prompt(cancelCtx, tc.msg)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
				t.Errorf("expected\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
		})
	}
}

// Tests back-and-forth conversation using Prompt().
func TestBaseCommand_Prompt_Dialog(t *testing.T) {
	t.Parallel()

	cmd := &RootCommand{}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	_, stderrWriter := io.Pipe()

	cmd.SetStdin(stdinReader)
	cmd.SetStdout(stdoutWriter)
	cmd.SetStderr(stderrWriter)

	ctx := context.Background()
	errCh := make(chan error)
	var color, iceCream string
	go func() {
		defer close(errCh)
		var err error
		color, err = cmd.Prompt(ctx, "Please enter a color: ")
		if err != nil {
			errCh <- err
			return
		}

		iceCream, err = cmd.Prompt(ctx, "Please enter a flavor of ice cream: ")
		if err != nil {
			errCh <- err
			return
		}
	}()

	readWithTimeout(t, stdoutReader, "Please enter a color:")
	writeWithTimeout(t, stdinWriter, "blue\n")
	readWithTimeout(t, stdoutReader, "Please enter a flavor of ice cream:")
	writeWithTimeout(t, stdinWriter, "mint chip\n")

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for background Prompt() goroutine to exit")
	}

	if color != "blue" {
		t.Fatalf(`got color %q, wanted "blue"`, color)
	}
	if iceCream != "mint chip" {
		t.Fatalf(`got iceCream %q, wanted "mint chip"`, iceCream)
	}
}

func TestShouldPrompt_Pipe(t *testing.T) {
	stdinReader, _ := io.Pipe()
	_, stdoutWriter := io.Pipe()
	_, stderrWriter := io.Pipe()

	if !shouldPrompt(stdinReader, stdoutWriter, stderrWriter) {
		t.Fatal("shouldPrompt() got false, want true, when passed io.Pipe")
	}
}

func TestShouldPrompt_ByteBuffer(t *testing.T) {
	if shouldPrompt(&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}) {
		t.Fatal("shouldPrompt() got true, want false, when passed a Buffer")
	}
}

// readWithTimeout does a single read from the given reader. It calls Fatal if
// that read fails or the returned string doesn't contain wantSubStr. May leak a
// goroutine on timeout.
func readWithTimeout(tb testing.TB, r io.Reader, wantSubstr string) {
	tb.Helper()

	tb.Logf("readWith starting with %q", wantSubstr)

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
		tb.Fatalf("timed out waiting to read %q", wantSubstr)
	}

	if !strings.Contains(got, wantSubstr) {
		tb.Fatalf("got a prompt %q, but wanted a prompt containing %q", got, wantSubstr)
	}
}

// writeWithTimeout does a single write to the given writer. It calls Fatal
// if that read doesn't contain wantSubStr. May leak a goroutine on timeout.
func writeWithTimeout(tb testing.TB, w io.Writer, msg string) {
	tb.Helper()

	tb.Logf("writeWithTimeout starting with %q", msg)

	errCh := make(chan error)
	go func() {
		_, err := w.Write([]byte(msg))
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
		tc := tc

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
		tc := tc

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
