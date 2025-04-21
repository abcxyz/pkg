// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package run provides enhanced functionality for executing external commands,
// offering control over environment variables, process attributes, I/O,
// timeouts, and integrated logging, using a functional options pattern.
package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcxyz/pkg/logging" // Import the specific logging package
)

// DefaultRunTimeout is how long commands wait if the context doesn't have a deadline.
const DefaultRunTimeout = time.Minute

// DefaultWaitDelay is how long after context cancellation processes are actually killed
// if another value isn't set. See exec.Cmd.WaitDelay for more information.
const DefaultWaitDelay = time.Second

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
	opts := []*Option{
		WithStdout(&stdoutBuf),
		WithStderr(&stderrBuf),
	}

	_, err := Run(ctx, opts, args...)

	return stdoutBuf.String(), stderrBuf.String(), err
}

// Run executes the command specified by args, applying configurations from opts.
//
// By default, a non-zero exit code results in an error (*exec.ExitError),
// unless the AllowNonzeroExit option is used.
// The error message includes stdout/stderr content if they are captured to buffers.
//
// If the context doesn't have a deadline, DefaultRunTimeout is applied.
//
// The input args must have len>=1. opts may be nil if no special options are
// needed.
//
// This doesn't execute a shell (unless of course args[0] is the name of a shell
// binary).
func Run(ctx context.Context, opts []*Option, args ...string) (exitCode int, _ error) {
	logger := logging.FromContext(ctx)

	if len(args) == 0 {
		logger.DebugContext(ctx, "run called with no arguments")
		return -1, errors.New("run: must provide at least one argument (the command)")
	}

	// Apply default timeout if none exists on the context.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRunTimeout)
		defer cancel() // Ensure the derived context is cancelled
	}

	compiledOpts := compileOpts(opts)

	// #nosec G204 -- Execution of external commands is the purpose of this package. Inputs must be trusted by the caller.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	cmd.Dir = compiledOpts.cwd
	cmd.Stdin = compiledOpts.stdin
	// Handle stdout/stderr, capturing output if writers are bytes.Buffer for error reporting.
	var stdoutBuf, stderrBuf *bytes.Buffer
	if compiledOpts.stdout != nil {
		cmd.Stdout = compiledOpts.stdout
		if bb, ok := compiledOpts.stdout.(*bytes.Buffer); ok {
			stdoutBuf = bb
		}
	} else {
		cmd.Stdout = os.Stdout
	}
	if compiledOpts.stderr != nil {
		cmd.Stderr = compiledOpts.stderr
		if bb, ok := compiledOpts.stderr.(*bytes.Buffer); ok {
			stderrBuf = bb
		}
	} else {
		cmd.Stderr = os.Stderr
	}

	useCustomEnv := len(compiledOpts.allowedEnvKeys) > 0 || len(compiledOpts.deniedEnvKeys) > 0 || len(compiledOpts.additionalEnv) > 0
	if useCustomEnv {
		currentEnv := os.Environ()
		logger.DebugContext(ctx, "calculating custom environment",
			"inherited_count", len(currentEnv),
			"allowed_patterns", compiledOpts.allowedEnvKeys,
			"denied_patterns", compiledOpts.deniedEnvKeys,
			"additional_vars_count", len(compiledOpts.additionalEnv))
		env := environ(currentEnv, compiledOpts.allowedEnvKeys, compiledOpts.deniedEnvKeys, compiledOpts.additionalEnv)
		logger.DebugContext(ctx, "computed environment", "env", env)
		cmd.Env = env
	} else {
		logger.DebugContext(ctx, "using inherited environment")
	}

	if compiledOpts.waitDelay != nil {
		cmd.WaitDelay = *compiledOpts.waitDelay
	} else {
		cmd.WaitDelay = DefaultWaitDelay
	}
	logger.DebugContext(ctx, "set wait delay", "delay", cmd.WaitDelay)

	logger.DebugContext(ctx, "starting command", "args", args, "options", compiledOpts)
	err := cmd.Run() // This blocks until the command exits or context is cancelled

	// Exit code -1 if ProcessState is nil (e.g., start failed)
	exitCode = -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		var exitErr *exec.ExitError
		isExitError := errors.As(err, &exitErr)

		if isExitError && compiledOpts.allowNonZeroExit {
			logger.DebugContext(ctx, "command exited non-zero, but allowed by option", "exit_code", exitCode)
			err = nil
		} else if isExitError {
			stdoutContent := "[stdout not captured]"
			if stdoutBuf != nil {
				stdoutContent = stdoutBuf.String()
			}
			stderrContent := "[stderr not captured]"
			if stderrBuf != nil {
				stderrContent = stderrBuf.String()
			}
			err = fmt.Errorf("command %v exited non-zero (%d): %w (context error: %v)\nstdout:\n%s\nstderr:\n%s",
				args, exitCode, err, ctx.Err(), stdoutContent, stderrContent)
			logger.DebugContext(ctx, "command exited non-zero", "exit_code", exitCode, "error", err)
		} else {
			// It's an actual execution error (e.g., command not found, context cancelled early)
			stdoutContent := "[stdout not captured]"
			if stdoutBuf != nil {
				stdoutContent = stdoutBuf.String()
			}
			stderrContent := "[stderr not captured]"
			if stderrBuf != nil {
				stderrContent = stderrBuf.String()
			}
			err = fmt.Errorf("command %v failed: %w (context error: %v)\nstdout:\n%s\nstderr:\n%s",
				args, err, ctx.Err(), stdoutContent, stderrContent)
			logger.DebugContext(ctx, "command failed with execution error", "exit_code", exitCode, "error", err)
		}
	} else {
		// Command ran successfully (exit code 0)
		logger.DebugContext(ctx, "command finished successfully", "exit_code", exitCode)
	}

	return exitCode, err
}

// Option implements the functional options pattern for [Run].
// It holds all configurable settings for executing a command.
type Option struct {
	// Basic execution control
	allowNonZeroExit bool
	cwd              string
	stdin            io.Reader
	stdout           io.Writer
	stderr           io.Writer
	waitDelay        *time.Duration

	allowedEnvKeys []string
	deniedEnvKeys  []string
	additionalEnv  []string
}

// AllowNonzeroExit prevents Run from returning an error
// when the command exits with a non-zero status code.
func AllowNonzeroExit() *Option {
	return &Option{allowNonZeroExit: true}
}

// WithStdin provides the given reader as the command's
// standard input.
func WithStdin(stdin io.Reader) *Option {
	return &Option{stdin: stdin}
}

// WithStdinStr is a convenience option that uses the given string as the
// command's standard input.
func WithStdinStr(stdin string) *Option {
	return WithStdin(bytes.NewBufferString(stdin))
}

// WithStdout directs the command's standard output
// to the given writer.
func WithStdout(stdout io.Writer) *Option {
	return &Option{stdout: stdout}
}

// WithStderr directs the command's standard error
// to the given writer.
func WithStderr(stderr io.Writer) *Option {
	return &Option{stderr: stderr}
}

// WithCwd runs the command in the specified working directory.
func WithCwd(cwd string) *Option {
	return &Option{cwd: cwd}
}

// WithFilteredEnv filters the inherited environment variables.
// allowed is a list of glob patterns for keys to keep (empty means allow all initially).
// denied is a list of glob patterns for keys to remove (takes precedence).
func WithFilteredEnv(allowed, denied []string) *Option {
	return &Option{allowedEnvKeys: allowed, deniedEnvKeys: denied}
}

// WithAdditionalEnv returns an option that adds or overrides environment variables.
// vars is a slice of strings in "KEY=VALUE" format. Can be added multiple times.
func WithAdditionalEnv(vars []string) *Option {
	return &Option{additionalEnv: vars}
}

// WithWaitDelay sets a grace period for the command to exit after the context
// is cancelled before being forcefully killed.
// Default: DefautWaitDelay.
// See exec.Cmd.WaitDelay for more information.
func WithWaitDelay(d time.Duration) *Option {
	return &Option{waitDelay: &d}
}

// TODO: handle process group and cancel semantics.
// WithProcessGroup returns an option that attempts to run the command in a new
// process group (primarily effective on Unix-like systems).
//func WithProcessGroup(set bool) *Option {
//	return &Option{setProcessGroup: set}
//}

// The last option specified for most fields takes precedence.
// additionalEnv is appended across options.
func compileOpts(opts []*Option) *Option {
	out := Option{}

	for _, opt := range opts {
		if opt.allowNonZeroExit {
			out.allowNonZeroExit = true
		}
		if opt.cwd != "" {
			out.cwd = opt.cwd
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
		if opt.waitDelay != nil {
			out.waitDelay = opt.waitDelay
		}
		if opt.allowedEnvKeys != nil {
			out.allowedEnvKeys = opt.allowedEnvKeys
		}
		if opt.deniedEnvKeys != nil {
			out.deniedEnvKeys = opt.deniedEnvKeys
		}
		if len(opt.additionalEnv) > 0 {
			out.additionalEnv = append(out.additionalEnv, opt.additionalEnv...)
		}
	}
	return &out
}

// anyGlobMatch checks if string s matches any of the glob patterns.
func anyGlobMatch(s string, patterns []string) bool {
	for _, p := range patterns {
		if p == "*" || p == ".*" { // Common fast-path wildcards
			return true
		}
	}
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, s); ok { // Ignore Match errors
			return true
		}
	}
	return false
}

// environ compiles the appropriate environment to pass to the child process.
// The overridden environment is always added, even if not explicitly
// allowed/denied.
func environ(osEnv, allowedKeys, deniedKeys, overrideEnv []string) []string {
	finalEnv := make([]string, 0, len(osEnv)+len(overrideEnv))

	// Select keys that match the allow filter (if given) but not the deny filter.
	for _, v := range osEnv {
		k := strings.SplitN(v, "=", 2)[0]
		if (len(allowedKeys) == 0 || anyGlobMatch(k, allowedKeys)) &&
			!anyGlobMatch(k, deniedKeys) {
			finalEnv = append(finalEnv, v)
		}
	}

	// Add overrides at the end, after any filtering. os/exec has "last wins"
	// semantics, so appending is an overriding action.
	finalEnv = append(finalEnv, overrideEnv...)

	return finalEnv
}
