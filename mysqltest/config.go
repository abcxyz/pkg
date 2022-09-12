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

import "log"

// This file implements the "functional options" pattern.

type config struct {
	killAfterSec   int // This is in integer seconds because that's what Docker takes.
	mySQLVersion   string
	progressLogger Logger
}

type Logger interface {
	Printf(fmtStr string, args ...any)
}

func makeDefaultConfig() *config {
	return &config{
		killAfterSec:   10 * 60,
		mySQLVersion:   "5.7",
		progressLogger: &stdlibLogger{},
	}
}

func buildConfig(opts ...Option) *config {
	config := makeDefaultConfig()
	for _, opt := range opts {
		config = opt(config)
	}
	return config
}

type Option func(*config) *config

// WithKillAfterSeconds is an option that overrides the default time period after which the mysql docker
// container will kill itself.
//
// Containers might bypass the normal clean shutdown logic if the test terminates abnormally, such
// as when ctrl-C is pressed during a test. Therefore we instruct the container to kill itself after
// a while. The duration must be longer than longest test that uses MySQL. There's no harm in
// leaving lots of extra time.
func WithKillAfterSeconds(seconds int) Option {
	return func(c *config) *config {
		c.killAfterSec = seconds
		return c
	}
}

// WithVersion chooses a MySQL server version. This overrides the default MySQL server version.
func WithVersion(v string) Option {
	return func(c *config) *config {
		c.mySQLVersion = v
		return c
	}
}

// stdlibLogger is the default implementation of the Logger interface that calls log.Printf.
type stdlibLogger struct{}

func (s *stdlibLogger) Printf(fmtStr string, args ...any) {
	log.Printf(fmtStr, args...)
}
