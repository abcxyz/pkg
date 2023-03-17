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
			name: "all_correct",
			content: `
				resource "google_project_service" "run_api" {
					for_each = toset(["name"])
					provider = "someprovider"

					organization = "abcxyz"
					folder = "fid"
					project = "pid"
					project_id = "pid"

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
			},
			wantError: false,
		},
		// Terraform AST treats comments on a line differently than any other token.
		// Comments absorb the newline character instead of treating it as a separate token.
		// This requires us to check for either a true newline token or a comment token
		// that we can treat as the end of the line. See issue #83.
		{
			name: "repro_panic_on_comment_at_end_of_line",
			content: `
				resource "a" "b" {
					c = var.d # e
				}
			`,
			wantError: false,
		},
		{
			name: "resource with hyphen in name",
			content: `
				resource "google_project_service" "run-api" {
					service = "run.googleapis.com"
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationHyphenInResouceName, "run-api"),
					Path:          "/test/test.tf",
					Line:          2,
				},
			},
			wantError: false,
		},
		{
			name: "module with hyphen in name",
			content: `
				module "my-cool-module" {
					x = "some value"
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationHyphenInResouceName, "my-cool-module"),
					Path:          "/test/test.tf",
					Line:          2,
				},
			},
			wantError: false,
		},
		{
			name: "variable with hyphen in name",
			content: `
				variable "billing-account" {
					description = "The ID of the billing account to associate projects with"
					type        = string
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationHyphenInResouceName, "billing-account"),
					Path:          "/test/test.tf",
					Line:          2,
				},
			},
			wantError: false,
		},
		{
			name: "output with hyphen in name",
			content: `
				output "my-output" {
					value       = module.my-output
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationHyphenInResouceName, "my-output"),
					Path:          "/test/test.tf",
					Line:          2,
				},
			},
			wantError: false,
		},
		{
			name: "provider_project_at_top",
			content: `
				resource "google_project_service" "run_api" {
					project = "pid"
					folder = "fid"
					organization = "abcxyz"
				}
				`,
			filename: "/test/test.tf",
			expect: []*ViolationInstance{
				{
					ViolationType: fmt.Sprintf(violationProviderAttributes, attrProviderFolder),
					Path:          "/test/test.tf",
					Line:          4,
				},
				{
					ViolationType: fmt.Sprintf(violationProviderAttributes, attrProviderOrganization),
					Path:          "/test/test.tf",
					Line:          5,
				},
			},
			wantError: false,
		},
		// Issue #87 - source and for_each are both valid at the top and shouldn't
		// cause violations if both are present.
		{
			name: "for_each_and_source_both_present_repro",
			content: `
				module "some_module" {
					source = "git://https://github.com/abc/def"
					for_each = local.mylocal
				}

				module "some_module" {
					for_each = local.mylocal
					source = "git://https://github.com/abc/def"
				}
			`,
			wantError: false,
		},
		{
			// linter is detecting the "module" ident token in the trimprefix call and starting a new
			// block which throws all of the block selection logic into a broken state. This causes it
			// to see the "for_each" in the following resource as being on the wrong line (not at the top)
			// causing a false violation
			name: "special_ident_tokens_in_locals",
			content: `
			  locals {
				ingestion_backed_client_env_vars = {
				  "AUDIT_CLIENT_BACKEND_REMOTE_ADDRESS" : "${trimprefix(module.server_service.audit_log_server_url, "https://")}:443",
				  "AUDIT_CLIENT_CONDITION_REGEX_PRINCIPAL_INCLUDE" : ".*",
				}
			  }
			  
			  resource "google_cloud_run_service" "ingestion_backend_client_services" {
				for_each = var.client_images
			  }
			`,
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
