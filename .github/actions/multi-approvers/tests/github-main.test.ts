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
import { main } from "../src/github-main";

type Core = typeof ghCore;
type Context = typeof ghContext;

function newFakeCore(inputs: { [key: string]: string }): Core {
  return {
    debug: () => {},
    getInput: (name: string) => inputs[name],
    info: () => {},
    setFailed: () => {},
  } as unknown as Core;
}

test("#github-main", { concurrency: true }, async (suite) => {
  await suite.test("should fail on unsupported event", async (t) => {
    const core = newFakeCore({ token: "fake-token", team: "fake-team" });
    const setFailed = t.mock.method(core, "setFailed", () => {});
    const context = {
      eventName: "push",
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
    } as unknown as Context;

    await main(core, context);

    assert.equal(setFailed.mock.calls.length, 1);
    const failMsg = setFailed.mock.calls[0].arguments[0];
    assert.equal(
      failMsg,
      "Multi-approvers action failed: unexpected event [push].",
    );
  });
});
