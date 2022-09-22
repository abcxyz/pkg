# abcxyz Common Packages

**"abcxyz Common Packages" is not an official Google product.**

```
                    ___
                  .' _ '.
                 / /` `\ \
                 | |   [__]
                 | |    {{
                 | |    }}
              _  | |  _ {{
  ___________<_>_| |_<_>}}________
      .=======^=(___)=^={{====.
     / .----------------}}---. \
    / /                 {{    \ \
   / /                  }}     \ \
  (  '========================='  )
   '-----------------------------'
```

abcxyz `pkg` provides a place for sharing common abcxyz packages across the
abcxyz repos.


## GitHub Actions

There are reusable workflows inside [./.github/workflows](.github/workflows),
which incapsulate common CI/CD logic to reduce repetition. For security, the
reusable workflows are pinned to specific references using
[ratchet](https://github.com/sethvargo/ratchet).


#### go-lint.yml

Use this workflow to perform basic Go linting checks:

```yaml
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
    with:
      go_version: '1.19'
```

Linting is done via [golangci-lint](https://golangci-lint.run/). If a
`.golangci.yml` file exists at the root of the repository, it uses those linter
settings. Otherwise, it uses a set of sane defaults.


#### go-test.yml

Use this workflow to perform basic Go tests:

```yaml
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
    with:
      go_version: '1.19'
```

Testing is done via the `go test` command with:

-   Test caching disabled
-   Test shuffling enabled
-   Race detector enabled
-   A 10 minute timeout


#### terraform-lint.yml

Use this workflow to perform basic Terraform linting checks:

```yaml
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
