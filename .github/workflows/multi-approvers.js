const APPROVED = 'APPROVED';
const COMMENTED = 'COMMENTED';
const MIN_APPROVED_COUNT = 2;

/** Implements approvedCount using the contains method. */
class AbstractMembers {

  /** Returns the number of approvals from members in the given list. */
  async approvedCount(submittedReviews, login) {
    const reviewStateByLogin = {};
    submittedReviews
      // Remove the PR user.
      .filter((r) => r.user.login !== prLogin)
      // Only consider users in the org.
      .filter((r) => this.contains(r.user.login))
      // Sort chronologically ascending. Note that a reviewer can submit multiple reviews.
      .sort((a, b) => new Date(a.submitted_at) - new Date(b.submitted_at))
      .forEach((r) => {
        const reviewerLogin = r.user.login;

        // Set state if it does not exist.
        if (!Object.hasOwn(reviewStateByLogin, reviewerLogin)) {
          reviewStateByLogin[reviewerLogin] = r.state;
          return;
        }

        // Always update state if not approved.
        if (reviewStateByLogin[reviewerLogin] !== APPROVED) {
          reviewStateByLogin[reviewerLogin] = r.state;
          return;
        }

        // Do not update approved state for a comment.
        if (reviewStateByLogin[reviewerLogin] === APPROVED && r.state !== COMMENTED) {
          reviewStateByLogin[reviewerLogin] = r.state;
        }
      })

    return Object.values(reviewStateByLogin).filter((s) => s === APPROVED).length;
  }
}

/** Members backed by a JSON file. */
class JsonMembers extends AbstractMembers {
  members;

  constructor(membersPath) {
    super();
    this.members = require(membersPath).reduce((acc, v) => acc.set(v.login, v), new Map());
  }

  async contains(login) {
    return this.members.has(login);
  }
}

/** Members backed by a GitHub team. */
class TeamMembers extends AbstractMembers {
  static ALLOWED_ROLES = ["maintainer", "member"];
  static ACTIVE = "active";

  org;
  teamSlug;
  github;

  constructor(org, teamSlug, github) {
    super();
    this.org = org;
    this.teamSlug = teamSlug;
    this.github = github;
  }

  async contains(login) {
    try {
      const response = await this.github.rest.teams.getMembershipForUserInOrg({
        org: this.org,
        team_slug: this.teamSlug,
        username: login,
      });
      return TeamMembers.ALLOWED_ROLES.indexOf(response.data.role) >= 0 &&
          response.data.state === TeamMembers.ACTIVE;
    } catch (error) {
      if (error.status === 404) {
        // We can get here for two reasons:
        // 1) The user is not a member
        // 2) The team does not exist
        // Either way, it's safe to return false here.
        return false;
      }
      throw error;
    }
  }
}

/** Checks that approval requirements are satisfied. */
async function onPullRequest({orgTeam, membersPath, prNumber, repoName, repoOwner, github, core}) {
  const members = (function() {
    if (orgTeam) {
      if (orgTeam.indexOf('/') <= 0) {
        throw new Error(`Malformed team [${orgTeam}]. Team must be in the format \${org}/\${team-slug}.`);
      }
      const teamOrg = orgTeam.split('/')[0];
      const teamSlug = orgTeam.split('/')[1];
      return new TeamMembers(teamOrg, teamSlug, github);
    }
    if (membersPath) {
      return new JsonMembers(membersPath);
    }
    throw new Error('Neither orgTeam nor membersPath is set.');
  })();

  const prResponse = await github.rest.pulls.get({owner: repoOwner, repo: repoName, pull_number: prNumber});
  const prLogin = prResponse.data.user.login;

  const isInternalPr = await members.contains(prLogin);
  if (isInternalPr) {
    // Do nothing if the pull request owner is an internal user.
    core.info(`Pull request login ${prLogin} is an internal member, therefore no special approval rules apply.`);
    return;
  }

  const submittedReviews = await github.paginate(github.rest.pulls.listReviews, {
    owner: repoOwner,
    repo: repoName,
    pull_number: prNumber,
  });

  const approvedCount = await members.approvedCount(submittedReviews, prLogin);

  core.info(`Found ${approvedCount} ${APPROVED} internal reviews.`);

  if (approvedCount < MIN_APPROVED_COUNT) {
    core.setFailed(`This pull request has ${approvedCount} of ${MIN_APPROVED_COUNT} required internal approvals.`);
  }
}

/**
 * Re-runs the approval checks on pull request review.
 *
 * This is required because GitHub treats checks made by pull_request and
 * pull_request_review as different status checks.
 */
async function onPullRequestReview({workflowRef, repoName, repoOwner, branch, prNumber, github}) {
  // Get the filename of the workflow.
  const workflowFilename = workflowRef.split('@')[0].split('/').pop();

  // Get all failed runs.
  const runs = await github.paginate(github.rest.actions.listWorkflowRuns, {
    owner: repoOwner,
    repo: repoName,
    workflow_id: workflowFilename,
    branch,
    event: 'pull_request',
    status: 'failure',
    per_page: 100,
  });

  const failedRuns = runs
    .filter((r) =>
      r.pull_requests.map((pr) => pr.number).includes(prNumber)
    )
    .sort((v) => v.id);

  // If there are failed runs for this PR, re-run the workflow.
  if (failedRuns.length > 0) {
    await github.rest.actions.reRunWorkflow({
      owner: repoOwner,
      repo: repoName,
      run_id: failedRuns[0].id,
    });
  }
}

module.exports = {onPullRequest, onPullRequestReview};
