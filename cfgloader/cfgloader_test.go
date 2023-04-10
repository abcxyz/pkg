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

package cfgloader

import (
	"context"
	"fmt"
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

type fakeCfg struct {
	StrVal string `yaml:"str_val,omitempty" env:"STR_VAL,overwrite,default=foo"`
	NumVal int    `yaml:"num_val,omitempty" env:"NUM_VAL,overwrite,default=1"`
}

func (c *fakeCfg) Validate() error {
	if c.StrVal == "fail_me" {
		return fmt.Errorf("StrVal cannot be 'fail_me'")
	}
	return nil
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		input   *fakeCfg
		want    *fakeCfg
		wantErr string
	}{
		{
			name:  "no_option_set_default",
			opts:  []Option{},
			input: &fakeCfg{},
			want: &fakeCfg{
				StrVal: "foo",
				NumVal: 1,
			},
		},
		{
			name: "with_yaml",
			opts: []Option{WithYAML([]byte(`str_val: bar
num_val: 2`))},
			input: &fakeCfg{},
			want: &fakeCfg{
				StrVal: "bar",
				NumVal: 2,
			},
		},
		{
			name: "with_prefix_lookuper",
			opts: []Option{
				WithEnvPrefix("TEST_"),
				WithLookuper(envconfig.MapLookuper(map[string]string{
					"TEST_STR_VAL": "bar",
					"TEST_NUM_VAL": "2",
				})),
			},
			input: &fakeCfg{},
			want: &fakeCfg{
				StrVal: "bar",
				NumVal: 2,
			},
		},
		{
			name: "config_already_has_values",
			opts: []Option{},
			input: &fakeCfg{
				StrVal: "bar",
			},
			want: &fakeCfg{
				StrVal: "bar",
				NumVal: 1,
			},
		},
		{
			name: "validation_failure",
			opts: []Option{},
			input: &fakeCfg{
				StrVal: "fail_me",
			},
			wantErr: "StrVal cannot be 'fail_me'",
		},
		{
			name: "nil_lookuper",
			opts: []Option{
				WithLookuper(nil),
			},
			input: &fakeCfg{},
			want: &fakeCfg{
				StrVal: "foo",
				NumVal: 1,
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.input
			err := Load(context.Background(), got, tc.opts...)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Load got unexpected err: %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Loaded config (-want,+got):\n%s", diff)
			}
		})
	}
}
