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

package containertest

import (
	"log"
)

// This file implements the "functional options" pattern.

type config struct {
	killAfterSec   int // This is in integer seconds because that's what Docker takes.
	service        Service
	progressLogger TestLogger
}

// TestLogger allows the caller to optionally provide a custom logger for printing status updates about
// service startup progress. The default is to use the go "log" package.
// testing.TB satisfies TestLogger, and is usually what you want to put here.
type TestLogger interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

func makeDefaultConfig(service Service) *config {
	return &config{
		killAfterSec:   10 * 60,
		service:        service,
		progressLogger: &stdlibLogger{},
	}
}

func buildConfig(service Service, opts ...Option) *config {
	config := makeDefaultConfig(service)
	for _, opt := range opts {
		config = opt(config)
	}
	return config
}

// Option sets a configuration option for this package. Users should not implement these functions,
// they should use one of the With* functions.
type Option func(*config) *config

// WithKillAfterSeconds is an option that overrides the default time period after which the docker
// container will kill itself.
//
// Containers might bypass the normal clean shutdown logic if the test terminates abnormally, such
// as when ctrl-C is pressed during a test. Therefore we instruct the container to kill itself after
// a while. The duration must be longer than longest test that uses the container. There's no harm in
// leaving lots of extra time.
func WithKillAfterSeconds(seconds int) Option {
	return func(c *config) *config {
		c.killAfterSec = seconds
		return c
	}
}

// WithLogger overrides the default logger. This logger will receive messages about service startup
// progress. The default is to use the go "log" package.
func WithLogger(l TestLogger) Option {
	return func(c *config) *config {
		c.progressLogger = l
		return c
	}
}

// stdlibLogger is the default implementation of the TestLogger interface that calls log.Logf.
type stdlibLogger struct{}

func (s *stdlibLogger) Logf(fmtStr string, args ...any) {
	log.Printf(fmtStr, args...)
}

func (s *stdlibLogger) Log(args ...any) {
	log.Print(args...)
}
