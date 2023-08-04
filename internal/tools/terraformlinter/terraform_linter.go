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

// Package terraformlinter contains a linter implementation that verifies terraform
// files against our internal style guide and reports on all violations.
package terraformlinter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Top level terraform types to validate.
const (
	tokenTypeResource = "resource"
	tokenTypeModule   = "module"
	tokenTypeVariable = "variable"
	tokenTypeOutput   = "output"
	tokenTypeLocals   = "locals"
	tokenTypeImport   = "import"
)

// List of valid extensions that can be linted.
var terraformSelectors = []string{".tf", ".tf.json"}

// Enum of positional locations in order.
type tokenPosition int32

const (
	None tokenPosition = iota
	LeadingStart
	LeadingEnd
	ProviderStart
	ProviderCenter
	ProviderEnd
	Ignored
	Trailing
)

// tokenAttr defines an individual attribute within a block of terraform.
type tokenAttr struct {
	tokenPos        tokenPosition
	token           hclsyntax.Token
	trailingNewline bool
}

// keywords to match on.
const (
	attrForEach                = "for_each"
	attrCount                  = "count"
	attrProvider               = "provider"
	attrSource                 = "source"
	attrProviderProject        = "project"
	attrProviderProjectID      = "project_id"
	attrProviderFolder         = "folder"
	attrProviderFolderID       = "folder_id"
	attrProviderOrganization   = "organization"
	attrProviderOrganizationID = "organization_id"
	attrProviderOrgID          = "org_id"
	attrDependsOn              = "depends_on"
	attrLifecycle              = "lifecycle"
)

// mapping of attributes to their expected position.
var positionMap = map[string]tokenPosition{
	attrForEach:                LeadingStart,
	attrCount:                  LeadingStart,
	attrSource:                 LeadingStart,
	attrProvider:               LeadingEnd,
	attrProviderProject:        ProviderEnd,
	attrProviderProjectID:      ProviderEnd,
	attrProviderFolder:         ProviderCenter,
	attrProviderFolderID:       ProviderCenter,
	attrProviderOrganization:   ProviderStart,
	attrProviderOrganizationID: ProviderStart,
	attrProviderOrgID:          ProviderStart,
	attrDependsOn:              Trailing,
	attrLifecycle:              Trailing,
}

const (
	violationLeadingMetaBlockAttribute  = "The attribute %q must be in the meta block at the top of the definition."
	violationMetaBlockNewline           = "The meta block must have an additional new line separating it from the next section."
	violationProviderAttributes         = "The attribute %q must me below any meta attributes (for_each, count, etc.) but above all other attributes. Attributes should be ordered organization > folder > project."
	violationProviderNewline            = "The provider specific attributes must have an additional new line separating it from the next section."
	violationTrailingMetaBlockAttribute = `The attribute %q must be at the bottom of the resource definition and in the order "depends_on" then "lifecycle."`
	violationHyphenInResouceName        = `The resource %q must not contain a "-" in its name`
)

// ViolationInstance is an object that contains a reference to a location
// in a file where a lint violation was detected.
type ViolationInstance struct {
	ViolationType string
	Path          string
	Line          int
}

// RunLinter executes the specified linter for a set of files.
func RunLinter(paths []string) error {
	var violations []*ViolationInstance
	// Process each provided path looking for violations
	for _, path := range paths {
		instances, err := lint(path)
		if err != nil {
			return fmt.Errorf("error linting files at %q: %w", path, err)
		}
		violations = append(violations, instances...)
	}
	for _, instance := range violations {
		// Output as errorformat "%f:%l: %m" (file:line: message)
		fmt.Printf("%s:%d: %s\n", instance.Path, instance.Line, instance.ViolationType)
	}
	if len(violations) > 0 {
		return fmt.Errorf("found %d violation(s)", len(violations))
	}

	return nil
}

// lint reads a path and determines if it is a file or a directory.
// When it finds a file it reads it and checks it for violations.
// When it finds a directory it calls itself recursively.
func lint(path string) ([]*ViolationInstance, error) {
	instances := []*ViolationInstance{}
	if err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		for _, sel := range terraformSelectors {
			if strings.HasSuffix(path, sel) {
				content, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading file %q: %w", path, err)
				}
				results, err := findViolations(content, path)
				if err != nil {
					return fmt.Errorf("error linting file %q: %w", path, err)
				}
				instances = append(instances, results...)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error walking path %q: %w", path, err)
	}
	return instances, nil
}

// findViolations inspects a set of bytes that represent hcl from a terraform configuration file
// looking for attributes of a resource and ensuring that the ordering matches our style guide.
func findViolations(content []byte, path string) ([]*ViolationInstance, error) {
	tokens, diags := hclsyntax.LexConfig(content, path, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags.HasErrors() {
		// diags.Error is just a string, but the golangci linter gets angry that we aren't using
		// %w in the error message. Attempts to use the nolint tag also get flagged as not needed
		// in newer versions so to appease the linter we wrap the string in an error.
		return nil, fmt.Errorf("error lexing hcl file contents: [%w]", errors.New(diags.Error()))
	}

	inBlock := false
	depth, start := 0, 0
	var instances []*ViolationInstance
	// First break apart the terraform into the major blocks of resources / modules
	for idx, token := range tokens {
		if token.Bytes == nil {
			continue
		}
		contents := string(token.Bytes)
		// Each Ident token starts a new object, we are only looking for resource, module, output and variable types
		if !inBlock && token.Type == hclsyntax.TokenIdent &&
			(contents == tokenTypeResource ||
				contents == tokenTypeModule ||
				contents == tokenTypeOutput ||
				contents == tokenTypeVariable ||
				contents == tokenTypeLocals ||
				contents == tokenTypeImport) {
			inBlock = true
			start = idx
			depth = 0
		}
		// If we are in a block, look for the closing braces to find the end
		if inBlock {
			// Before dropping into the block itself, look for names that have a hyphen
			if depth == 0 && token.Type == hclsyntax.TokenQuotedLit {
				if strings.Contains(contents, "-") {
					instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationHyphenInResouceName, contents), Path: token.Range.Filename, Line: token.Range.Start.Line})
				}
			}
			if token.Type == hclsyntax.TokenOBrace {
				depth = depth + 1
			}
			if token.Type == hclsyntax.TokenCBrace {
				depth = depth - 1
				// Last brace signals the end of the entire block
				if depth == 0 {
					inBlock = false
					// Validate the block against the rules
					results := validateBlock(tokens[start : idx+1])
					instances = append(instances, results...)
				}
			}
		}
	}
	return instances, nil
}

// validateBlock scans a block of terraform looking for violations
// of our style guide.
func validateBlock(tokens hclsyntax.Tokens) []*ViolationInstance {
	var attrs []tokenAttr
	var token hclsyntax.Token
	for len(tokens) > 0 {
		// Pop the first token off
		token, tokens = tokens[0], tokens[1:]
		contents := string(token.Bytes)
		if token.Type == hclsyntax.TokenIdent {
			if contents == tokenTypeModule || contents == tokenTypeResource {
				continue
			}
			var t hclsyntax.Token
			skipping := true
			depth := 0
			// while there are tokens to skip and we haven't exceeded the length of the slice
			for skipping && len(tokens) > 1 {
				t, tokens = tokens[0], tokens[1:]
				if t.Type == hclsyntax.TokenOBrace || t.Type == hclsyntax.TokenOBrack {
					depth = depth + 1
				}
				if t.Type == hclsyntax.TokenCBrace || t.Type == hclsyntax.TokenCBrack {
					depth = depth - 1
				}
				if depth == 0 && (t.Type == hclsyntax.TokenNewline || t.Type == hclsyntax.TokenComment) {
					// Check for an extra newline
					trailingNewline := false
					if len(tokens) > 0 && tokens[0].Type == hclsyntax.TokenNewline {
						trailingNewline = true
					}
					position, ok := positionMap[contents]
					if !ok {
						position = Ignored
					}
					attrs = append(attrs, tokenAttr{tokenPos: position, token: token, trailingNewline: trailingNewline})
					skipping = false
				}
				// Reached the end of the file
				if len(tokens) < 2 {
					skipping = false
				}
			}
		}
	}
	return generateViolations(attrs)
}

func generateViolations(idents []tokenAttr) []*ViolationInstance {
	var instances []*ViolationInstance
	var lastAttr tokenAttr
	for pos, token := range idents {
		contents := string(token.token.Bytes)
		switch contents {
		// for_each, count and source should be at the top
		case attrForEach, attrCount, attrSource:
			if pos != 0 && lastAttr.tokenPos != LeadingStart {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, contents), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		// provider is at the top but below for_each or count if they exist
		case attrProvider:
			if pos > 0 && lastAttr.tokenPos != LeadingStart {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationLeadingMetaBlockAttribute, attrProvider), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		case attrDependsOn:
			// depends_on somewhere above where it should be
			if pos < len(idents)-1 && idents[len(idents)-1].tokenPos != Trailing {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrDependsOn), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
			// depends_on after lifecycle
			if pos == len(idents)-1 && lastAttr.tokenPos == Trailing {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrDependsOn), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		case attrLifecycle:
			// lifecycle should be last
			if pos != len(idents)-1 {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationTrailingMetaBlockAttribute, attrLifecycle), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		// All provider specific entries follow the same logic. Should be below the metadata segment and above everything else
		// Expect order
		//   organization
		//   folder
		//   project
		case attrProviderOrganization,
			attrProviderOrganizationID,
			attrProviderOrgID:
			if lastAttr.tokenPos > ProviderStart {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationProviderAttributes, contents), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
			if (lastAttr.tokenPos == LeadingStart || lastAttr.tokenPos == LeadingEnd) && !lastAttr.trailingNewline {
				instances = append(instances, &ViolationInstance{ViolationType: violationMetaBlockNewline, Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		case attrProviderFolder,
			attrProviderFolderID:
			if lastAttr.tokenPos > ProviderCenter {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationProviderAttributes, contents), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
			if (lastAttr.tokenPos == LeadingStart || lastAttr.tokenPos == LeadingEnd) && !lastAttr.trailingNewline {
				instances = append(instances, &ViolationInstance{ViolationType: violationMetaBlockNewline, Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		case attrProviderProject,
			attrProviderProjectID:
			if lastAttr.tokenPos > ProviderEnd {
				instances = append(instances, &ViolationInstance{ViolationType: fmt.Sprintf(violationProviderAttributes, contents), Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
			if (lastAttr.tokenPos == LeadingStart || lastAttr.tokenPos == LeadingEnd) && !lastAttr.trailingNewline {
				instances = append(instances, &ViolationInstance{ViolationType: violationMetaBlockNewline, Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		// Check for trailing newlines where required
		default:
			if lastAttr.tokenPos == ProviderEnd && !lastAttr.trailingNewline {
				instances = append(instances, &ViolationInstance{ViolationType: violationProviderNewline, Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
			if (lastAttr.tokenPos == LeadingStart || lastAttr.tokenPos == LeadingEnd) && !lastAttr.trailingNewline {
				instances = append(instances, &ViolationInstance{ViolationType: violationMetaBlockNewline, Path: token.token.Range.Filename, Line: token.token.Range.Start.Line})
			}
		}

		lastAttr = token
	}

	return instances
}
