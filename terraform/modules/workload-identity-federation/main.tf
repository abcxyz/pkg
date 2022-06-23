/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
resource "google_project_service" "serviceusage" {
  project            = var.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "iamcredentials.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

resource "google_iam_workload_identity_pool" "pool" {
  provider                  = google-beta
  project                   = var.project_id
  workload_identity_pool_id = "github-pool"
  description               = "GitHub pool"
}

resource "google_iam_workload_identity_pool_provider" "pool_provider" {
  provider                           = google-beta
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "GitHub provider"
  attribute_mapping                  = var.pool_provider_attribute_mapping
  attribute_condition                = <<EOT
  assertion.repository_owner == 'abcxyz' &&
  assertion.repository_owner_id == '93787867'&&
  assertion.repository == '${var.github_slug}' &&
  !(assertion.event_name == 'pull_request_target')
  EOT
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}
