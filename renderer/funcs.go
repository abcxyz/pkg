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

package renderer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

func builtinFuncs() template.FuncMap {
	return map[string]any{
		"joinStrings":    joinStrings,
		"toSentence":     toSentence,
		"trimSpace":      trimSpace,
		"stringContains": strings.Contains,
		"toLower":        strings.ToLower,
		"toUpper":        strings.ToUpper,
		"toJSON":         json.Marshal,
		"toBase64":       base64.StdEncoding.EncodeToString,
		"toPercent":      toPercent,
		"safeHTML":       safeHTML,
		"checkedIf":      valueIfTruthy("checked"),
		"requiredIf":     valueIfTruthy("required"),
		"selectedIf":     valueIfTruthy("selected"),
		"readonlyIf":     valueIfTruthy("readonly"),
		"disabledIf":     valueIfTruthy("disabled"),
		"invalidIf":      valueIfTruthy("is-invalid"),

		"pathEscape":    url.PathEscape,
		"pathUnescape":  url.PathUnescape,
		"queryEscape":   url.QueryEscape,
		"queryUnescape": url.QueryUnescape,
	}
}

// safeHTML un-escapes known safe html.
func safeHTML(s string) template.HTML {
	return template.HTML(s) //nolint:gosec // Requires users to explicitly invoke
}

// toStringSlice converts the input slice to strings. The values must be
// primitive or implement the fmt.Stringer interface.
func toStringSlice(i any) ([]string, error) {
	if i == nil {
		return nil, nil
	}

	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return nil, fmt.Errorf("value is not a slice: %T", i)
	}

	s := reflect.ValueOf(i)
	for s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	l := make([]string, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		val := s.Index(i)
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		switch t := val.Interface().(type) {
		case fmt.Stringer:
			l = append(l, t.String())
		case error:
			l = append(l, t.Error())
		case string:
			l = append(l, t)
		case int:
			l = append(l, strconv.FormatInt(int64(t), 10))
		case int8:
			l = append(l, strconv.FormatInt(int64(t), 10))
		case int16:
			l = append(l, strconv.FormatInt(int64(t), 10))
		case int32:
			l = append(l, strconv.FormatInt(int64(t), 10))
		case int64:
			l = append(l, strconv.FormatInt(t, 10))
		case uint:
			l = append(l, strconv.FormatUint(uint64(t), 10))
		case uint8:
			l = append(l, strconv.FormatUint(uint64(t), 10))
		case uint16:
			l = append(l, strconv.FormatUint(uint64(t), 10))
		case uint32:
			l = append(l, strconv.FormatUint(uint64(t), 10))
		case uint64:
			l = append(l, strconv.FormatUint(t, 10))
		}
	}

	return l, nil
}

// joinStrings joins a list of strings or string-like things.
func joinStrings(i any, sep string) (string, error) {
	l, err := toStringSlice(i)
	if err != nil {
		return "", nil
	}
	return strings.Join(l, sep), nil
}

// toSentence joins a list of string like things into a human-friendly sentence.
func toSentence(i any, joiner string) (string, error) {
	l, err := toStringSlice(i)
	if err != nil {
		return "", nil
	}

	switch len(l) {
	case 0:
		return "", nil
	case 1:
		return l[0], nil
	case 2:
		return l[0] + " " + joiner + " " + l[1], nil
	default:
		parts, last := l[0:len(l)-1], l[len(l)-1]
		return strings.Join(parts, ", ") + ", " + joiner + " " + last, nil
	}
}

func valueIfTruthy(s string) func(i any) template.HTMLAttr {
	return func(i any) template.HTMLAttr {
		if i == nil {
			return ""
		}

		v := reflect.ValueOf(i)
		if !v.IsValid() {
			return ""
		}

		//nolint:exhaustive
		switch v.Kind() {
		case reflect.Bool:
			if v.Bool() {
				return template.HTMLAttr(s) //nolint:gosec // Booleans are "true" or "false"
			}
		case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
			if v.Len() > 0 {
				return template.HTMLAttr(s) //nolint:gosec // Trusted source
			}
		default:
		}

		return ""
	}
}

// toPercent takes the given float, multiplies by 100, and then appends a
// trailing percent symbol.
func toPercent(f float64) string {
	return fmt.Sprintf("%.2f%%", f*100.0)
}

// trimSpace trims space and "zero-width no-break space".
func trimSpace(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || !unicode.IsPrint(r) || r == '\uFEFF'
	})
}
