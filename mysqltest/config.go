// Copyright 2022 The Authors (see AUTHORS file)
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

package mysqltest

// This file implements the "functional options" pattern.

type config struct {
	killAfterSec int
	mySQLVersion string
}

var defaultConfig = config{
	killAfterSec: 10 * 60,
	mySQLVersion: "5.7",
}

func buildConfig(opts ...Option) *config {
	out := defaultConfig // shallow copy is good enough. There are no pointers.
	for _, opt := range opts {
		opt(&out)
	}
	return &out
}

type Option func(*config)

// KillAfterSeconds is an option that overrides the default time period after which the mysql docker
// container will kill itself.
func KillAfterSeconds(seconds int) Option {
	return func(c *config) {
		c.killAfterSec = seconds
	}
}

// Version is an option that overrides the default MySQL server version.
func Version(v string) Option {
	return func(c *config) {
		c.mySQLVersion = v
	}
}
