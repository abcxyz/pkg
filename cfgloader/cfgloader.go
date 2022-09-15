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

// Package cfgloader provides common functionality to load configs.
package cfgloader

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

// Validatable is the interface to validate a config.
type Validatable interface {
	Validate() error
}

type options struct {
	yamlBytes []byte
	envPrefix string
	lookuper  envconfig.Lookuper
}

// Option is the config loading option type.
type Option func(*options) *options

// WithYAML instructs the loader to load config from the given yaml bytes.
func WithYAML(b []byte) Option {
	return func(o *options) *options {
		o.yamlBytes = b
		return o
	}
}

// WithEnvPrefix instructs the loader to load config from env vars with the given prefix.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) *options {
		o.envPrefix = prefix
		return o
	}
}

// WithLookuper instructs the loader to use the giver lookuper to find config values.
func WithLookuper(lookuper envconfig.Lookuper) Option {
	return func(o *options) *options {
		o.lookuper = lookuper
		return o
	}
}

// Load loads config into the given config value. The loading order is:
//
//  1. The existing values in the given config.
//  2. Unmarshaled yaml bytes
//  3. Env vars
//
// The values loaded later will overwrite previously loaded values.
//
// The given config type must have the yaml tags to load from yaml bytes.
// It must have the [env tag] to load from env vars. E.g.
//
//	type Cfg struct {
//		StrVal string `yaml:"str_val,omitempty" env:"STR_VAL,overwrite,default=foo"`
//		NumVal int    `yaml:"num_val,omitempty" env:"NUM_VAL,overwrite,default=1"`
//	}
//
// [env tag]: https://github.com/sethvargo/go-envconfig
func Load(ctx context.Context, cfg any, opt ...Option) error {
	opts := &options{
		// Default to OS lookuper.
		lookuper: envconfig.OsLookuper(),
	}
	for _, o := range opt {
		opts = o(opts)
	}

	// Load from yaml bytes first if provided.
	if opts.yamlBytes != nil {
		if err := yaml.Unmarshal(opts.yamlBytes, cfg); err != nil {
			return fmt.Errorf("failed to unmarshal yaml bytes: %w", err)
		}
	}

	lookuper := opts.lookuper
	if opts.envPrefix != "" {
		lookuper = envconfig.PrefixLookuper(opts.envPrefix, lookuper)
	}

	if err := envconfig.ProcessWith(ctx, cfg, lookuper); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	v, ok := cfg.(Validatable)
	if ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("config invalid: %w", err)
		}
	}

	return nil
}
