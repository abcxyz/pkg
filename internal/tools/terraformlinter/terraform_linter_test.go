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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTerraformLinter_FindViolations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		content  string
		filename string
		expect   []*ViolationInstance
	}{
		{
			name: "no_special_attributes",
			content: `
				resource "google_project_service" "run_api" {
					service = "run.googleapis.com"
					disable_on_destroy = true
				}
				`,
			filename: "/test/test.tf",
			expect:   nil,
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
					Message: `The attribute "for_each" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
					Message: `The attribute "count" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
					Message: `The attribute "provider" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The provider specific attributes must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
					Message: `The attribute "project" must be below any meta attributes (e.g. "for_each", "count") but above all other attributes. Attributes must be ordered organization > folder > project.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The provider specific attributes must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The attribute "depends_on" must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The attribute "lifecycle" must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The attribute "lifecycle" must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`,
					Path:    "/test/test.tf",
					Line:    5,
				},
				{
					Message: `The attribute "depends_on" must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`,
					Path:    "/test/test.tf",
					Line:    8,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
			},
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
					Message: `The attribute "source" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
			filename: "/test/test.tf",
			expect:   nil,
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
					Message: `The attribute "provider" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
				{
					Message: `The attribute "for_each" must be in the meta block at the top of the definition.`,
					Path:    "/test/test.tf",
					Line:    6,
				},
				{
					Message: `The meta block must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    7,
				},
				{
					Message: `The provider specific attributes must have an additional newline separating it from the next section.`,
					Path:    "/test/test.tf",
					Line:    8,
				},
				{
					Message: `The attribute "lifecycle" must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`,
					Path:    "/test/test.tf",
					Line:    9,
				},
				{
					Message: `The attribute "organization" must be below any meta attributes (e.g. "for_each", "count") but above all other attributes. Attributes must be ordered organization > folder > project.`,
					Path:    "/test/test.tf",
					Line:    12,
				},
			},
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
					Message: `The resource "run-api" must not contain a "-" in its name.`,
					Path:    "/test/test.tf",
					Line:    2,
				},
			},
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
					Message: `The resource "my-cool-module" must not contain a "-" in its name.`,
					Path:    "/test/test.tf",
					Line:    2,
				},
			},
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
					Message: `The resource "billing-account" must not contain a "-" in its name.`,
					Path:    "/test/test.tf",
					Line:    2,
				},
			},
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
					Message: `The resource "my-output" must not contain a "-" in its name.`,
					Path:    "/test/test.tf",
					Line:    2,
				},
			},
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
					Message: `The attribute "folder" must be below any meta attributes (e.g. "for_each", "count") but above all other attributes. Attributes must be ordered organization > folder > project.`,
					Path:    "/test/test.tf",
					Line:    4,
				},
				{
					Message: `The attribute "organization" must be below any meta attributes (e.g. "for_each", "count") but above all other attributes. Attributes must be ordered organization > folder > project.`,
					Path:    "/test/test.tf",
					Line:    5,
				},
			},
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
		},
		{
			name: "allows_import_blocks",
			content: `
			import {
			  to = module.project.google_project.default
			  id = "project-id-with-hyphens"
			}
			`,
		},
		{
			name: "allows_moved_blocks",
			content: `
				moved {
					from = google_bigquery_table_iam_member.editors["serviceAccount:service-123456789@dataflow-service-producer-prod.iam.gserviceaccount.com"]
					to   = module.project.google_bigquery_table_iam_member.editors["serviceAccount:service-123456789@dataflow-service-producer-prod.iam.gserviceaccount.com"]
				}
			`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			results, err := findViolations([]byte(tc.content), tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expect, results); diff != "" {
				t.Errorf("results (-want,+got):\n%s", diff)
			}
		})
	}
}
