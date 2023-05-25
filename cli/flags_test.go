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

package cli

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewFlagSet(t *testing.T) {
	t.Parallel()

	fs := NewFlagSet()

	if got, want := fs.flagSet.ErrorHandling(), flag.ContinueOnError; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
	if got, want := fs.flagSet.Output(), io.Discard; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestFlagSet_NewSection(t *testing.T) {
	t.Parallel()

	fs := NewFlagSet()
	sec := fs.NewSection("child")

	if got, want := sec.name, "child"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
	// object equality check
	if got, want := sec.flagSet, fs.flagSet; got != want {
		t.Errorf("expected %v to be %v", got, want)
	}
	if got, want := fs.sections, []*FlagSection{sec}; !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v to be %v", got, want)
	}
}

func TestFlagSet_Help(t *testing.T) {
	t.Parallel()

	fs := NewFlagSet()

	sec1 := fs.NewSection("child1")
	sec1.BoolVar(&BoolVar{
		Name:   "my-bool",
		Usage:  "One usage.",
		Target: ptrTo(true),
	})
	sec1.Int64Var(&Int64Var{
		Name:   "my-int",
		Usage:  "One usage.",
		Hidden: true,
		Target: ptrTo(int64(0)),
	})

	sec2 := fs.NewSection("child2")
	sec2.StringVar(&StringVar{
		Name:    "two",
		Usage:   "Two usage.",
		Aliases: []string{"t", "at"},
		Example: "example",
		Target:  ptrTo(""),
	})

	if got, want := fs.Help(), "One usage. The default value is"; !strings.Contains(got, want) {
		t.Errorf("expected\n\n%s\n\nto include %q", got, want)
	}
	if got, want := fs.Help(), `-t, -at, -two="example"`; !strings.Contains(got, want) {
		t.Errorf("expected\n\n%s\n\nto include %q", got, want)
	}
	if got, want := fs.Help(), "my-int"; strings.Contains(got, want) {
		t.Errorf("expected\n\n%s\n\nto not include %q", got, want)
	}
}

func TestFlagSet_Default(t *testing.T) {
	t.Parallel()

	t.Run("no_setter", func(t *testing.T) {
		t.Parallel()

		var got []string
		want := []string{"foo", "bar"}
		fs := NewFlagSet()
		sec := fs.NewSection("sec")
		sec.StringSliceVar(&StringSliceVar{
			Name:    "string-slice",
			Usage:   "Give a string slice.",
			Default: want,
			Target:  &got,
		})

		if err := fs.Parse([]string{}); err != nil {
			t.Fatalf("FlagSet.Parse got unexpected err: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("string slice value from default (-want,+got):\n%s", diff)
		}
	})

	t.Run("with_setter", func(t *testing.T) {
		t.Parallel()

		got := []string{"abcxyz"}
		want := []string{"abcxyz", "foo", "bar"}
		fs := NewFlagSet()
		sec := fs.NewSection("sec")

		Flag(sec, &Var[[]string]{
			Name:    "string-slice",
			Usage:   "Give a string slice.",
			Default: []string{"foo", "bar"},
			Target:  &got,
			Parser: func(s string) ([]string, error) {
				return strings.Split(s, ","), nil
			},
			Printer: func(cur []string) string {
				return fmt.Sprint(cur)
			},
			Setter: func(cur *[]string, val []string) {
				// We *append* the default value to the target rather than *assign* so
				// it's different from the default setter logic.
				*cur = append(*cur, val...)
			},
		})

		if err := fs.Parse([]string{}); err != nil {
			t.Fatalf("FlagSet.Parse got unexpected err: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("string slice value from default (-want,+got):\n%s", diff)
		}
	})
}

func ptrTo[T any](v T) *T {
	return &v
}
