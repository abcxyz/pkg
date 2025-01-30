# abcxyz Common Packages

**"abcxyz Common Packages" is not an official Google product.**

![Kitchen Sink](./docs/sink.svg)

abcxyz `pkg` provides a place for sharing common abcxyz packages across the
abcxyz repos.


## GitHub Actions

There are reusable workflows inside [./.github/workflows](.github/workflows),
which encapsulate common CI/CD logic to reduce repetition. For security, the
reusable workflows are pinned to specific references using
[ratchet](https://github.com/sethvargo/ratchet).

The reusable workflows use a default runner image of `ubuntu-latest`, since this
works for most use cases. We do not recommend customizing the runner image
unless you need additional performance. To customize the runner image, set the
`runs-on` key as an input to the reusable workflow to a **valid JSON string**
representing the [GitHub `runs-on`
configuration](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idruns-on).
Because GitHub Actions reusable workflows do not support complex input types,
these values must be a valid JSON-encoded string.

```yaml
# Example of using a different runner
uses: 'abcxyz/pkg/.github/workflows/workflow.yml@main'
with:
  runs-on: '"macos-latest"' # double quoting is required

# Example of using a self-hosted runner
uses: 'abcxyz/pkg/.github/workflows/workflow.yml@main'
with:
  runs-on: '["self-hosted", "ubuntu-22.04"]'

# Example of using a label
uses: 'abcxyz/pkg/.github/workflows/workflow.yml@main'
with:
  runs-on: '{"label": "4-core"}'
```


#### go-lint.yml

Use this workflow to perform basic Go linting checks:

```yaml
name: 'go_lint'
on:
  push:
    branches:
      - 'main'
    paths:
      - '**.go'
  pull_request:
    branches:
      - 'main'
    paths:
      - '**.go'
  workflow_dispatch:

jobs:
  lint:
    uses: 'abcxyz/pkg/.github/workflows/go-lint.yml@main'
```

Linting is done via [golangci-lint](https://golangci-lint.run/). If a
`.golangci.yml` file exists at the root of the repository, it uses those linter
settings. Otherwise, it uses a set of sane defaults.


#### go-test.yml

Use this workflow to perform basic Go tests:

```yaml
name: 'go_test'
on:
  push:
    branches:
      - 'main'
    paths:
      - '**.go'
  pull_request:
    branches:
      - 'main'
    paths:
      - '**.go'
  workflow_dispatch:

jobs:
  lint:
    uses: 'abcxyz/pkg/.github/workflows/go-test.yml@main'
```

Testing is done via the `go test` command with:

-   Test caching disabled
-   Test shuffling enabled
-   Race detector enabled
-   A 10 minute timeout


#### terraform-lint.yml

Use this workflow to perform basic Terraform linting checks:

```yaml
name: 'terraform_lint'
on:
  push:
    branches:
      - 'main'
    paths:
      - '**.tf'
  pull_request:
    branches:
      - 'main'
    paths:
      - '**.tf'
  workflow_dispatch:

jobs:
  lint:
    uses: 'abcxyz/pkg/.github/workflows/terraform-lint.yml@main'
    with:
      terraform_version: '1.2'
      directory: './terraform'
```

If you have multiple Terraform configurations, repeat the stanza for each
directory. Linting is done in two steps:

1.  Run `terraform validate`. This will fail if the Terraform is invalid.

1.  Run `terraform fmt` and check the git diff. This will will if the Terraform
    file is not formatted. On failure, the output will include the diff.

#### want-lgtm-all.yml

Use this workflow to require an approval from all requested reviewers:

```yaml
name: 'want_lgtm_all'
on:
  pull_request:
    types:
      - 'opened'
      - 'edited'
      - 'reopened'
      - 'synchronize'
      - 'ready_for_review'
      - 'review_requested'
      - 'review_request_removed'
  pull_request_review:
    types:
      - 'submitted'
      - 'dismissed'

concurrency:
  group: '${{ github.workflow }}-${{ github.event_name }}-${{ github.head_ref || github.ref }}'
  cancel-in-progress: true

permissions:
  actions: 'write'
  pull-requests: 'read'

jobs:
  want-lgtm-all:
    uses: 'abcxyz/pkg/.github/workflows/want-lgtm-all.yml@main'
```

When creating a pull request, include the text `want_lgtm=all` in the body to require an
approval from all requested reviewers.

An admin will need to create a new ruleset within the repo to add want_lgtm_all to be included
as a required status check.

#### multi-approvers.yml

Use this workflow to require two in-org approvers for pull requests sent from an
out-of-org user. This prevents in-org users from creating "sock puppet" accounts
and approving their own pull requests with their in-org account.

This workflow requires one input: `org-members-path`. This is a JSON formatted
file with the following schema:

```json
[
  {"login": "github-user-name-1"},
  {"login": "github-user-name-2"}
]
```

This file can be generated using the following command:

```bash
gh api \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  --paginate \
  /orgs/{org}/members | \
jq '[.[] | {login}]'
```

```yaml
name: 'multi-approvers'

on:
  pull_request:
    types:
      - 'opened'
      - 'edited'
      - 'reopened'
      - 'synchronize'
      - 'ready_for_review'
      - 'review_requested'
      - 'review_request_removed'
  pull_request_review:
    types:
      - 'submitted'
      - 'dismissed'

permissions:
  actions: 'write'
  contents: 'read'
  pull-requests: 'read'

concurrency:
  group: '${{ github.workflow }}-${{ github.head_ref || github.ref }}'
  cancel-in-progress: true

jobs:
  multi-approvers:
    uses: 'abcxyz/pkg/.github/workflows/multi-approvers.yml@main'
    with:
      org-members-path: 'abcxyz/pkg/main/.github/workflows/members.json'
```

Note: the `org-members-path` should be the full path to the JSON file without
the leading `/` and should be accessible by using the URL:
https://raw.githubusercontent.com/${org-members-path}.

### maybe-build-docker.yml

Use this workflow to build and push docker images to several supported docker
registries. Docker images will only be rebuilt if the hash of the dockerfile
(or other configurable input files) changes.

```yaml
name: 'ci_docker_test'
on:
  pull_request:

permissions:
  contents: 'read'  # For checking out repository code.
  packages: 'write' # For pushing/pulling from github docker registry.

jobs:
  maybe-create-ci-image:
    uses: abcxyz/pkg/.github/workflows/maybe-build-docker.yml
    with:
      dockerfile: '.github/workflows/ci.dockerfile'
      github-image-name: 'ghcr.io/${{ github.repo }}/ci-docker-test'

  # Then we run using the latest docker image and build all artifacts in that
  # container.
  ci-on-docker-image:
    needs:
      - 'maybe-create-ci-image'
    runs-on: 'ubuntu-latest'
    container:
      image: 'ghcr.io/${{ github.repo }}/ci-docker-test:${{ needs.maybe-create-ci-image.outputs.docker-tag }}'
      credentials:
        username: '${{ github.actor }}'
        password: '${{ secrets.GITHUB_TOKEN }}'
    steps:
      - name: 'Checkout'
        uses: 'actions/checkout@0ad4b8fadaa221de15dcec353f45205ec38ea70b' # ratchet:actions/checkout@v4

      - name: 'Run something...'
        shell: 'bash'
        run: |
          # Do something in the docker container context...
```

For a full list of all input arguments and configuration see the
[workflow file](.github/workflows/maybe-build-docker.yml).

This is particularly useful:

1. If you have a complex build environment with many non-standard system dependencies.
2. If you would like to share your CI env easily with engineers without
   forcing them to re-build the image locally every time there is a change.
3. You are compiling for other architectures and your build tooling does not
   natively support it.
4. You would prefer to manage CI dependencies in docker rather than github actions.
