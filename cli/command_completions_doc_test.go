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

package cli_test

import (
	"context"
	"strconv"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

type SingCommand struct {
	cli.BaseCommand

	flagSong string
	flagFade time.Duration
	flagNow  int64
}

func (c *SingCommand) Desc() string {
	return "Sings a song"
}

func (c *SingCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options] PATH

  Sings the given song at the audio file.
`
}

// PredictArgs is an optional interface which will predict argument values. If
// omitted, no arguments are suggested.
func (c *SingCommand) PredictArgs() complete.Predictor {
	return predict.Files("*.mp3")
}

func (c *SingCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	f := set.NewSection("Song options")

	f.StringVar(&cli.StringVar{
		Name:    "song",
		Aliases: []string{"s"},
		Example: "Itsy Bitsy Spider",
		Target:  &c.flagSong,
		Predict: predict.Set{"Happy Birthday", "Twinkly Twinkle Little Star"},
		Usage:   "Name of the song to play.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "fade",
		Example: "5s",
		Default: 5 * time.Second,
		Target:  &c.flagFade,
		Predict: predict.Set{"1s", "5s", "10s"},
		Usage:   "Duration to fade audio tracks.",
	})

	f.Int64Var(&cli.Int64Var{
		Name:    "now",
		Example: "10404929",
		Default: time.Now().Unix(),
		Target:  &c.flagNow,
		Predict: complete.PredictFunc(func(prefix string) []string {
			return []string{strconv.FormatInt(time.Now().Unix(), 10)}
		}),
		Usage: "Curring timestamp, in unix seconds.",
	})

	return set
}

func (c *SingCommand) Run(ctx context.Context, args []string) error {
	return nil
}

func Example_completions() {
	// The example is above, demonstrating various ways to define completions. To
	// see even more ways, look at the examples at
	// github.com/posener/complete/tree/master.
	//
	// Since completions are a shell function, users must add something to their
	// shell configuration (e.g. .bashrc, .zshrc). The easiest method to install
	// completions is to allow the binary to do it. It will detect the shell and
	// insert the correct code into the user's shell profile. Instruct users to
	// run the following command:
	//
	//     COMP_INSTALL=1 COMP_YES=1 my-cli
	//
	// This will automatically install shell completions. To uninstall
	// completions, instruct users to run:
	//
	//     COMP_UNINSTALL=1 COMP_YES=1 my-cli
	//
	// This will automatically uninstall the completions.
	//
	// If users want to install the completions manually, you will need to provide
	// them shell-specific instructions. The setup usually requires adding the
	// following lines:
	//
	//     autoload -U +X bashcompinit && bashcompinit
	//     complete -o nospace -C /full/path/to/my-cli my-cli
	//
	// Of note:
	//
	//   1. You must use the full filepath to the CLI binary
	//   2. The argument to the CLI binary is itself
}
