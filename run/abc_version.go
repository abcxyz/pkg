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

package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// DefaultRunTimeout is how long we'll wait for commands to run in the case
// where the context doesn't already have a timeout. This was chosen
// arbitrarily.
const DefaultRunTimeout = time.Minute

// Simple is a wrapper around [Run] that captures stdout and stderr as strings.
// This is intended to be used for commands that run non-interactively then
// exit.
//
// If the command exits with a nonzero status code, an *exec.ExitError will be
// returned.
//
// If the command fails, the error message will include the contents of stdout
// and stderr. This saves boilerplate in the caller.
func Simple(ctx context.Context, args ...string) (stdout, stderr string, _ error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	_, err := Run(ctx, []*Option{
		WithStdout(&stdoutBuf),
		WithStderr(&stderrBuf),
	}, args...)
	return stdoutBuf.String(), stderrBuf.String(), err
}

// Run runs the command provided by args, using the options in opts. By default,
// if the command returns a nonzero exit code, an error is returned, but this
// behavior may be overridden by the AllowNonzeroExit option.
//
// If the incoming context doesn't already have a timeout, then a default
// timeout will be added (see DefaultRunTimeout).
//
// If the command fails, the error message will include the contents of stdout
// and stderr. This saves boilerplate in the caller.
//
// The input args must have len>=1. opts may be nil if no special options are
// needed.
//
// This doesn't execute a shell (unless of course args[0] is the name of a shell
// binary).
func Run(ctx context.Context, opts []*Option, args ...string) (exitCode int, _ error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRunTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec // run'ing the input args is fundamentally the whole point

	// any of these can be nil
	compiledOpts := compileOpts(opts)
	cmd.Stdout = compiledOpts.stdout
	cmd.Stderr = compiledOpts.stderr
	cmd.Stdin = compiledOpts.stdin
	cmd.Dir = compiledOpts.cwd

	err := cmd.Run()
	if err != nil {
		// Don't return error if both (a) the caller indicated they're OK with a
		// nonzero exit code and (b) the error is of a type that means the only
		// problem was a nonzero exit code.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && compiledOpts.allowNonZeroExit {
			err = nil
		} else {
			err = fmt.Errorf(`run of %v failed: error was "%w", context error was "%w"\nstdout: %s\nstderr: %s`,
				args, err, ctx.Err(), cmd.Stdout, cmd.Stderr)
		}
	}
	return cmd.ProcessState.ExitCode(), err
}

// Many calls [Simple] for each command in args. If any command returns error,
// then no further commands will be run, and that error will be returned. For
// any commands that were actually executed (not aborted by a previous error),
// their stdout and stderr will be returned. It's guaranteed that
// len(stdouts)==len(stderrs).
func Many(ctx context.Context, args ...[]string) (stdouts, stderrs []string, _ error) {
	for _, cmd := range args {
		stdout, stderr, err := Simple(ctx, cmd...)
		stdouts = append(stdouts, stdout)
		stderrs = append(stderrs, stderr)
		if err != nil {
			return stdouts, stderrs, err
		}
	}
	return stdouts, stderrs, nil
}

// Option implements the functional options pattern for [Run].
type Option struct {
	allowNonZeroExit bool
	cwd              string
	stdin            io.Reader
	stdout           io.Writer
	stderr           io.Writer
}

// AllowNonzeroExit is an option that will NOT treat a nonzero exit code from
// the command as an error (so [Run] won't return error). The default behavior
// of [Run] is that if a command exits with a nonzero status code, then that
// becomes a Go error.
func AllowNonzeroExit() *Option {
	return &Option{allowNonZeroExit: true}
}

// WithStdin passes the given reader as the command's standard input.
func WithStdin(stdin io.Reader) *Option {
	return &Option{stdin: stdin}
}

// WithStdinStr is a convenient wrapper around WithStdin that passes the given
// string as the command's standard input.
func WithStdinStr(stdin string) *Option {
	return WithStdin(bytes.NewBufferString(stdin))
}

// WithStdout writes the command's standard output to the given writer.
func WithStdout(stdout io.Writer) *Option {
	return &Option{stdout: stdout}
}

// WithStderr writes the command's standard error to the given writer.
func WithStderr(stderr io.Writer) *Option {
	return &Option{stderr: stderr}
}

// WithCwd runs the command in the given working directory.
func WithCwd(cwd string) *Option {
	return &Option{cwd: cwd}
}

func compileOpts(opts []*Option) *Option {
	var out Option
	for _, opt := range opts {
		if opt.allowNonZeroExit {
			out.allowNonZeroExit = true
		}
		if opt.stdin != nil {
			out.stdin = opt.stdin
		}
		if opt.stdout != nil {
			out.stdout = opt.stdout
		}
		if opt.stderr != nil {
			out.stderr = opt.stderr
		}
		if opt.cwd != "" {
			out.cwd = opt.cwd
		}
	}

	return &out
}
