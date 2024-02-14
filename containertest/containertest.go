// Copyright 2023 The Authors (see AUTHORS file)
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

// Package containertest provides an ephemeral container (such as a database)
// for integration testing. It's designed to be used in code that needs to work
// inside and outside google.
//
// Deprecated: Use the standalone [github.com/abcxyz/containertest] package
// instead.
package containertest

import (
	"github.com/abcxyz/containertest"
)

// ConnInfo specifies how connect to the created container.
//
// Deprecated: This has moved to a new package. Use
// [github.com/abcxyz/containertest.ConnInfo] instead.
type ConnInfo = containertest.ConnInfo

// Service provides information about what container image should be started and
// how to know when it has finished stating up.
//
// Deprecated: This has moved to a new package. Use
// [github.com/abcxyz/containertest.Service] instead.
type Service = containertest.Service

// Start starts a container, or returns an error. On err ConnInfo will be
// automatically closed and nil will be returned.
//
// Deprecated: This has moved to a new package. Use
// [github.com/abcxyz/containertest.Start] instead.
var Start = containertest.Start
