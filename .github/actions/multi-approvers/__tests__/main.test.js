/**
 * Copyright 2024 The Authors (see AUTHORS file)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 * * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const core = require('@actions/core');
const github = require('@actions/github');

jest.mock('@actions/core');
jest.mock('@actions/github');

function mockCoreGetInput(inputs = {}) {
    core.getInput.mockImplementation((name) => {
      if (Object.hasOwn(inputs, name)) {
        return inputs[name];
      }
      return jest.requireActual('@actions/core').getInput(name);
    });
}

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
    mockCoreGetInput({token: 'fake-token', team: 'fake-team'});
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
    mockCoreGetInput({team: 'fake-team'});
    github.context = getGhContext();
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Invalid input(s): token is required"));
  });

  it('should fail when team is not set', async () => {
    mockCoreGetInput({token: 'fake-token'});
    github.context = getGhContext();
    const { main } = require('../src/main');

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(new Error("Invalid input(s): team is required"));
  });
});
