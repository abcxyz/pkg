// Copyright 2023 The Authors (see AUTHORS file)
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

package exp

import (
	"time"

	"golang.org/x/exp/slog"
)

// Attr is an alias of [slog.Attr].
type Attr = slog.Attr

// Any is a local reference to [slog.Any].
func Any(key string, v any) Attr {
	return slog.Any(key, v)
}

// Bool is a local reference to [slog.Bool].
func Bool(key string, v bool) Attr {
	return slog.Bool(key, v)
}

// Duration is a local reference to [slog.Duration].
func Duration(key string, v time.Duration) Attr {
	return slog.Duration(key, v)
}

// Float64 is a local reference to [slog.Float64].
func Float64(key string, v float64) Attr {
	return slog.Float64(key, v)
}

// Group is a local reference to [slog.Group].
func Group(key string, as ...any) Attr {
	return slog.Group(key, as...)
}

// Int is a local reference to [slog.Int].
func Int(key string, v int) Attr {
	return slog.Int(key, v)
}

// Int64 is a local reference to [slog.Int64].
func Int64(key string, v int64) Attr {
	return slog.Int64(key, v)
}

// String is a local reference to [slog.String].
func String(key, v string) Attr {
	return slog.String(key, v)
}

// Time is a local reference to [slog.Time].
func Time(key string, v time.Time) Attr {
	return slog.Time(key, v)
}

// Uint64 is a local reference to [slog.Uint64].
func Uint64(key string, v uint64) Attr {
	return slog.Uint64(key, v)
}

// Record is an alias of [slog.Record].
type Record = slog.Record

// New record is local reference to [slog.NewRecord].
func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return slog.NewRecord(t, level, msg, pc)
}

// Value is an alias of [slog.Value].
type Value = slog.Value

// AnyValue is a local reference to [slog.AnyValue].
func AnyValue(v any) Value {
	return slog.AnyValue(v)
}

// BoolValue is a local reference to [slog.BoolValue].
func BoolValue(v bool) Value {
	return slog.BoolValue(v)
}

// DurationValue is a local reference to [slog.DurationValue].
func DurationValue(v time.Duration) Value {
	return slog.DurationValue(v)
}

// Float64Value is a local reference to [slog.Float64Value].
func Float64Value(v float64) Value {
	return slog.Float64Value(v)
}

// GroupValue is a local reference to [slog.GroupValue].
func GroupValue(as ...Attr) Value {
	return slog.GroupValue(as...)
}

// Int64Value is a local reference to [slog.Int64Value].
func Int64Value(v int64) Value {
	return slog.Int64Value(v)
}

// IntValue is a local reference to [slog.IntValue].
func IntValue(v int) Value {
	return slog.IntValue(v)
}

// StringValue is a local reference to [slog.StringValue].
func StringValue(v string) Value {
	return slog.StringValue(v)
}

// TimeValue is a local reference to [slog.TimeValue].
func TimeValue(v time.Time) Value {
	return slog.TimeValue(v)
}

// Uint64Value is a local reference to [slog.Uint64Value].
func Uint64Value(v uint64) Value {
	return slog.Uint64Value(v)
}
