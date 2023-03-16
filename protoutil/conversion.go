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
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

// ToProtoStruct converts v, which must marshal into a JSON object, into a proto
// struct.
func ToProtoStruct(v any) (*structpb.Struct, error) {
	jb, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}
	x := &structpb.Struct{}
	if err := x.UnmarshalJSON(jb); err != nil {
		return nil, fmt.Errorf("structpb.Struct.UnmarshalJSON: %w", err)
	}
	return x, nil
}

// UnmarshalYAML unmarshals the give YAML bytes to the given proto message.
func UnmarshalYAML(b []byte, msg proto.Message) error {
	tmp := map[string]any{}
	if err := yaml.Unmarshal(b, tmp); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	jb, err := json.Marshal(tmp)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	if err := protojson.Unmarshal(jb, msg); err != nil {
		return fmt.Errorf("failed to unmarshal proto: %w", err)
	}
	return nil
}
