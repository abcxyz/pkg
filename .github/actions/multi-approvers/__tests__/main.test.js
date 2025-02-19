const core = require('@actions/core');
const github = require('@actions/github');

jest.mock('@actions/core');
jest.mock('@actions/github');

/** Returns a GitHub Actions context. **/
function getGhContext({
    eventName = 'pull_request',
    runId = 1,
    prNumber = 1,
    branch = 'my-test/branch',
    repoName = 'test-repo',
    org = 'test-org'} = {}) {
  return {
    eventName,
    runId,
    payload: {
      pull_request: {
        number: prNumber,
        head: {
          ref: branch,
        },
      },
      repository: {
        name: repoName,
        owner: {
          login: org,
        },
      },
    },
  };
}

describe('main', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('should fail on unsupported event', async () => {
    core.getInput.mockImplementation((name) => {
      if (name === 'token') {
        return 'a-fake-token';
      }
      if (name === 'team') {
        return 'a-fake-team';
      }
      return jest.requireActual('@actions/core').getInput(name);
    });
    github.context = getGhContext({eventName: 'push'});
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Unexpected event [push]. Supported events are pull_request, pull_request_review"));
  });

  it('should fail when no inputs are set', async () => {
    github.context = getGhContext();
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Invalid input(s): token is required; team is required"));
  });

  it('should fail when token is not set', async () => {
    core.getInput.mockImplementation((name) => {
      if (name === 'token') {
        return 'a-fake-token';
      }
      return jest.requireActual('@actions/core').getInput(name);
    });
    github.context = getGhContext();
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Invalid input(s): team is required"));
  });

  it('should fail when team is not set', async () => {
    core.getInput.mockImplementation((name) => {
      if (name === 'team') {
        return 'a-fake-team';
      }
      return jest.requireActual('@actions/core').getInput(name);
    });
    github.context = getGhContext();
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Invalid input(s): token is required"));
  });
});
