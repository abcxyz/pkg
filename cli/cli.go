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

// Package cli defines an SDK for building performant and consistent CLIs. All
// commands start with a [RootCommand] which can then accept one or more nested
// subcommands. Subcommands can also be [RootCommand], which creates nested CLIs
// (e.g. "my-tool do the-thing").
//
// The CLI provides opinionated, formatted help output including flag structure.
// It also provides a more integrated experience for defining CLI flags, hiding
// flags, and generating aliases.
//
// To minimize startup times, things are as lazy-loaded as possible. This means
// commands are instantiated only when needed. Most applications will create a
// private global variable that returns the root command:
//
//	var rootCmd = func() cli.Command {
//	  return &cli.RootCommand{
//	    Name:    "my-tool",
//	    Version: "1.2.3",
//	    Commands: map[string]cli.CommandFactory{
//	      "eat": func() cli.Command {
//	        return &EatCommand{}
//	      },
//	      "sleep": func() cli.Command {
//	        return &SleepCommand{}
//	      },
//	    },
//	  }
//	}
//
// This CLI could be invoked via:
//
//	$ my-tool eat
//	$ my-tool sleep
//
// Deeply-nested [RootCommand] behave like nested CLIs:
//
//	var rootCmd = func() cli.Command {
//	  return &cli.RootCommand{
//	    Name:    "my-tool",
//	    Version: "1.2.3",
//	    Commands: map[string]cli.CommandFactory{
//	      "transport": func() cli.Command {
//	        return &cli.RootCommand{
//	          Name:        "transport",
//	          Description: "Subcommands for transportation",
//	          Commands: map[string]cli.CommandFactory{
//	            "bus": func() cli.Command {
//	              return &BusCommand{}
//	            },
//	            "car": func() cli.Command {
//	              return &CarCommand{}
//	            },
//	            "train": func() cli.Command {
//	              return &TrainCommand{}
//	            },
//	          },
//	        }
//	      },
//	    },
//	  }
//	}
//
// This CLI could be invoked via:
//
//	$ my-tool transport bus
//	$ my-tool transport car
//	$ my-tool transport train
package cli
