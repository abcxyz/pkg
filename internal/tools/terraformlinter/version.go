// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package terraformlinter

import "github.com/abcxyz/pkg/buildinfo"

var (
	// Name is the name of the binary. This can be overridden by the build
	// process.
	Name = "terraform-linter"

	// Version is the main package version. This can be overridden by the build
	// process.
	Version = buildinfo.Version()

	// Commit is the git sha. This can be overridden by the build process.
	Commit = buildinfo.Commit()

	// OSArch is the operating system and architecture combination.
	OSArch = buildinfo.OSArch()

	// HumanVersion is the compiled version.
	HumanVersion = Name + " " + Version + " (" + Commit + ", " + OSArch + ")"
)
