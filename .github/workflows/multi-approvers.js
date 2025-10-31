const APPROVED = 'APPROVED';
const COMMENTED = 'COMMENTED';
const MIN_APPROVED_COUNT = 2;

/** Returns the number of approvals from members in the given list. */
function inOrgApprovedCount(members, submittedReviews, prLogin) {
  const reviewStateByLogin = {};
  submittedReviews
    // Remove the PR user.
    .filter((r) => r.user.login !== prLogin)
    // Only consider users in the org.
    .filter((r) => members.has(r.user.login))
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

/** Checks that approval requirements are satisfied. */
async function onPullRequest({orgMembersPath, prNumber, repoName, repoOwner, github, core}) {
  core.warning("This workflow is deprecated. Please migrate to the new multi-approvers action found at https://github.com/abcxyz/actions/tree/main/.github/actions/multi-approvers.");

  const members = require(orgMembersPath).reduce((acc, v) => acc.set(v.login, v), new Map());
  const prResponse = await github.rest.pulls.get({owner: repoOwner, repo: repoName, pull_number: prNumber});
  const prLogin = prResponse.data.user.login;

  if (members.has(prLogin)) {
    // Do nothing if the pull request owner is a member of the org.
    core.info(`Pull request login ${prLogin} is a member of the org, therefore no special approval rules apply.`);
    return;
  }

  const submittedReviews = await github.paginate(github.rest.pulls.listReviews, {
    owner: repoOwner,
    repo: repoName,
    pull_number: prNumber,
  });

  const approvedCount = inOrgApprovedCount(members, submittedReviews, prLogin);

  core.info(`Found ${approvedCount} ${APPROVED} reviews.`);

  if (approvedCount < MIN_APPROVED_COUNT) {
    core.setFailed(`This pull request has ${approvedCount} of ${MIN_APPROVED_COUNT} required approvals from members of the org.`);
  }
}

/**
 * Re-runs the approval checks on pull request review.
 *
 * This is required because GitHub treats checks made by pull_request and
 * pull_request_review as different status checks.
 */
async function onPullRequestReview({workflowRef, repoName, repoOwner, branch, prNumber, github, core}) {
  core.warning("This workflow is deprecated. Please migrate to the new multi-approvers action found at https://github.com/abcxyz/actions/tree/main/.github/actions/multi-approvers.");

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
    .sort((a, b) => b.id - a.id);

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
