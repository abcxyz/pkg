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

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/abcxyz/pkg/internal/tools/terraformlinter"
)

const lintCommandHelp = `
Lint a set of files or directory
EXAMPLES
  terraform-linter <file1> <file2> <directory>
FLAGS
`

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain() error {
	f := flag.NewFlagSet("", flag.ExitOnError)
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", strings.TrimSpace(lintCommandHelp))
		f.PrintDefaults()
	}
	showVersion := f.Bool("version", false, "display version information")

	if err := f.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	if *showVersion {
		fmt.Fprintln(os.Stderr, terraformlinter.HumanVersion)
		return nil
	}

	// The linter needs at least one file or directory
	args := f.Args()
	if got := len(args); got < 1 {
		return fmt.Errorf("expected at least one argument, got %d", got)
	}

	if err := terraformlinter.RunLinter(args); err != nil {
		return fmt.Errorf("error running linter %w", err)
	}
	return nil
}
