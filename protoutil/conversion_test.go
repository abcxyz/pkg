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

// Package protoutil provides mechanisms for interacting with proto.
package protoutil

import (
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestParseProject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		input         any
		want          *structpb.Struct
		wantErrSubstr string
	}{
		{
			name: "success",
			input: map[string]any{
				"FieldA": "A",
				"FieldB": []string{"B1", "B2"},
				"FieldC": map[string]string{
					"keyC": "ValueC",
				},
			},
			want: &structpb.Struct{Fields: map[string]*structpb.Value{
				"FieldA": structpb.NewStringValue("A"),
				"FieldB": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("B1"), structpb.NewStringValue("B2")}}),
				"FieldC": structpb.NewStructValue(&structpb.Struct{
					Fields: map[string]*structpb.Value{
						"keyC": structpb.NewStringValue("ValueC"),
					},
				}),
			}},
		},
		{
			name: "failure_with_json_marshal",
			input: map[string]any{
				"FieldA": make(chan int),
			},
			wantErrSubstr: "json.Marshal: json: unsupported type: chan int",
		},
		{
			name:          "failure_with_structpb_unmarshal",
			input:         nil,
			wantErrSubstr: "structpb.Struct.UnmarshalJSON: proto:",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := ToProtoStruct(tc.input)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("ToProtoStruct(%+v) got diff (-want, +got): %v", tc.name, diff)
			}
		})
	}
}