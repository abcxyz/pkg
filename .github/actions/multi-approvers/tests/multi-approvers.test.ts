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
import nock from "nock";
import {
  MultiApproversAction,
  MultiApproversParams,
} from "../src/multi-approvers";

const GITHUB_API_BASE_URL = "https://api.github.com";

const BASE_PARAMS = {
  eventName: "pull_request",
  runId: 1,
  branch: "twig",
  pullNumber: 12,
  repoName: "speed-trap",
  repoOwner: "acme-org",
  token: "fake-token",
  team: "roadrunners",
  octokitOptions: { request: fetch },
  // eslint-disable-next-line no-unused-vars
  logDebug: (_: string) => {},
  // eslint-disable-next-line no-unused-vars
  logInfo: (_: string) => {},
  // eslint-disable-next-line no-unused-vars
  logNotice: (_: string) => {},
} as MultiApproversParams;

async function assertDoesNotReject(
  nockScope: any,
  paramOverrides: Partial<MultiApproversParams> = {},
) {
  const params = Object.assign({}, BASE_PARAMS, paramOverrides);
  const multiApproversAction = new MultiApproversAction(params);
  await assert.doesNotReject(() => multiApproversAction.validate());
  assert(
    nockScope.isDone(),
    `Unexecuted nock HTTP mocks: ${nockScope.pendingMocks()}`,
  );
}

async function assertRejects(
  nockScope: any,
  message: string,
  paramOverrides: Partial<MultiApproversParams> = {},
) {
  const params = Object.assign({}, BASE_PARAMS, paramOverrides);
  const multiApproversAction = new MultiApproversAction(params);
  await assert.rejects(() => multiApproversAction.validate(), {
    name: "Error",
    message,
  });
  assert(
    nockScope.isDone(),
    `Unexecuted nock HTTP mocks: ${nockScope.pendingMocks()}`,
  );
}

// Note that the { request: fetch } OctokitOptions are required for nock to work
// with octokit. This is because, by default, octokit uses a non-standard http
// library that nock does not recognize.
test("#multi-approvers", { concurrency: true }, async (suite) => {
  suite.beforeEach(async () => {
    nock.cleanAll();
  });

  await suite.test("should ignore PRs from internal users", async () => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "wile-e-coyote";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: prLogin,
        role: "member",
        state: "active",
      });

    await assertDoesNotReject(nockScope);
  });

  await suite.test(
    "should reject PRs from external users and no internal approvals",
    async () => {
      const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
      const prLogin = "wile-e-coyote";

      const nockScope = nock(GITHUB_API_BASE_URL)
        .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
        .reply(200, {
          owner: repoOwner,
          pull_number: pullNumber,
          repoName,
          user: {
            login: prLogin,
          },
        })
        .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
        .reply(404)
        .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
        .reply(200, []);

      await assertRejects(
        nockScope,
        "This pull request has 0 of 2 required internal approvals.",
      );
    },
  );

  await suite.test(
    "should succeed for PRs from external users and 2 internal approvals",
    async () => {
      const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
      const prLogin = "wile-e-coyote";
      const approver1 = "approver-1";
      const approver2 = "approver-2";

      const nockScope = nock(GITHUB_API_BASE_URL)
        .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
        .reply(200, {
          owner: repoOwner,
          pull_number: pullNumber,
          repoName: repoName,
          user: {
            login: prLogin,
          },
        })
        .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
        .reply(404)
        .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
        .reply(200, [
          {
            submitted_at: 1714636800,
            user: {
              login: approver1,
            },
            state: "APPROVED",
          },
          {
            submitted_at: 1714636801,
            user: {
              login: approver2,
            },
            state: "APPROVED",
          },
        ])
        .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
        .reply(200, {
          org: repoOwner,
          team_slug: team,
          username: approver1,
          role: "member",
          state: "active",
        })
        .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
        .reply(200, {
          org: repoOwner,
          team_slug: team,
          username: approver2,
          role: "member",
          state: "active",
        });

      await assertDoesNotReject(nockScope);
    },
  );

  await suite.test("should ignore PR review comments", async () => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "wile-e-coyote";
    const approver1 = "approver-1";
    const approver2 = "approver-2";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          submitted_at: 1714636800,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636801,
          user: {
            login: approver2,
          },
          state: "COMMENTED",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "active",
      });

    await assertRejects(
      nockScope,
      "This pull request has 1 of 2 required internal approvals.",
    );
  });

  await suite.test("should handle rescinded approval", async () => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "pr-owner";
    const approver1 = "approver-1";
    const approver2 = "approver-2";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          submitted_at: 1714636800,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636801,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636802,
          user: {
            login: approver2,
          },
          state: "request_changes",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "active",
      });

    await assertRejects(
      nockScope,
      "This pull request has 1 of 2 required internal approvals.",
    );
  });

  await suite.test("should fail with pending member approval", async () => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "pr-owner";
    const approver1 = "approver-1";
    const approver2 = "approver-2";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          submitted_at: 1714636800,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636801,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "pending",
      });

    await assertRejects(
      nockScope,
      "This pull request has 1 of 2 required internal approvals.",
    );
  });

  await suite.test("should re-run failed runs on PR reviews", async () => {
    const { repoOwner, repoName, pullNumber, team, branch, runId } =
      BASE_PARAMS;
    const eventName = "pull_request_review";
    const prLogin = "pr-owner";
    const approver1 = "approver-1";
    const approver2 = "approver-2";
    const workflowId = 37;
    const failedRunId = 827;

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          run_number: 21,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          run_number: 22,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "active",
      })
      .get(`/repos/${repoOwner}/${repoName}/actions/runs/${runId}`)
      .reply(200, {
        workflow_id: workflowId,
      })
      .get(
        `/repos/${repoOwner}/${repoName}/actions/workflows/${workflowId}/runs`,
      )
      .query({
        branch,
        event: "pull_request",
        per_page: 100,
      })
      .reply(200, [
        {
          id: failedRunId,
          pull_requests: [
            {
              number: pullNumber,
            },
          ],
        },
      ])
      .post(`/repos/${repoOwner}/${repoName}/actions/runs/${failedRunId}/rerun`)
      .reply(200, {});

    await assertDoesNotReject(nockScope, { eventName });
  });

  await suite.test("handles review with unset user", async (t) => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "pr-owner";
    const approver1 = "approver-1";
    const approver2 = "approver-2";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          submitted_at: 1714636804,
          state: "APPROVED",
        },
        {
          submitted_at: 1714636800,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636801,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "pending",
      });

    const overrideParams = {
      // eslint-disable-next-line no-unused-vars
      logNotice: (_: string) => {},
    };
    const mockLogNotice = t.mock.method(overrideParams, "logNotice");

    await assertRejects(
      nockScope,
      "This pull request has 1 of 2 required internal approvals.",
      overrideParams,
    );

    assert.equal(mockLogNotice.mock.callCount(), 1);
    const msg = mockLogNotice.mock.calls[0].arguments[0];
    assert(
      msg.startsWith("Ignoring pull request review because user is unset: "),
    );
  });

  await suite.test("caches membership test results", async () => {
    const { repoOwner, repoName, pullNumber, team } = BASE_PARAMS;
    const prLogin = "wile-e-coyote";
    const approver1 = "approver-1";
    const approver2 = "approver-2";

    const nockScope = nock(GITHUB_API_BASE_URL)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}`)
      .reply(200, {
        owner: repoOwner,
        pull_number: pullNumber,
        repoName,
        user: {
          login: prLogin,
        },
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${prLogin}`)
      .reply(404)
      .get(`/repos/${repoOwner}/${repoName}/pulls/${pullNumber}/reviews`)
      .reply(200, [
        {
          submitted_at: 1714636800,
          user: {
            login: approver1,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636801,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
        {
          submitted_at: 1714636802,
          user: {
            login: approver2,
          },
          state: "APPROVED",
        },
      ])
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver1}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver1,
        role: "member",
        state: "active",
      })
      .get(`/orgs/${repoOwner}/teams/${team}/memberships/${approver2}`)
      .reply(200, {
        org: repoOwner,
        team_slug: team,
        username: approver2,
        role: "member",
        state: "active",
      });

    await assertDoesNotReject(nockScope);
  });
});
