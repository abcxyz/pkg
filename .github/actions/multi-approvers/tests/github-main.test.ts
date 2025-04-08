// Copyright 2025 Google LLC
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

import assert from "node:assert/strict";
import { test } from "node:test";
import * as ghCore from "@actions/core";
import { context as ghContext } from "@actions/github";
import { DeepPartial } from "./deep-partial";
import { main } from "../src/github-main";

type Core = typeof ghCore;
type Context = typeof ghContext;
type PartialContext = DeepPartial<Context>;

interface TestCase {
  description: string;
  inputs: {
    [key: string]: string;
  };
  contextOverrides: PartialContext;
  expected: {
    failMessage: string;
  };
}

function newFakeCore(inputs: { [key: string]: string }): Core {
  return {
    debug: () => {},
    getInput: (name: string) => inputs[name],
    info: () => {},
    setFailed: () => {},
  } as unknown as Core;
}

const BASE_CONTEXT = {
  eventName: "pull_request",
  runId: 1,
  payload: {
    pull_request: {
      number: 1,
      head: {
        ref: "fake-branch",
      },
    },
    repository: {
      name: "fake-repository",
      owner: {
        login: "test-org",
      },
    },
  },
} as PartialContext;

const TEST_CASES = [
  {
    description: "should fail on unsupported event",
    inputs: {
      team: "fake-team",
      token: "fake-token",
    },
    contextOverrides: {
      eventName: "push",
    },
    expected: {
      failMessage: "Multi-approvers action failed: Unexpected event [push].",
    },
  },
  {
    description: "fails when no inputs are set",
    inputs: {},
    expected: {
      failMessage:
        "Multi-approvers action failed: Invalid input(s): token is required; team is required",
    },
  },
  {
    description: "fails when token input is not set",
    inputs: {
      team: "fake-team",
    },
    expected: {
      failMessage:
        "Multi-approvers action failed: Invalid input(s): token is required",
    },
  },
  {
    description: "fails when team input is not set",
    inputs: {
      token: "fake-token",
    },
    expected: {
      failMessage:
        "Multi-approvers action failed: Invalid input(s): team is required",
    },
  },
] as Array<TestCase>;

test("#github-main", { concurrency: true }, async (suite) => {
  for (const c of TEST_CASES) {
    await suite.test(c.description, async (t) => {
      const core = newFakeCore(c.inputs);
      const setFailed = t.mock.method(core, "setFailed", () => {});
      const context = Object.assign(
        {},
        BASE_CONTEXT,
        c.contextOverrides,
      ) as unknown as Context;

      await main(core, context);

      assert.equal(setFailed.mock.calls.length, 1);
      const failMsg = setFailed.mock.calls[0].arguments[0];
      assert.equal(failMsg, c.expected.failMessage);
    });
  }
});
