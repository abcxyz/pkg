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

package timeutil

import (
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input time.Duration
		exp   string
	}{
		{
			name:  "zero",
			input: 0,
			exp:   "0s",
		},
		{
			name:  "rounds_seconds",
			input: 5500 * time.Millisecond,
			exp:   "6s",
		},
		{
			name:  "zero_seconds",
			input: 90 * time.Minute,
			exp:   "1h30m",
		},
		{
			name:  "zero_minutes",
			input: 1*time.Hour + 4*time.Second,
			exp:   "1h4s",
		},
		{
			name:  "zero_minutes_seconds",
			input: 1 * time.Hour,
			exp:   "1h",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, want := HumanDuration(tc.input), tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}
