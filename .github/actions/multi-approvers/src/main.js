/**
 * Copyright 2024 The Authors (see AUTHORS file)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const core = require('@actions/core');
const github = require('@actions/github');

const PULL_REQUEST_REVIEW = 'pull_request_review';
const PULL_REQUEST = 'pull_request';
const SUPPORTED_EVENTS = [PULL_REQUEST, PULL_REQUEST_REVIEW];
const APPROVED = 'APPROVED';
const COMMENTED = 'COMMENTED';
const MIN_APPROVED_COUNT = 2;

/** Members backed by a GitHub team. */
class TeamMembers {
  static ALLOWED_ROLES = ['maintainer', 'member'];
  static ACTIVE = 'active';

  org;
  teamSlug;
  octokit;

  constructor(org, teamSlug, octokit) {
    this.org = org;
    this.teamSlug = teamSlug;
    this.octokit = octokit;
  }

  async contains(login) {
    try {
      const response = await this.octokit.rest.teams.getMembershipForUserInOrg({
        org: this.org,
        team_slug: this.teamSlug,
        username: login,
      });
      return TeamMembers.ALLOWED_ROLES.indexOf(response.data.role) >= 0 &&
          response.data.state === TeamMembers.ACTIVE;
    } catch (err) {
      if (err.status === 404) {
        core.debug(
            `Received 404 testing membership; assuming user is not a member: ${
                JSON.stringify(err)}`);
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

  /** Returns the number of approvals from members in the given list. */
  async approvedCount(submittedReviews, prLogin) {
    // Sort by chronological order.
    const sortedReviews = submittedReviews.sort(
        (a, b) => new Date(a.submitted_at) - new Date(b.submitted_at));
    const reviewStateByLogin = {};

    for (const r of sortedReviews) {
      const reviewerLogin = r.user.login;

      // Ignore the PR user.
      if (reviewerLogin === prLogin) {
        continue;
      }

      // Only consider internal users.
      const isInternalUser = await this.contains(reviewerLogin);
      if (!isInternalUser) {
        continue;
      }

      // Set state if it does not exist.
      if (!Object.hasOwn(reviewStateByLogin, reviewerLogin)) {
        reviewStateByLogin[reviewerLogin] = r.state;
        continue;
      }

      // Always update state if not approved.
      if (reviewStateByLogin[reviewerLogin] !== APPROVED) {
        reviewStateByLogin[reviewerLogin] = r.state;
        continue;
      }

      // Do not update approved state for a comment.
      if (reviewStateByLogin[reviewerLogin] === APPROVED &&
          r.state !== COMMENTED) {
        reviewStateByLogin[reviewerLogin] = r.state;
        continue;
      }
    }

    return Object.values(reviewStateByLogin)
        .filter((s) => s === APPROVED)
        .length;
  }
}

/** Checks that approval requirements are satisfied. */
async function validateApprovers(
    {team, prNumber, repoName, repoOwner, octokit}) {
  const members = new TeamMembers(repoOwner, team, octokit);
  const prResponse = await octokit.rest.pulls.get(
      {owner: repoOwner, repo: repoName, pull_number: prNumber});
  const prLogin = prResponse.data.user.login;

  const isInternalPr = await members.contains(prLogin);
  if (isInternalPr) {
    // Do nothing if the pull request owner is an internal user.
    core.info(`Pull request login ${
        prLogin} is a member of the org, therefore no special approval rules apply.`);
    return;
  }
  const submittedReviews =
      await octokit.paginate(octokit.rest.pulls.listReviews, {
        owner: repoOwner,
        repo: repoName,
        pull_number: prNumber,
      });

  const approvedCount = await members.approvedCount(submittedReviews, prLogin);

  core.info(`Found ${approvedCount} ${APPROVED} internal reviews.`);

  if (approvedCount < MIN_APPROVED_COUNT) {
    core.setFailed(`This pull request has ${approvedCount} of ${
        MIN_APPROVED_COUNT} required internal approvals.`);
  }
}

/**
 * Re-runs the approval checks on pull request review.
 *
 * This is required because GitHub treats checks made by pull_request and
 * pull_request_review as different status checks.
 */
async function revalidateApprovers(
    {workflowId, repoName, repoOwner, branch, prNumber, octokit}) {
  // Get all failed runs.
  const runs = await octokit.paginate(octokit.rest.actions.listWorkflowRuns, {
    owner: repoOwner,
    repo: repoName,
    workflow_id: workflowId,
    branch,
    event: 'pull_request',
    status: 'failure',
    per_page: 100,
  });

  const failedRuns =
      runs.filter(
              (r) => r.pull_requests.map((pr) => pr.number).includes(prNumber))
          .sort((v) => v.id);

  // If there are failed runs for this PR, re-run the workflow.
  if (failedRuns.length > 0) {
    await octokit.rest.actions.reRunWorkflow({
      owner: repoOwner,
      repo: repoName,
      run_id: failedRuns[0].id,
    });
  }
}

async function getWorkflowId(octokit, repoOwner, repoName, runId) {
  const response = await octokit.rest.actions.getWorkflowRun(
      {owner: repoOwner, repo: repoName, run_id: runId});
  return response.data.workflow_id;
}

function validateInputs({token, team}) {
  const errors = [];
  if (!token) {
    errors.push('token is required');
  }
  if (!team) {
    errors.push('team is required');
  }
  if (errors.length > 0) {
    throw new Error(`Invalid input(s): ${errors.join('; ')}`);
  }
}

function validateEvent(eventName) {
  if (SUPPORTED_EVENTS.indexOf(eventName) < 0) {
    throw new Error(`Unexpected event [${eventName}]. Supported events are ${
        SUPPORTED_EVENTS.join(', ')}`);
  }
}

async function main() {
  try {
    const eventName = github.context.eventName;
    const runId = github.context.runId;
    const payload = github.context.payload;
    const branch = payload.pull_request.head.ref;
    const prNumber = payload.pull_request.number;
    const repoName = payload.repository.name;
    const repoOwner = payload.repository.owner.login;
    const token = core.getInput('token');
    const team = core.getInput('team');
    const octokit = github.getOctokit(token);

    validateEvent(eventName);
    validateInputs({token, team});

    await validateApprovers({team, prNumber, repoName, repoOwner, octokit});

    // If this action was triggered by a review, we want to re-run for previous
    // failed runs.
    if (eventName === PULL_REQUEST_REVIEW) {
      const workflowId =
          await getWorkflowId(octokit, repoOwner, repoName, runId);
      await revalidateApprovers(
          {workflowId, branch, repoName, repoOwner, prNumber, octokit});
    }
  } catch (err) {
    core.debug(JSON.stringify(err));
    core.setFailed(err);
  }
}

module.exports = {main};
