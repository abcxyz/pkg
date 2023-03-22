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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"
)

// Command is the interface for a command or subcommand. Most of these functions
// have default implementations on [BaseCommand].
type Command interface {
	// Desc provides a short, one-line description of the command. It should be
	// shorter than 50 characters.
	Desc() string

	// Help is the long-form help output. It should include usage instructions and
	// flag information.
	Help() string

	// Hidden indicates whether the command is hidden from help output.
	Hidden() bool

	// Run executes the command.
	Run(ctx context.Context, args []string) error

	Prompt(msg string) (string, error)

	// Stdout returns the stdout stream. SetStdout sets the stdout stream.
	Stdout() io.Writer
	SetStdout(w io.Writer)

	// Stderr returns the stderr stream. SetStderr sets the stderr stream.
	Stderr() io.Writer
	SetStderr(w io.Writer)

	// Stdin returns the stdin stream. SetStdin sets the stdin stream.
	Stdin() io.Reader
	SetStdin(r io.Reader)

	// Pipe creates new unqiue stdin, stdout, and stderr buffers, sets them on the
	// command, and returns them. This is most useful for testing where callers
	// want to simulate inputs or assert certain command outputs.
	Pipe() (stdin, stdout, stderr *bytes.Buffer)
}

// CommandFactory returns a new instance of a command. This returns a function
// instead of allocations because we want the CLI to load as fast as possible,
// so we lazy load as much as possible.
type CommandFactory func() Command

// Ensure [RootCommand] implements [Command].
var _ Command = (*RootCommand)(nil)

// RootCommand represents a command root for a parent or collection of
// subcommands.
type RootCommand struct {
	BaseCommand

	// Name is the name of the command or subcommand. For top-level commands, this
	// should be the binary name. For subcommands, this should be the name of the
	// subcommand.
	Name string

	// Description is the human-friendly description of the command.
	Description string

	// Hide marks the entire subcommand as hidden. It will not be shown in help
	// output.
	Hide bool

	// Version defines the version information for the command. This can be
	// omitted for subcommands as it will be inherited from the parent.
	Version string

	// Commands is the list of sub commands.
	Commands map[string]CommandFactory
}

// Desc is the root command description. It is used to satisfy the [Command]
// interface.
func (r *RootCommand) Desc() string {
	return r.Description
}

// Hidden determines whether the command group is hidden. It is used to satisfy
// the [Command] interface.
func (r *RootCommand) Hidden() bool {
	return r.Hide
}

// Help compiles structured help information. It is used to satisfy the
// [Command] interface.
func (r *RootCommand) Help() string {
	var b strings.Builder

	longest := 0
	names := make([]string, 0, len(r.Commands))
	for name := range r.Commands {
		names = append(names, name)
		if l := len(name); l > longest {
			longest = l
		}
	}
	sort.Strings(names)

	fmt.Fprintf(&b, "Usage: %s COMMAND\n\n", r.Name)
	for _, name := range names {
		cmd := r.Commands[name]()
		if cmd == nil {
			continue
		}

		if !cmd.Hidden() {
			fmt.Fprintf(&b, "  %-*s%s\n", longest+4, name, cmd.Desc())
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// Run executes the command and prints help output or delegates to a subcommand.
func (r *RootCommand) Run(ctx context.Context, args []string) error {
	name, args := extractCommandAndArgs(args)

	// Short-circuit top-level help.
	if name == "" || name == "-h" || name == "-help" || name == "--help" {
		fmt.Fprintln(r.Stderr(), r.Help())
		return nil
	}

	// Short-circuit version.
	if name == "-v" || name == "-version" || name == "--version" {
		fmt.Fprintln(r.Stderr(), r.Version)
		return nil
	}

	cmd, ok := r.Commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}
	instance := cmd()

	// Ensure the child inherits the streams from the root.
	instance.SetStdin(r.stdin)
	instance.SetStdout(r.stdout)
	instance.SetStderr(r.stderr)

	// If this is a subcommand, prefix the name with the parent and inherit some
	// values.
	if typ, ok := instance.(*RootCommand); ok {
		typ.Name = r.Name + " " + typ.Name
		typ.Version = r.Version
		return typ.Run(ctx, args)
	}

	if err := instance.Run(ctx, args); err != nil {
		// Special case requesting help.
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(instance.Stderr(), instance.Help())
			return nil
		}
		//nolint:wrapcheck // We want to bubble this error exactly as-is.
		return err
	}
	return nil
}

// extractCommandAndArgs is a helper that pulls the subcommand and arguments.
func extractCommandAndArgs(args []string) (string, []string) {
	switch len(args) {
	case 0:
		return "", nil
	case 1:
		return args[0], nil
	default:
		return args[0], args[1:]
	}
}

// BaseCommand is the default command structure. All commands should embed this
// structure.
type BaseCommand struct {
	stdout, stderr io.Writer
	stdin          io.Reader
}

// Hidden indicates whether the command is hidden. The default is unhidden.
func (c *BaseCommand) Hidden() bool {
	return false
}

// Prompt prompts the user for a value. If stdin is a tty, it prompts. Otherwise
// it reads from the reader.
func (c *BaseCommand) Prompt(msg string) (string, error) {
	scanner := bufio.NewScanner(io.LimitReader(c.Stdin(), 64*1_000))

	if c.Stdin() == os.Stdin && isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprint(c.Stdout(), msg)
	}

	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	return scanner.Text(), nil
}

// Stdout returns the stdout stream.
func (c *BaseCommand) Stdout() io.Writer {
	if v := c.stdout; v != nil {
		return v
	}
	return os.Stdout
}

// SetStdout sets the standard out.
func (c *BaseCommand) SetStdout(w io.Writer) {
	c.stdout = w
}

// Stderr returns the stderr stream.
func (c *BaseCommand) Stderr() io.Writer {
	if v := c.stderr; v != nil {
		return v
	}
	return os.Stderr
}

// SetStdout sets the standard error.
func (c *BaseCommand) SetStderr(w io.Writer) {
	c.stderr = w
}

// Stdin returns the stdin stream.
func (c *BaseCommand) Stdin() io.Reader {
	if v := c.stdin; v != nil {
		return v
	}
	return os.Stdin
}

// SetStdout sets the standard input.
func (c *BaseCommand) SetStdin(r io.Reader) {
	c.stdin = r
}

// Pipe creates new unqiue stdin, stdout, and stderr buffers, sets them on the
// command, and returns them. This is most useful for testing where callers want
// to simulate inputs or assert certain command outputs.
func (c *BaseCommand) Pipe() (stdin, stdout, stderr *bytes.Buffer) {
	stdin = bytes.NewBuffer(nil)
	stdout = bytes.NewBuffer(nil)
	stderr = bytes.NewBuffer(nil)
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	return
}
