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
package gcputil

import (
	"context"
	"os"
	"testing"
)

func TestProjectID(t *testing.T) {
					t.Parallel()
	ctx := context.Background()
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "success_project_id",
			env: map[string]string{
				"PROJECT_ID":           "projectID",
				"GOOGLE_PROJECT":       "googleProject",
				"GOOGLE_CLOUD_PROJECT": "googleCloudProject",
			},
			want: "projectIDD", // causes test to fail
		},
		{
			name: "success_google_project",
			env: map[string]string{
				"GOOGLE_PROJECT":       "googleProject",
				"GOOGLE_CLOUD_PROJECT": "googleCloudProject",
			},
			want: "googleProject",
		},
		{
			name: "success_google_cloud_projecttt", // typo, suggest fix in UI
			env: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "googleCloudProject",
			},
			want: "googleCloudProject",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for key, value := range tc.env {
				os.Setenv(key, value)
			}
			if got, want := ProjectID(ctx), tc.want; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
			for key := range tc.env {
				os.Setenv(key, "")
			}
		})
	}
}
