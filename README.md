# abcxyz Common Packages

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

Use this workflow to perform basic linting checks:

```yaml
jobs:
  lint:
    uses: 'abcxyz/pkg/.github/workflows/go-lint.yml@main'
    with:
      go_version: '1.18'
```

Linting is done via [golangci-lint](https://golangci-lint.run/). If a
`.golangci.yml` file exists at the root of the repository, it uses those linter
settings. Otherwise, it uses a set of sane defaults.


#### go-test.yml

Use this workflow to perform basic go tests:

```yaml
jobs:
  lint:
    uses: 'abcxyz/pkg/.github/workflows/go-test.yml@main'
    with:
      go_version: '1.18'
```

Testing is done via the `go test` command with:

- Test caching disabled
- Test shuffling enabled
- Race detector enabled
- A 10 minute timeout
