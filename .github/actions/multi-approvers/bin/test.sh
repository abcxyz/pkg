#!/usr/bin/env bash
set -eEuo pipefail

#
# As of Node 20, the --test parameter does not support globbing, and it does not
# support variable Windows paths. We also cannot invoke the test runner
# directly, because while it has an API, there's no way to force it to transpile
# the Typescript into JavaScript before passing it to the runner.
#
# So we're left with this solution, which finds all non-./node_modules files
# that end in *.test.ts, and then execs out to that node. We have to exec so the
# stderr/stdout and exit code is appropriately fed to the caller.
#

files_input="$(find ./* -type f -not -path './node_modules/*' -name '*.test.ts' | sort)"
readarray -t files <<<"${files_input}"

set -x
exec node \
  --require ts-node/register \
  --test-reporter spec \
  --test "${files[@]}"
