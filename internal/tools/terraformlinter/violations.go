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

package terraformlinter

import (
	"cmp"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ViolationInstance is an object that contains a reference to a location
// in a file where a lint violation was detected.
type ViolationInstance struct {
	Message string
	Path    string
	Line    int
}

// ViolationInstanceSorter is a function that orders a slice of
// [ViolationInstance].
var ViolationInstanceSorter = func(a, b *ViolationInstance) int {
	if v := cmp.Compare(a.Path, b.Path); v != 0 {
		return v
	}
	if v := cmp.Compare(a.Line, b.Line); v != 0 {
		return v
	}
	return cmp.Compare(a.Message, b.Message)
}

func newViolation(token hclsyntax.Token, message string) *ViolationInstance {
	return &ViolationInstance{
		Message: message,
		Path:    token.Range.Filename,
		Line:    token.Range.Start.Line,
	}
}

func newLeadingMetaBlockAttributeViolation(token hclsyntax.Token, attr string) *ViolationInstance {
	message := fmt.Sprintf(`The attribute %q must be in the meta block at the top of the definition.`, attr)
	return newViolation(token, message)
}

func newMetaBlockNewlineViolation(token hclsyntax.Token) *ViolationInstance {
	message := `The meta block must have an additional newline separating it from the next section.`
	return newViolation(token, message)
}

func newProviderAttributesViolation(token hclsyntax.Token, attr string) *ViolationInstance {
	message := fmt.Sprintf(`The attribute %q must be below any meta attributes (e.g. "for_each", "count") but above all other attributes. Attributes must be ordered organization > folder > project.`, attr)
	return newViolation(token, message)
}

func newProviderNewlineViolation(token hclsyntax.Token, attr string) *ViolationInstance {
	message := `The provider specific attributes must have an additional newline separating it from the next section.`
	return newViolation(token, message)
}

func newTrailingMetaBlockAttributeViolation(token hclsyntax.Token, attr string) *ViolationInstance {
	message := fmt.Sprintf(`The attribute %q must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`, attr)
	return newViolation(token, message)
}

func newHyphenInNameViolation(token hclsyntax.Token, attr string) *ViolationInstance {
	message := fmt.Sprintf(`The resource %q must not contain a "-" in its name.`, attr)
	return newViolation(token, message)
}
