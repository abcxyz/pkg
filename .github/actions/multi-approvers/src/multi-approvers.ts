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

/*eslint no-unused-vars: ["error", { "argsIgnorePattern": "^_$" }]*/

import { getOctokit } from "@actions/github";
import { OctokitOptions } from "@octokit/core";
import { RestEndpointMethodTypes } from "@octokit/rest";
import { RequestError } from "@octokit/request-error";

type PullRequestReview =
  RestEndpointMethodTypes["pulls"]["listReviews"]["response"]["data"];
type Octokit = ReturnType<typeof getOctokit>;
export type EventName = "pull_request" | "pull_request_review";

export function isEventName(v: string): v is EventName {
  return ["pull_request", "pull_request_review"].includes(v);
}

const MIN_APPROVED_COUNT = 2;
const APPROVED = "approved";
const COMMENTED = "commented";
const ALLOWED_TEAM_MEMBER_ROLES = ["maintainer", "member"];
const ACTIVE = "active";

export interface MultiApproversParams {
  eventName: EventName;
  runId: number;
  branch: string;
  pullNumber: number;
  repoName: string;
  repoOwner: string;
  token: string;
  team: string;
  octokitOptions?: OctokitOptions;
  logDebug: (_: string) => void;
  logInfo: (_: string) => void;
  logNotice: (_: string) => void;
}

export class MultiApproversAction {
  private readonly eventName: string;
  private readonly runId: number;
  private readonly branch: string;
  private readonly pullNumber: number;
  private readonly repoName: string;
  private readonly repoOwner: string;
  private readonly team: string;
  private readonly octokit: Octokit;

  constructor(params: MultiApproversParams) {
    this.eventName = params.eventName;
    this.runId = params.runId;
    this.branch = params.branch;
    this.pullNumber = params.pullNumber;
    this.repoName = params.repoName;
    this.repoOwner = params.repoOwner;
    this.team = params.team;
    this.logDebug = params.logDebug;
    this.logInfo = params.logInfo;
    this.logNotice = params.logNotice;

    this.octokit = getOctokit(params.token, params.octokitOptions);
  }

  // Set in the constructor.
  private logDebug: (_: string) => void;

  // Set in the constructor.
  private logInfo: (_: string) => void;

  // Set in the constructor.
  private logNotice: (_: string) => void;

  // Tests whether the given login is an active member of the team.
  private async isInternal(login: string): Promise<boolean> {
    try {
      const response = await this.octokit.rest.teams.getMembershipForUserInOrg({
        org: this.repoOwner,
        team_slug: this.team,
        username: login,
      });
      return (
        ALLOWED_TEAM_MEMBER_ROLES.includes(response.data.role) &&
        response.data.state === ACTIVE
      );
    } catch (err) {
      if (err instanceof RequestError && err.status === 404) {
        this.logDebug(
          `Received 404 testing membership; assuming user is not a member: ${JSON.stringify(
            err,
          )}`,
        );
        // We can get here for a few known reasons:
        // 1) The user is not a member
        // 2) The team does not exist
        // 3) Invalid token
        // In all these cases, it's safe to return false.
        return false;
      }
      throw err;
    }
  }

  // Returns the number of approvals from members in the given list.
  private async internalApprovedCount(
    submittedReviews: PullRequestReview,
    prLogin: string,
  ): Promise<number> {
    // Sort by chronological order.
    const sortedReviews = submittedReviews.sort(
      (a, b) =>
        new Date(a.submitted_at || 0).getTime() -
        new Date(b.submitted_at || 0).getTime(),
    );
    const reviewStateByLogin = new Map<string, string>();

    for (const r of sortedReviews) {
      if (!r.user) {
        this.logNotice(
          `Ignoring pull request review because user is unset: ${JSON.stringify(r)}`,
        );
        continue;
      }

      const reviewerLogin = r.user.login;

      // Ignore the PR user.
      if (reviewerLogin === prLogin) {
        continue;
      }

      // Only consider internal users.
      const isInternalUser = await this.isInternal(reviewerLogin);
      if (!isInternalUser) {
        continue;
      }

      // Set state if it does not exist.
      if (!reviewStateByLogin.has(reviewerLogin)) {
        reviewStateByLogin.set(reviewerLogin, r.state);
        continue;
      }

      // Always update state if not approved.
      if (reviewStateByLogin.get(reviewerLogin) !== APPROVED) {
        reviewStateByLogin.set(reviewerLogin, r.state);
        continue;
      }

      // Do not update approved state for a comment.
      if (
        reviewStateByLogin.get(reviewerLogin) === APPROVED &&
        r.state !== COMMENTED
      ) {
        reviewStateByLogin.set(reviewerLogin, r.state);
        continue;
      }
    }

    return Array.from(reviewStateByLogin.values()).filter((s) => s === APPROVED)
      .length;
  }

  /** Checks that approval requirements are satisfied. */
  private async validateApprovers() {
    const prResponse = await this.octokit.rest.pulls.get({
      owner: this.repoOwner,
      repo: this.repoName,
      pull_number: this.pullNumber,
    });
    const prLogin = prResponse.data.user.login;

    const isInternalPr = await this.isInternal(prLogin);
    if (isInternalPr) {
      // Do nothing if the pull request owner is an internal user.
      this.logInfo(
        `Pull request login ${
          prLogin
        } is an internal member, therefore no special approval rules apply.`,
      );
      return;
    }
    const submittedReviews: PullRequestReview = await this.octokit.paginate(
      this.octokit.rest.pulls.listReviews,
      {
        owner: this.repoOwner,
        repo: this.repoName,
        pull_number: this.pullNumber,
      },
    );

    const approvedCount = await this.internalApprovedCount(
      submittedReviews,
      prLogin,
    );

    this.logInfo(`Found ${approvedCount} ${APPROVED} internal reviews.`);

    if (approvedCount < MIN_APPROVED_COUNT) {
      throw new Error(
        `This pull request has ${approvedCount} of ${
          MIN_APPROVED_COUNT
        } required internal approvals.`,
      );
    }
  }

  /**
   * Re-runs the approval checks on pull request review.
   *
   * This is required because GitHub treats checks made by pull_request and
   * pull_request_review as different status checks.
   */
  private async revalidateApprovers(workflowId: number) {
    // Get all failed runs.
    const runs = await this.octokit.paginate(
      this.octokit.rest.actions.listWorkflowRuns,
      {
        owner: this.repoOwner,
        repo: this.repoName,
        workflow_id: workflowId,
        branch: this.branch,
        event: "pull_request",
        status: "failure",
        per_page: 100,
      },
    );

    const failedRuns = runs
      .filter((r) =>
        (r.pull_requests || [])
          .map((pr) => pr.number)
          .includes(this.pullNumber),
      )
      .sort((v) => v.id);

    // If there are failed runs for this PR, re-run the workflow.
    if (failedRuns.length > 0) {
      await this.octokit.rest.actions.reRunWorkflow({
        owner: this.repoOwner,
        repo: this.repoName,
        run_id: failedRuns[0].id,
      });
    }
  }

  private async getWorkflowId(): Promise<number> {
    const response = await this.octokit.rest.actions.getWorkflowRun({
      owner: this.repoOwner,
      repo: this.repoName,
      run_id: this.runId,
    });
    return response.data.workflow_id;
  }

  async validate() {
    await this.validateApprovers();

    // If this action was triggered by a review, we want to re-run for previous
    // failed runs.
    if (this.eventName === "pull_request_review") {
      const workflowId = await this.getWorkflowId();
      await this.revalidateApprovers(workflowId);
    }
  }
}
