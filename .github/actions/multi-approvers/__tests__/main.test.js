/**
 * Copyright 2024 The Authors (see AUTHORS file)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const core = require('@actions/core');
const github = require('@actions/github');

class FakeOctokit {
  static EMPTY_MODEL = {
    pulls: [],
    teams: [],
  };

  model;

  constructor(model) {
    // The logic in this class depends on at least the EMPTY_MODEL being
    // defined.
    //
    // Note that the splat operator is not recursive, if more than
    // top-level members are required, a recursive application of the splat
    // operator will have to be used.
    this.model = {...FakeOctokit.EMPTY_MODEL, ...model};
  }

  paginate(fn, ...args) {
    return fn(...args);
  }

  rest = {
    pulls: {
      get: ({owner, repo, pull_number}) => {
        const data = this.model.pulls.find(
            (v) => v.owner === owner && v.repo === repo &&
                v.pull_number === pull_number);
        if (data) {
          return {
            data,
          };
        }
        throw {status: 404};
      },
      listReviews:
          ({owner, repo, pull_number}) => {
            const {data} = this.rest.pulls.get({owner, repo, pull_number});
            return data.reviews || [];
          },
    },
    teams: {
      getMembershipForUserInOrg:
          ({org, team_slug, username}) => {
            const data =
                (this.model.teams || [])
                    .find(
                        (v) => v.org === org && v.team_slug === team_slug &&
                            v.username === username);
            if (data) {
              return {
                data,
              };
            }
            throw {status: 404};
          },
    },
  };
}

jest.mock('@actions/core');

function setup(v) {
  core.getInput.mockImplementation((name) => {
    if (Object.hasOwn(v.inputs, name)) {
      return v.inputs[name];
    }
    return jest.requireActual('@actions/core').getInput(name);
  });
  github.context = v.github.context;
  github.getOctokit = () => new FakeOctokit(v.model);
}

describe('main', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('should fail on unsupported event', async () => {
    setup(require('./data/unsupported-event.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(new Error(
            'Unexpected event [push]. Supported events are pull_request, pull_request_review'));
  });

  it('should fail when no inputs are set', async () => {
    setup(require('./data/no-inputs.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(
            new Error('Invalid input(s): token is required; team is required'));
  });

  it('should fail when token is not set', async () => {
    setup(require('./data/token-not-set.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(new Error('Invalid input(s): token is required'));
  });

  it('should fail when team is not set', async () => {
    setup(require('./data/team-not-set.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(new Error('Invalid input(s): team is required'));
  });

  it('should ignore PRs from internal users', async () => {
    setup(require('./data/internal-pr.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed).not.toHaveBeenCalled();
  });

  it('should reject PRs from external users and no internal approvals',
     async () => {
       setup(require('./data/no-internal-approvals.json'));
       const {main} = require('../src/main');

       await main();

       expect(core.setFailed)
           .toHaveBeenCalledWith(
               'This pull request has 0 of 2 required internal approvals.');
     });

  it('should succeed for PRs from external users and 2 internal approvals',
     async () => {
       setup(require('./data/two-internal-approvals.json'));
       const {main} = require('../src/main');

       await main();

       expect(core.setFailed).not.toHaveBeenCalled();
     });

  it('should ignore PR review comments', async () => {
    setup(require('./data/ignore-pr-review-comments.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(
            'This pull request has 1 of 2 required internal approvals.');
  });

  it('should handle rescinded approval', async () => {
    setup(require('./data/rescinded-approval.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(
            'This pull request has 1 of 2 required internal approvals.');
  });

  it('should fail with pending member approval', async () => {
    setup(require('./data/pending-member-approval.json'));
    const {main} = require('../src/main');

    await main();

    expect(core.setFailed)
        .toHaveBeenCalledWith(
            'This pull request has 1 of 2 required internal approvals.');
  });
});
