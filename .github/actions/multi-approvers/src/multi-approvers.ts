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

import { getOctokit } from "@actions/github";
import { OctokitOptions } from "@octokit/core";
import { RequestError } from "@octokit/request-error";
import { components } from "@octokit/openapi-types";

type PullRequestReview = components["schemas"]["pull-request-review"];
type Octokit = ReturnType<typeof getOctokit>;
export type EventName = "pull_request" | "pull_request_review";

export function isEventName(v: string): v is EventName {
  return ["pull_request", "pull_request_review"].includes(v);
}

const MIN_APPROVED_COUNT = 2;

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
  // eslint-disable-next-line no-unused-vars
  logDebug: (_: string) => void;
  // eslint-disable-next-line no-unused-vars
  logInfo: (_: string) => void;
  // eslint-disable-next-line no-unused-vars
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
  // eslint-disable-next-line no-unused-vars
  private logDebug: (_: string) => void;

  // Set in the constructor.
  // eslint-disable-next-line no-unused-vars
  private logInfo: (_: string) => void;

  // Set in the constructor.
  // eslint-disable-next-line no-unused-vars
  private logNotice: (_: string) => void;

  // Tests whether the given login is an active member of the team.
  private async isInternal(login: string): Promise<boolean> {
    try {
      const response = await this.octokit.rest.teams.getMembershipForUserInOrg({
        org: this.repoOwner,
        team_slug: this.team,
        username: login,
      });
      const state = response.data.state;
      if (state !== "active") {
        this.logDebug(
          `Skipping because ${login} membership state is not active: ${state}`,
        );
        return false;
      }
      const role = response.data.role;
      if (role !== "maintainer" && role !== "member") {
        this.logDebug(
          `Skipping because ${login} membership role "${role}" is not in ["maintainer", "member"]`,
        );
        return false;
      }
      return true;
    } catch (err) {
      if (err instanceof RequestError && err.status === 404) {
        this.logDebug(
          `Received 404 testing membership; assuming ${login} is not a member: ${JSON.stringify(
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

  private async fetchReviews(): Promise<Array<PullRequestReview>> {
    return await this.octokit.paginate(this.octokit.rest.pulls.listReviews, {
      owner: this.repoOwner,
      repo: this.repoName,
      pull_number: this.pullNumber,
    });
  }

  /**
   * Group reviews by reviewer.
   *
   * This method assumes that all reviews have a user.
   *
   * Object.groupBy can be used once it becomes available. It would look like
   * this: Object.groupBy(reviews, (r) => r.user!.login);
   */
  private groupReviewsByReviewer(reviews: Array<PullRequestReview>): {
    [key: string]: Array<PullRequestReview>;
  } {
    return reviews.reduce((acc, r) => {
      const reviewerLogin = r.user!.login;
      let reviews = acc[reviewerLogin];
      if (reviews === undefined) {
        reviews = [];
        acc[reviewerLogin] = reviews;
      }
      reviews.push(r);
      return acc;
    }, Object.create(null));
  }

  /**
   * Returns the number of approvals from unique internal reviewers.
   *
   * Steps:
   *
   * 1. Fetch all reviews for the PR.
   * 2. Filter out reviews that do not have a `user` field (this case is not
   *    expected, but possible according to the type system).
   * 3. Filter out reviews from the PR author.
   * 4. Filter out comments -- we only care about approvals and change requests.
   * 5. Group reviews by reviewer.
   * 6. Ignore reviews by external reviewers.
   * 7. Sort all reviews by a single reviewer and get the last (non-comment)
   *    review status.
   * 8. Return the count of unique reviewers who's last (non-comment) review is
   *    approved.
   *
   * Written in functional pseudocode, this would look like this:
   *
   * fetchReviews()
   * .filter(r => !!r.user)
   * .filter(r => r.user.login !== prLogin)
   * .filter(r.state !== "COMMENTED")
   * .groupBy(r => r.user.login)
   * .filter([login, reviews] => isInternal(login))
   * .map([login, reviews] => [login, reviews.sort(r => r.submitted_at).reverse())
   * .map([login, reviews] => reviews[0].state)
   * .filter(s => s === "APPROVED")
   * .count();
   */
  private async internalApprovedCount(prLogin: string): Promise<number> {
    const reviews = (await this.fetchReviews())
      // Filter out reviews that do not have a user.
      .filter((r) => {
        if (!r.user) {
          this.logNotice(
            `Ignoring pull request review because user is unset: ${JSON.stringify(r)}`,
          );
          return false;
        }
        return true;
      })
      // Ignore the PR user.
      .filter((r) => {
        if (r.user!.login === prLogin) {
          this.logDebug(`Ignoring review from ${prLogin} (self)`);
          return false;
        }
        return true;
      })
      // Filter out comments -- we only care about APPROVED and CHANGES_REQUESTED.
      .filter((r) => r.state !== "COMMENTED");

    // Group reviews by reviewer.
    const groups = this.groupReviewsByReviewer(reviews);

    let approvedCount = 0;
    for (const [reviewerLogin, reviews] of Object.entries(groups)) {
      // Only consider internal reviewers.
      const isInternal = await this.isInternal(reviewerLogin);
      if (!isInternal) {
        this.logDebug(
          `Ignoring reviewer ${reviewerLogin} because they are not a member of ${this.team}`,
        );
        continue;
      }

      // Get the most recent (non-comment) review by this reviewer.
      const mostRecentReview = reviews
        .sort(
          (a: any, b: any) =>
            new Date(a.submitted_at || 0).getTime() -
            new Date(b.submitted_at || 0).getTime(),
        )
        .reverse()[0];

      if (mostRecentReview.state === "APPROVED") {
        approvedCount++;
      }
    }

    return approvedCount;
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

    const approvedCount = await this.internalApprovedCount(prLogin);

    this.logInfo(`Found ${approvedCount} APPROVED internal reviews.`);

    if (approvedCount < MIN_APPROVED_COUNT) {
      throw new Error(
        `This pull request has ${approvedCount} of ${
          MIN_APPROVED_COUNT
        } required internal approvals.`,
      );
    }
  }

  /**
   * Re-runs the most-recent pull_request-triggered workflow, if one exists.
   *
   * This is required because GitHub treats checks made by pull_request and
   * pull_request_review as different status checks.
   *
   * Without this, the "multi-approvers (pull_request_review)" and
   * "multi-approvers (pull_request)" checks would get out of sync.
   */
  private async rerunLatestPullRequestWorkflow() {
    const workflowId = await this.getWorkflowId();
    // Get all potential runs.
    const runs = await this.octokit.paginate(
      this.octokit.rest.actions.listWorkflowRuns,
      {
        owner: this.repoOwner,
        repo: this.repoName,
        workflow_id: workflowId,
        branch: this.branch,
        event: "pull_request",
        per_page: 100,
      },
    );

    const prRuns = runs
      // Remove any runs not associated with this.pullNumber.
      .filter((r) =>
        (r.pull_requests || [])
          .map((pr) => pr.number)
          .includes(this.pullNumber),
      )
      .sort((a, b) => a.run_number - b.run_number)
      // Reverse the array so the latest is at the head.
      .reverse();

    // Re-run the latest if there is one.
    if (prRuns.length > 0) {
      await this.octokit.rest.actions.reRunWorkflow({
        owner: this.repoOwner,
        repo: this.repoName,
        run_id: prRuns[0].id,
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
    if (this.eventName === "pull_request_review") {
      // Re-run the latest pull_request-triggered workflow to keep the checks
      // (pull_request and pull_request_review)in sync.
      await this.rerunLatestPullRequestWorkflow();
    }

    await this.validateApprovers();
  }
}
