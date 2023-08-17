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

// Package buildinfo provides functions for setting build information for CLIs
// and programs. Since CLIs and programs can be compiled, downloaded as a
// binary, or installed via `go install`, there's some nuanced logic in getting
// these values correct across all instances.
//
// Consumers are encouraged to create an internal package in their module at
// "internal/version" with the following contents:
//
//	var (
//	  Name = "my-program"
//
//	  Version = buildinfo.Version()
//
//	  Commit = buildinfo.Commit()
//
//	  OSArch = buildinfo.OSArch()
//
//	  HumanVersion = Name + " " + Version + " (" + Commit + ", " + OSArch + ")"
//	)
//
// These variables can then be referenced throughout the program. The values can
// still be overridden with LDFLAGS (which will take precedent over any values
// defined here).
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

// Version attempts to read the module version injected by the compiler. If no
// information is present, it returns "source".
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" {
			return v // e.g. "v0.0.1-alpha6.0.20230815191505-8628f8201363"
		}
	}

	return "source"
}

// Commit returns the VCS information, specifically the revision. Since most of
// our modules use Git, this is the Git SHA. If no SHA exists (e.g. outside of a
// repo), it returns "HEAD".
func Commit() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "HEAD"
}

// OSArch returns the denormalized operating system and architecture, separated
// by a slash (e.g. "linux/amd64").
func OSArch() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
