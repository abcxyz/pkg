// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformlinter

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTerraformLinter_FindViolations(t *testing.T) {
	t.Parallel()

	testNoSpecialAttributes := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyForEachCorrect := `
	resource "google_project_service" "run_api" {  
		for_each = toset(["name"])

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyForEachMissingNewLine := `
	resource "google_project_service" "run_api" {  
		for_each = toset(["name"])
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyForEachMid := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		for_each = toset(["name"])
		disable_on_destroy = true
	}
	`
	testOnlyForEachEnd := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		for_each = toset(["name"])
	}
	`
	testOnlyCountCorrect := `
	resource "google_project_service" "run_api" {  
		count = 3

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyCountMissingNewLine := `
	resource "google_project_service" "run_api" {  
		count = 3
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyCountMid := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		count = 3
		disable_on_destroy = true
	}
	`
	testOnlyCountEnd := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		count = 3
	}
	`
	testOnlyProviderCorrect := `
	resource "google_project_service" "run_api" {  
		provider = "some_provider"

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyProviderMissingNewLine := `
	resource "google_project_service" "run_api" {  
		provider = "some_provider"
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testOnlyProviderMid := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		provider = "some_provider"
		disable_on_destroy = true
	}
	`
	testOnlyProviderEnd := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		provider = "some_provider"
	}
	`
	testProjectCorrectNoMeta := `
	resource "google_project_service" "run_api" {  
		project = "some_project_id"

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testProjectCorrectMeta := `
	resource "google_project_service" "run_api" {  
		for_each = toset(["name"]) 

		project = "some_project_id"

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testProjectMissingNewLine := `
	resource "google_project_service" "run_api" {  
		project = "some_project_id"
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testProjectOutOfOrder := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		project = "some_project_id"
		disable_on_destroy = true
	}
	`
	testDependsOnCorrect := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		depends_on = [
			"something"
		]
	}
	`
	testDependsOnOutOfOrder := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		depends_on = [
			"something"
		]
		disable_on_destroy = true
	}
	`
	testLifecycleCorrect := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		lifecycle = {
			prevent_destroy = true
		}
	}
	`
	testLifecycleOutOfOrder := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		lifecycle = {
			prevent_destroy = true
		}
		disable_on_destroy = true
	}
	`
	testTrailingMixCorrect := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		depends_on = [
			"something"
		]
		lifecycle = {
			prevent_destroy = true
		}
	}
	`
	testTrailingMixOutOfOrder := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		disable_on_destroy = true
		lifecycle = {
			prevent_destroy = true
		}
		depends_on = [
			"something"
		]
	}
	`
	testSourceCorrect := `
	resource "google_project_service" "run_api" {  
		source = "http://somerepo"

		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testSourceMissingNewLine := `
	resource "google_project_service" "run_api" {  
		source = "http://somerepo"
		service = "run.googleapis.com"  
		disable_on_destroy = true
	}
	`
	testSourceOutOfOrder := `
	resource "google_project_service" "run_api" {  
		service = "run.googleapis.com"  
		source = "http://somerepo"
		disable_on_destroy = true
	}
	`
	testAllCorrect := `
	resource "google_project_service" "run_api" {  
		for_each = toset(["name"])
		provider = "someprovider"

		project = "pid"
		project_id = "pid"
		folder = "fid"
		organization = "abcxyz"

		service = "run.googleapis.com"  
		disable_on_destroy = true

		depends_on = [
			"something"
		]
		lifecycle = {
			prevent_destroy = true
		}
	}
	`
	testMixedOutOfOrder := `
	resource "google_project_service" "run_api" {  
		folder = "fid"
		provider = "someprovider"
		project = "pid"
		for_each = toset(["name"])
		project_id = "pid"
		service = "run.googleapis.com"  
		lifecycle = {
			prevent_destroy = true
		}
		organization = "abcxyz"
		disable_on_destroy = true
		depends_on = [
			"something"
		]
	}
	`

	cases := []struct {
		name      string
		content   string
		filename  string
		expect    []*ViolationInstance
		wantError bool
	}{
		{
			name:      "No special attributes",
			content:   testNoSpecialAttributes,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:      "Test only for_each - correct",
			content:   testOnlyForEachCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test only for_each - missing newline",
			content:  testOnlyForEachMissingNewLine,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:     "Test for_each mid - violation",
			content:  testOnlyForEachMid,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrForEach),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:     "Test for_each end - violation",
			content:  testOnlyForEachEnd,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrForEach),
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:      "Test only count - correct",
			content:   testOnlyCountCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test only count - missing newline",
			content:  testOnlyCountMissingNewLine,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:     "Test count mid - violation",
			content:  testOnlyCountMid,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrCount),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:     "Test count end - violation",
			content:  testOnlyCountEnd,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrCount),
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:      "Test only provider - correct",
			content:   testOnlyProviderCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test only provider - missing newline",
			content:  testOnlyProviderMissingNewLine,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},

		{
			name:     "Test provider mid - violation",
			content:  testOnlyProviderMid,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrProvider),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:     "Test provider end - violation",
			content:  testOnlyProviderEnd,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrProvider),
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:      "Test project correct no meta block",
			content:   testProjectCorrectNoMeta,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:      "Test project correct meta block",
			content:   testProjectCorrectMeta,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test project missing newline",
			content:  testProjectMissingNewLine,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: errorProviderNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:     "Test project out of order",
			content:  testProjectOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorProviderAttributes, attrProviderProject),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorProviderNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:      "Test depends_on correct",
			content:   testDependsOnCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test depends_on out of order",
			content:  testDependsOnOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorTrailingMetaBlockAttribute, attrDependsOn),
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:      "Test lifecycle correct",
			content:   testLifecycleCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test lifecycle out of order",
			content:  testLifecycleOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:      "Test trailing mix correct",
			content:   testTrailingMixCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test trailing mix out of order",
			content:  testTrailingMixOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          5,
				},
				{
					ViolationType: fmt.Sprintf(errorTrailingMetaBlockAttribute, attrDependsOn),
					Path:          "/test/test.tf",
					Line:          8,
				},
			},
			wantError: false,
		},
		{
			name:      "Test source correct",
			content:   testSourceCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test source missing newline",
			content:  testSourceMissingNewLine,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name:     "Test source out of order",
			content:  testSourceOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrSource),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name:      "Test all correct",
			content:   testAllCorrect,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name:     "Test mixed out of order",
			content:  testMixedOutOfOrder,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrProvider),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
				{
					ViolationType: fmt.Sprintf(errorLeadingMetaBlockAttribute, attrForEach),
					Path:          "/test/test.tf",
					Line:          6,
				},
				{
					ViolationType: errorMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          7,
				},
				{
					ViolationType: errorProviderNewline,
					Path:          "/test/test.tf",
					Line:          8,
				},
				{
					ViolationType: fmt.Sprintf(errorTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          9,
				},
				{
					ViolationType: fmt.Sprintf(errorProviderAttributes, attrProviderOrganization),
					Path:          "/test/test.tf",
					Line:          12,
				},
				{
					ViolationType: errorProviderNewline,
					Path:          "/test/test.tf",
					Line:          13,
				},
			},
			wantError: false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			results, err := findViolations([]byte(tc.content), tc.filename)
			if tc.wantError != (err != nil) {
				t.Errorf("expected error want: %#v, got: %#v - error: %v", tc.wantError, err != nil, err)
			}
			if diff := cmp.Diff(tc.expect, results); diff != "" {
				t.Errorf("results (-want,+got):\n%s", diff)
			}
		})
	}
}
