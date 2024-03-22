# Release Guide For Team Members

## Process

1. Merge any changes you want to include into `main`.
2. Run
   the [`draft-release`](https://github.com/abcxyz/pkg/actions/workflows/draft-release.yml)
   workflow. Run from the main branch. Choose what sort
   of [semantic versoning](https://semver.org)
   increment you want to make (Major, Minor, Patch, Prerelease).
3. Approve the `PR` created by `draft-release`. Once `PR` is merged
   the `release`
   workflow will automatically run.
4. Optional: Edit description at https://github.com/abcxyz/pkg/releases

### Patching Old Versions

Currently, you can only increment an existing version on the main branch.
There is currently no automated way to make a patch of an old release, and
doing so must be done by manually tagging and releasing.

### `VERSION` File

`VERSION` in the root of the repo is used by `draft-release` to determine the
"current" version to increment from. It must be a valid existing tag.

#### Failed Release
If a release fails, you will need to manually decrement the `VERSION` file
to its previous state before running `draft-release` again to create a new PR.
