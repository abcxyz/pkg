# Multi-approvers GitHub Action

This GitHub Action requires at least 2 internal approvers for pull requests with
external authors.

Internal approvers are GitHub users who are members of the given team. All other
users are external.

## Development

`npm install`: Downloads required node packages.

`npm run build-min`: Generates minimized versions of Javascript source code.
This MUST be run after making changes to code under the `src` directory.

`npm run tests`: runs tests.
