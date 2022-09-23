// Copyright 2022 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestContext(t *testing.T) {
	t.Parallel()

	logger1 := zap.NewNop().Sugar()
	logger2 := zap.NewExample().Sugar()

	checkFromContext(context.Background(), t, Default())

	ctx := WithLogger(context.Background(), logger1)
	checkFromContext(ctx, t, logger1)

	ctx = WithLogger(ctx, logger2)
	checkFromContext(ctx, t, logger2)
}

func checkFromContext(ctx context.Context, tb testing.TB, want *zap.SugaredLogger) {
	tb.Helper()

	if got := FromContext(ctx); want != got {
		tb.Errorf("unexpected logger in context. got: %v, want: %v", got, want)
	}
}
