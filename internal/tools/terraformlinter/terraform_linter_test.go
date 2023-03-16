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

	cases := []struct {
		name      string
		content   string
		filename  string
		expect    []*ViolationInstance
		wantError bool
	}{
		{
			name: "no_special_attributes",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "for_each_correct",
			content: `
				resource "google_project_service" "run_api" {  
					for_each = toset(["name"])

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "for_each_missing_newline",
			content: `
				resource "google_project_service" "run_api" {  
					for_each = toset(["name"])
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "for_each_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					for_each = toset(["name"])
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrForEach),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name: "count_correct",
			content: `
				resource "google_project_service" "run_api" {  
					count = 3

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "count_missing_newline",
			content: `
				resource "google_project_service" "run_api" {  
					count = 3
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "count_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					count = 3
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrCount),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name: "provider_correct",
			content: `
				resource "google_project_service" "run_api" {  
					provider = "some_provider"

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "provider_missing_newline",
			content: `
				resource "google_project_service" "run_api" {  
					provider = "some_provider"
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},

		{
			name: "provider_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					provider = "some_provider"
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrProvider),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name: "project_correct_no_meta_block",
			content: `
				resource "google_project_service" "run_api" {  
					project = "some_project_id"

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "project_correct_meta_block",
			content: `
				resource "google_project_service" "run_api" {  
					for_each = toset(["name"]) 

					project = "some_project_id"

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "project_missing_newline",
			content: `
				resource "google_project_service" "run_api" {  
					project = "some_project_id"
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: violationProviderNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "project_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					project = "some_project_id"
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationProviderAttributes, attrProviderProject),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationProviderNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name: "depends_on_correct",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					disable_on_destroy = true
					depends_on = [
						"something"
					]
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "depends_on_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					depends_on = [
						"something"
					]
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrDependsOn),
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "lifecycle_correct",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					disable_on_destroy = true
					lifecycle = {
						prevent_destroy = true
					}
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "lifecycle_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					lifecycle = {
						prevent_destroy = true
					}
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "trailing_mix_correct",
			content: `
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
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "trailing_mix_out_of_order",
			content: `
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
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          5,
				},
				{
					ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrDependsOn),
					Path:          "/test/test.tf",
					Line:          8,
				},
			},
			wantError: false,
		},
		{
			name: "source_correct",
			content: `
				resource "google_project_service" "run_api" {  
					source = "http://somerepo"

					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "source_missing_newline",
			content: `
				resource "google_project_service" "run_api" {  
					source = "http://somerepo"
					service = "run.googleapis.com"  
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          4,
				},
			},
			wantError: false,
		},
		{
			name: "source_out_of_order",
			content: `
				resource "google_project_service" "run_api" {  
					service = "run.googleapis.com"  
					source = "http://somerepo"
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrSource),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		{
			name: "repro_panic_on_comment_at_end_of_line",
			content: `
				resource "a" "b" {
					c = d # e
				}
			`,
			wantError: false,
		},
		{
			name: "all_correct",
			content: `
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
				`,
			filename:  "/test/test.tf",
			expect:    nil,
			wantError: false,
		},
		{
			name: "mixed_out_of_order",
			content: `
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
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrProvider),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          5,
				},
				{
					ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrForEach),
					Path:          "/test/test.tf",
					Line:          6,
				},
				{
					ViolationType: violationMetaBlockNewline,
					Path:          "/test/test.tf",
					Line:          7,
				},
				{
					ViolationType: violationProviderNewline,
					Path:          "/test/test.tf",
					Line:          8,
				},
				{
					ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrLifecycle),
					Path:          "/test/test.tf",
					Line:          9,
				},
				{
					ViolationType: fmt.Sprintf(violationProviderAttributes, attrProviderOrganization),
					Path:          "/test/test.tf",
					Line:          12,
				},
				{
					ViolationType: violationProviderNewline,
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
