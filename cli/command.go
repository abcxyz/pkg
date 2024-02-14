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
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/mattn/go-isatty"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

// isCompletionRequest returns true if the invocation is a completion request,
// or false otherwise. These are environment variables read by
// posener/complete/v2:
//
//	https://github.com/posener/complete/blob/3f9152130d1c1e72ef5b0091380bfbeb7fafecf5/complete.go#L61-L65
var isCompletionRequest = os.Getenv("COMP_LINE") != "" ||
	os.Getenv("COMP_INSTALL") != "" ||
	os.Getenv("COMP_UNINSTALL") != ""

// Command is the interface for a command or subcommand. Most of these functions
// have default implementations on [BaseCommand].
type Command interface {
	// Desc provides a short, one-line description of the command. It must be
	// shorter than 50 characters and should not end with a period or punctuation.
	Desc() string

	// Help is the long-form help output. It should include usage instructions and
	// flag information.
	//
	// Callers can insert the literal string "{{ COMMAND }}" which will be
	// replaced with the actual subcommand structure.
	Help() string

	// Flags returns the list of flags that are defined on the command.
	Flags() *FlagSet

	// Hidden indicates whether the command is hidden from help output.
	Hidden() bool

	// Run executes the command.
	Run(ctx context.Context, args []string) error

	// Stdout returns the stdout stream. SetStdout sets the stdout stream.
	Stdout() io.Writer
	SetStdout(w io.Writer)

	// Outf is a shortcut to write to [Command.Stdout]. It automatically appends a
	// trailing newline if one is not present.
	Outf(format string, a ...any)

	// Stderr returns the stderr stream. SetStderr sets the stderr stream.
	Stderr() io.Writer
	SetStderr(w io.Writer)

	// Errf is a shortcut to write to [Command.Stderr]. It automatically appends a
	// trailing newline if one is not present.
	Errf(format string, a ...any)

	// Stdin returns the stdin stream. SetStdin sets the stdin stream.
	Stdin() io.Reader
	SetStdin(r io.Reader)
}

// ArgPredictor is an optional interface that [Command] can implement to declare
// predictions for their arguments. By default, commands predict nothing for
// arguments.
type ArgPredictor interface {
	PredictArgs() complete.Predictor
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
			// Trim any trailing periods or spaces.
			desc := strings.TrimRightFunc(cmd.Desc(), func(r rune) bool {
				return unicode.IsSpace(r) || r == '\uFEFF' || r == '.' || r == '!' || r == '?'
			})
			fmt.Fprintf(&b, "  %-*s%s\n", longest+4, name, desc)
		}
	}

	return strings.TrimRightFunc(b.String(), unicode.IsSpace)
}

// Run executes the command and prints help output or delegates to a subcommand.
func (r *RootCommand) Run(ctx context.Context, args []string) error {
	// This can be a very expensive operation, since it requires instantiating all
	// commands. Therefore we only do this when we are certain the user is tabbing
	// for autocompletions.
	if isCompletionRequest {
		completer := buildCompleteCommands(r)
		completer.Complete(r.Name)
	}

	name, args := extractCommandAndArgs(args)

	// Short-circuit top-level help.
	if name == "" || name == "-h" || name == "-help" || name == "--help" {
		r.Errf(formatHelp(r.Help(), r.Name, r.Flags()))
		return nil
	}

	// Short-circuit version.
	if name == "-v" || name == "-version" || name == "--version" {
		r.Errf(r.Version)
		return nil
	}

	cmd, ok := r.Commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q: run \"%s -help\" for a list of "+
			"commands", name, r.Name)
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
			instance.Errf(formatHelp(instance.Help(), r.Name+" "+name, instance.Flags()))
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

// formatHelp is a helper function that does variable replacement from the help
// string.
func formatHelp(help, name string, flags *FlagSet) string {
	h := strings.Trim(help, "\n")
	if flags != nil {
		if v := strings.Trim(flags.Help(), "\n"); v != "" {
			h = h + "\n\n" + v
		}
	}
	return strings.ReplaceAll(h, "{{ COMMAND }}", name)
}

// BaseCommand is the default command structure. All commands should embed this
// structure.
type BaseCommand struct {
	stdout, stderr io.Writer
	stdin          io.Reader

	lookupEnv LookupEnvFunc
}

// NewFlagSet creates a new flag set that inherits properties from the command.
func (c *BaseCommand) NewFlagSet(opts ...Option) *FlagSet {
	return NewFlagSet(WithLookupEnv(c.LookupEnv))
}

// Flags returns the base command flags, which is always nil.
func (c *BaseCommand) Flags() *FlagSet {
	return nil
}

// Hidden indicates whether the command is hidden. The default is unhidden.
func (c *BaseCommand) Hidden() bool {
	return false
}

// Prompt asks for user input and reads it from [Stdin] until it encounters a
// newline character. If there's an input stream (e.g. a pipe), it will read the
// pipe. The result will not include the trailing newline or carriage return.
// For more information about the conditions under which the prompt is
// displayed, see [PromptTo].
func (c *BaseCommand) Prompt(ctx context.Context, msg string, args ...any) (string, error) {
	// Unlike [bufio.ScanLines], this function reads at most a single line and
	// then returns. bufio.ScanLines yields for each line, but this function
	// yields a single line and then marks the read as terminated. This ensures we
	// read up to the first newline character, but ignore lines thereafter
	// (matching the original implementation's behavior).
	splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			// We have a full newline-terminated line.
			return i + 1, dropCR(data[0:i]), bufio.ErrFinalToken
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), dropCR(data), bufio.ErrFinalToken
		}
		// Request more data.
		return 0, nil, nil
	}

	return c.PromptTo(ctx, splitFunc, msg, args...)
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// PromptAll asks for user input and reads from [Stdin] until it encounters an
// EOF. If there's an input stream (e.g. a pipe), it will read the pipe. For
// more information about the conditions under which the prompt is displayed,
// see [PromptTo].
func (c *BaseCommand) PromptAll(ctx context.Context, msg string, args ...any) (string, error) {
	splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		return len(data), data, nil
	}
	return c.PromptTo(ctx, splitFunc, msg, args...)
}

// PromptTo provides a mechanism for asking for user input. It reads from
// [Stdin], using the provided scanner split function. If there's an input
// stream (e.g. a pipe), it will read the pipe.
//
// The prompt will be printed to c.Stdout() if any of these cases is true:
//   - the terminal is a TTY (for real user interaction)
//   - c.StdIn(), c.Stdout(), and c.Stderr() came from io.Pipe() (for unit-testing back-and-forth dialog)
//
// It will fail if stdin pipe and the terminal is not a tty. If the context is
// canceled, [Stdin] could be in a partially-read state.
func (c *BaseCommand) PromptTo(ctx context.Context, splitFunc bufio.SplitFunc, msg string, args ...any) (string, error) {
	stdin, stdout, stderr := c.Stdin(), c.Stdout(), c.Stderr()

	if shouldPrompt(stdin, stdout, stderr) {
		fmt.Fprintf(stdout, msg, args...)
	}

	var b strings.Builder
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		scanner := bufio.NewScanner(stdin)
		scanner.Split(splitFunc)

		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}

			b.WriteString(scanner.Text())

			if err := scanner.Err(); err != nil {
				errCh <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("failed to prompt: %w", ctx.Err())
	case err := <-errCh:
		if ctx.Err() != nil {
			return "", fmt.Errorf("failed to prompt: %w", ctx.Err())
		}

		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
	}

	return b.String(), nil
}

// shouldPrompt returns whether we're in a situation where it makes sense to
// prompt for input. See the comment on Prompt().
func shouldPrompt(stdin io.Reader, stdout, stderr io.Writer) bool {
	if stdin == os.Stdin && isatty.IsTerminal(os.Stdin.Fd()) {
		return true
	}

	_, stdinIsPipe := stdin.(*io.PipeReader)
	_, stdoutIsPipe := stdout.(*io.PipeWriter)
	_, stderrIsPipe := stderr.(*io.PipeWriter)

	return stdinIsPipe && stdoutIsPipe && stderrIsPipe
}

// Outf is a shortcut to write to [BaseCommand.Stdout].
func (c *BaseCommand) Outf(format string, a ...any) {
	fmt.Fprintf(c.Stdout(), format+"\n", a...)
}

// Stdout returns the stdout stream.
func (c *BaseCommand) Stdout() io.Writer {
	if c.stdout != nil {
		return c.stdout
	}
	return os.Stdout
}

// SetStdout sets the standard out.
func (c *BaseCommand) SetStdout(w io.Writer) {
	c.stdout = w
}

// Errf is a shortcut to write to [BaseCommand.Stderr].
func (c *BaseCommand) Errf(format string, a ...any) {
	fmt.Fprintf(c.Stderr(), format+"\n", a...)
}

// Stderr returns the stderr stream.
func (c *BaseCommand) Stderr() io.Writer {
	if c.stderr != nil {
		return c.stderr
	}
	return os.Stderr
}

// SetStdout sets the standard error.
func (c *BaseCommand) SetStderr(w io.Writer) {
	c.stderr = w
}

// Stdin returns the stdin stream.
func (c *BaseCommand) Stdin() io.Reader {
	if c.stdin != nil {
		return c.stdin
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

// LookupEnv returns the value from the environment at the given key. The second
// return value indicates whether the value was set.
func (c *BaseCommand) LookupEnv(key string) (string, bool) {
	if c.lookupEnv != nil {
		return c.lookupEnv(key)
	}
	return os.LookupEnv(key)
}

// SetLookupEnv sets the CLIs environment lookup logic to use the provided
// function.
func (c *BaseCommand) SetLookupEnv(fn LookupEnvFunc) {
	c.lookupEnv = fn
}

// GetEnv returns the value from the environment at the given key.
func (c *BaseCommand) GetEnv(key string) string {
	v, _ := c.LookupEnv(key)
	return v
}

// WorkingDir returns the absolute path of current working directory from where
// the command was started. All symlinks are resolved to their real paths.
func (c *BaseCommand) WorkingDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	symCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks for current working directory: %w", err)
	}

	abs, err := filepath.Abs(symCwd)
	if err != nil {
		return "", fmt.Errorf("failed to compute absolute path for current working directory: %w", err)
	}

	return abs, nil
}

// ExecutablePath returns the absolute path of the CLI executable binary. All
// symlinks are resolved to their real values.
func (c *BaseCommand) ExecutablePath() (string, error) {
	pth, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	sym, err := filepath.EvalSymlinks(pth)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate executable path symlink: %w", err)
	}

	return sym, nil
}

// buildCompleteCommands maps a [Command] to its flag and argument completion. If
// the given command is a [RootCommand], it recursively builds the entire
// complete tree.
//
// WARNING: This function is expensive as it requires instantiating the entire
// command tree (including all subcommands), which is inherently a recursive
// operation. The function makes no attempt to detect cycles.
func buildCompleteCommands(cmd Command) *complete.Command {
	completer := &complete.Command{
		Sub:   make(map[string]*complete.Command),
		Flags: make(map[string]complete.Predictor),
		Args:  predict.Nothing,
	}

	if typ, ok := cmd.(ArgPredictor); ok {
		completer.Args = typ.PredictArgs()
	}

	f := cmd.Flags()
	if f != nil {
		f.VisitAll(func(f *flag.Flag) {
			typ, ok := f.Value.(Value)
			if !ok {
				panic(fmt.Sprintf("flag is incorrect type %T", f.Value))
			}

			// Do not process hidden flags.
			if typ.Hidden() {
				return
			}

			// Configure the predictor.
			completer.Flags[f.Name] = typ.Predictor()

			// Map any aliases to the flag predictor as well.
			for _, v := range typ.Aliases() {
				completer.Flags[v] = completer.Flags[f.Name]
			}
		})
	}

	// If this is a root command, recurse and build the child completions.
	r, ok := cmd.(*RootCommand)
	if ok {
		for name, fn := range r.Commands {
			instance := fn()

			// Ignore hidden commands from completions.
			if instance.Hidden() {
				continue
			}

			completer.Sub[name] = buildCompleteCommands(instance)
		}
	}

	return completer
}
