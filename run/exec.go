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
	"slices"
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
// exit. For large amounts of output, it's recommended to call Run with a
// streaming writer to avoid having all output in RAM.
//
// Sub-command inherits env variables.
//
// If the command exits with a nonzero status code, an *exec.ExitError will be
// returned.
//
// If the command fails, the error message will include the contents of stdout
// and stderr. This saves boilerplate in the caller.
func Simple(ctx context.Context, args ...string) (stdout, stderr string, _ error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	opts := &Option{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Env:    os.Environ(),
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
func Run(ctx context.Context, opts *Option, args ...string) (exitCode int, _ error) {
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

	// #nosec G204 -- Execution of external commands is the purpose of this package. Inputs must be trusted by the caller.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	cmd.Dir = opts.Cwd
	cmd.Stdin = opts.Stdin

	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	cmd.Env = opts.Env

	if opts.WaitDelay != nil {
		cmd.WaitDelay = *opts.WaitDelay
	} else {
		cmd.WaitDelay = DefaultWaitDelay
	}
	logger.DebugContext(ctx, "set wait delay", "delay", cmd.WaitDelay)

	logger.DebugContext(ctx, "starting command",
		"args", args,
		"options", opts,
	)
	err := cmd.Run() // This blocks until the command exits or context is cancelled

	// Exit code -1 if ProcessState is nil (e.g., start failed)
	exitCode = -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) {
			// TODO, should I allow -1 to be in any position?
			if (len(opts.AllowedExitCodes) > 0 && opts.AllowedExitCodes[0] == -1) || slices.Contains(opts.AllowedExitCodes, exitCode) {
				logger.DebugContext(ctx, "command exited non-zero, but allowed by option", "exit_code", exitCode)
				err = nil
			} else {
				logger.DebugContext(ctx, "command exited non-zero",
					"exit_code", exitCode,
					"error", err,
				)
				err = fmt.Errorf("command %v exited non-zero (%d): %w (context error: %v)\nstdout:\n%s\nstderr:\n%s",
					args, exitCode, err, ctx.Err(), cmd.Stdout, cmd.Stderr) //nolint:errorlint
			}
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
	// AllowNonzeroExitCodes prevents Run from returning error when one of
	// the listed non-zero status codes appear. If AllowedExitCodes[0] == -1,
	// all exit codes are allowed.
	AllowedExitCodes []int

	// Runs command in specified working directory.
	Cwd string
	// Env contains the list of env vars in KEY=VALUE form. Default is no env vars.
	Env []string
	// Sets the cmd's Stdin. Defaults is no Stdin.
	Stdin io.Reader
	// Sets the cmd's Stdout. Default is os.Stdout.
	Stdout io.Writer
	// Sets the cmd's Stderr. Default is os.Stderr.
	Stderr io.Writer
	// Sets a grace period for the command to exit after the context
	// is cancelled before being forcefully killed.
	// Default: DefautWaitDelay. 0 is no wait delay.
	// See exec.Cmd.WaitDelay for more information.
	WaitDelay *time.Duration
}

// TODO: handle process group and cancel semantics.

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

// Environ is a utility to compile environment variables to pass to the child.
// The overridden environment is always added, even if not explicitly
// allowed/denied.
func Environ(osEnv, allowedKeys, deniedKeys, overrideEnv []string) []string {
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
