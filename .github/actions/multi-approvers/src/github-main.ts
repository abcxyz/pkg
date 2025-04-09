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

// The purpose of this file is to kick off the action from a GitHub context
// (i.e. using the @actions/core.* and @actions/github.context).

import * as ghCore from "@actions/core";
import { context as ghContext } from "@actions/github";
import { errorMessage } from "@google-github-actions/actions-utils";
import {
  EventName,
  isEventName,
  MultiApproversAction,
} from "./multi-approvers";

type Core = typeof ghCore;
type Context = typeof ghContext;

function getEventName(rawEventName: string): EventName {
  if (!isEventName(rawEventName)) {
    throw new Error(`Unexpected event [${rawEventName}].`);
  }
  return rawEventName as EventName;
}

export async function main(core: Core = ghCore, context: Context = ghContext) {
  try {
    const payload = context.payload;
    const token = core.getInput("token", { required: true });
    const team = core.getInput("team", { required: true });
    const eventName = getEventName(context.eventName);

    const multiApproversAction = new MultiApproversAction({
      eventName: eventName,
      runId: context.runId,
      branch: payload.pull_request!.head.ref,
      pullNumber: payload.pull_request!.number,
      repoName: payload.repository!.name,
      repoOwner: payload.repository!.owner.login,
      token: token,
      team: team,
      logDebug: core.debug,
      logInfo: core.info,
      logNotice: core.notice,
    });

    await multiApproversAction.validate();
  } catch (err) {
    core.setFailed(`Multi-approvers action failed: ${errorMessage(err)}`);
  }
}
