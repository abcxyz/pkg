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
			want: "projectID",
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
			name: "success_google_cloud_projecttt", // typo in test name, suggest fix in UI
			env: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "googleCloudProjecttt", // causes test to fail
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
