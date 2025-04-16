// Copyright 2025 The Authors (see AUTHORS file)
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
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// ExecConfig are the inputs for a run operation.
type Option struct {
	stdin      io.Writer
	stdout     io.Writer
	stderr     io.Writer
	workingDir string

	// allowedEnvKeys and deniedEnvKeys respectively define an allow/deny list of
	// patterns for environment variable keys. Keys are matched using
	// [filepath.Match].
	allowedEnvKeys []string
	deniedEnvKeys  []string

	// overrideEnvVars are the environment variables to inject into the child
	// process, no matter what the user configured. These take precedence over all
	// other configurables.
	overrideEnvVars []string
}

// Run runs the command provided by args, using the options in opts. Error
// is only returned if something external to the process fails (timeout,
// command not found, ect). It is up to user to check status code.
//
// If the incoming context doesn't already have a timeout, then a default
// timeout will be added (see DefaultRunTimeout).
//
// opts may be nil if no special options are needed.
//
// This doesn't execute a shell (unless of course command is the name of a shell
// binary).

// TODO: what are the ergonomics like with opts not being varadic operator? I'm not sure I'm happy with this
func Run(ctx context.Context, opts []*Option, command string, args ...string) (exitCode int, _ error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRunTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, command, args...) //nolint:gosec // run'ing the input args is fundamentally the whole point
	setSysProcAttr(cmd)
	setCancel(cmd)
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
