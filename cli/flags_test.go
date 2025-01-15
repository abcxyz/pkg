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
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/pointer"
	"github.com/abcxyz/pkg/testutil"
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
		Target: pointer.To(true),
	})
	sec1.Int64Var(&Int64Var{
		Name:   "my-int",
		Usage:  "One usage.",
		Hidden: true,
		Target: pointer.To(int64(0)),
	})

	sec2 := fs.NewSection("child2")
	sec2.StringVar(&StringVar{
		Name:    "two",
		Usage:   "Two usage.",
		Aliases: []string{"t", "at"},
		Example: "example",
		Target:  pointer.To(""),
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

func TestFlagSection_StringMapVar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		def  map[string]string
		exp  map[string]string
	}{
		{
			name: "empty",
			args: []string{},
			def:  nil,
			exp:  map[string]string{},
		},
		{
			name: "default",
			args: []string{},
			def: map[string]string{
				"one": "hello",
			},
			exp: map[string]string{
				"one": "hello",
			},
		},
		{
			name: "overrides_default_single",
			args: []string{"-test", "a=b"},
			def: map[string]string{
				"one": "hello",
			},
			exp: map[string]string{
				"a": "b",
			},
		},
		{
			name: "overrides_default_many",
			args: []string{"-test", "a=b"},
			def: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
			exp: map[string]string{
				"a": "b",
			},
		},
		{
			name: "overrides_default_many_many",
			args: []string{"-test", "a=b", "-test", "c=d"},
			def: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
			exp: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "given_default_value_one",
			args: []string{"-test", "foo=bar"},
			def: map[string]string{
				"foo": "bar",
			},
			exp: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "given_default_value_one_and_more",
			args: []string{"-test", "foo=bar", "-test", "zip=zap"},
			def: map[string]string{
				"foo": "bar",
			},
			exp: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
		},
		{
			name: "given_default_value_many",
			args: []string{"-test", "foo=bar", "-test", "zip=zap"},
			def: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
			exp: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
		},
		{
			name: "given_default_value_many_and_more",
			args: []string{"-test", "foo=bar", "-test", "a=b"},
			def: map[string]string{
				"foo": "bar",
				"zip": "zap",
			},
			exp: map[string]string{
				"foo": "bar",
				"a":   "b",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target := make(map[string]string)

			fs := NewFlagSet()
			s := fs.NewSection("")
			s.StringMapVar(&StringMapVar{
				Name:    "test",
				Default: tc.def,
				Target:  &target,
			})

			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, target); diff != "" {
				t.Errorf("diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestFlagSection_StringSliceVar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		def  []string
		exp  []string
	}{
		{
			name: "empty",
			args: []string{},
			def:  nil,
			exp:  []string{},
		},
		{
			name: "default",
			args: []string{},
			def:  []string{"one"},
			exp:  []string{"one"},
		},
		{
			name: "overrides_default_single",
			args: []string{"-test", "a"},
			def:  []string{"one"},
			exp:  []string{"a"},
		},
		{
			name: "overrides_default_many",
			args: []string{"-test", "a"},
			def:  []string{"one", "two"},
			exp:  []string{"a"},
		},
		{
			name: "overrides_default_many_many",
			args: []string{"-test", "a, b, c,d"},
			def:  []string{"one", "two"},
			exp:  []string{"a", "b", "c", "d"},
		},
		{
			name: "given_default_value_one",
			args: []string{"-test", "a"},
			def:  []string{"a"},
			exp:  []string{"a"},
		},
		{
			name: "given_default_value_one_and_more",
			args: []string{"-test", "a", "-test", "b"},
			def:  []string{"a"},
			exp:  []string{"a", "b"},
		},
		{
			name: "given_default_value_many",
			args: []string{"-test", "a", "-test", "b"},
			def:  []string{"a", "b"},
			exp:  []string{"a", "b"},
		},
		{
			name: "given_default_value_many_and_more",
			args: []string{"-test", "a,b", "-test", "c"},
			def:  []string{"a", "b"},
			exp:  []string{"a", "b", "c"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target := make([]string, 0, 8)

			fs := NewFlagSet()
			s := fs.NewSection("")
			s.StringSliceVar(&StringSliceVar{
				Name:    "test",
				Default: tc.def,
				Target:  &target,
			})

			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, target); diff != "" {
				t.Errorf("diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestFlagSet_AfterParse(t *testing.T) {
	t.Parallel()

	t.Run("recovers_panic", func(t *testing.T) {
		t.Parallel()

		fs := NewFlagSet()
		fs.AfterParse(func(existingErr error) error {
			panic("oh no!")
		})

		// This implicitly checks we did not panic
		err := fs.Parse(nil)
		if diff := testutil.DiffErrString(err, "panic: oh no!"); diff != "" {
			t.Error(diff)
		}
	})

	t.Run("runs_all", func(t *testing.T) {
		t.Parallel()

		var names []string

		fs := NewFlagSet()
		fs.AfterParse(func(existingErr error) error {
			names = append(names, "one")
			return nil
		})
		fs.AfterParse(func(existingErr error) error {
			names = append(names, "two")
			return nil
		})

		if err := fs.Parse(nil); err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff([]string{"one", "two"}, names); diff != "" {
			t.Errorf("did not run all functions (-want, +got):\n%s", diff)
		}
	})

	t.Run("runs_all_error", func(t *testing.T) {
		t.Parallel()

		fs := NewFlagSet()
		fs.AfterParse(func(existingErr error) error {
			return fmt.Errorf("one")
		})
		fs.AfterParse(func(existingErr error) error {
			return fmt.Errorf("two")
		})

		err := fs.Parse(nil)
		if diff := testutil.DiffErrString(err, "one\ntwo"); diff != "" {
			t.Error(diff)
		}
	})
}

func ExampleFlagSet_AfterParse_validation() {
	set := NewFlagSet()
	f := set.NewSection("FLAGS")

	var address string
	f.StringVar(&StringVar{
		Name:   "address",
		Target: &address,
	})

	var protocol string
	f.StringVar(&StringVar{
		Name:   "protocol",
		Target: &protocol,
	})

	set.AfterParse(func(existingErr error) error {
		var merr error
		if address == "" {
			return fmt.Errorf("-address is required")
		}
		if address == "" {
			return fmt.Errorf("-protocol is required")
		}
		return merr
	})
}

func ExampleFlagSet_AfterParse_deferredDefault() {
	set := NewFlagSet()
	f := set.NewSection("FLAGS")

	// This is an old flag that we will remove in the future. We want "-address"
	// to default to the value of this flag. However, the value of this flag is
	// not known until after parsing, so we can't set `Default` on the address
	// flag to this flag, since that's resolved at compile time. Instead, we need
	// to use the `AfterParse` function to set the defaults.
	var host string
	f.StringVar(&StringVar{
		Name:   "host",
		Target: &host,
		Hidden: true,
	})

	var address string
	f.StringVar(&StringVar{
		Name:   "address",
		Target: &address,
	})

	set.AfterParse(func(existingErr error) error {
		if address == "" {
			address = host
		}
		return nil
	})
}

func ExampleFlagSet_AfterParse_deferredDefaultArgs() {
	set := NewFlagSet()
	f := set.NewSection("FLAGS")

	// The default value should be the first argument. Setting this to default to
	// `os.Args[1]` will not work, because arguments can shift after flag parsing.
	// Instead, we need to use the `AfterParse` function to set the default.
	var address string
	f.StringVar(&StringVar{
		Name:   "address",
		Target: &address,
	})

	set.AfterParse(func(existingErr error) error {
		if address == "" {
			address = set.Arg(1)
		}
		return nil
	})
}

func ExampleFlagSet_AfterParse_checkIfError() {
	set := NewFlagSet()

	set.AfterParse(func(existingErr error) error {
		// Do not run this function if flag parsing or other AfterParse functions
		// have failed.
		if existingErr != nil {
			return nil //nolint:nilerr
		}

		// Logic
		return nil
	})
}

func TestLogLevelVar(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name string
		args []string

		wantLevel slog.Level
		wantError string
	}{
		{
			name:      "empty",
			args:      nil,
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "long",
			args:      []string{"-log-level", "debug"},
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "short",
			args:      []string{"-l", "debug"},
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "invalid",
			args:      []string{"-log-level", "pants"},
			wantError: `invalid value "pants" for flag -log-level`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := logging.New(io.Discard, slog.LevelInfo, logging.FormatText, false)

			set := NewFlagSet()
			f := set.NewSection("FLAGS")

			f.LogLevelVar(&LogLevelVar{
				Logger: logger,
			})

			err := set.Parse(tc.args)
			if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
				t.Error(diff)
			}

			if !logger.Handler().Enabled(ctx, tc.wantLevel) {
				t.Errorf("expected handler to be at least %s", tc.wantLevel)
			}
		})
	}
}

func TestFlagSet_Parse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string

		wantArgs   []string
		wantFruits []string
		wantError  string
	}{
		{
			name:       "before_one",
			args:       []string{"{{dash}}fruit", "apple", "arg1"},
			wantArgs:   []string{"arg1"},
			wantFruits: []string{"apple"},
		},
		{
			name:       "before_two",
			args:       []string{"{{dash}}fruit=apple", "{{dash}}fruit", "banana", "arg1"},
			wantArgs:   []string{"arg1"},
			wantFruits: []string{"apple", "banana"},
		},
		{
			name:       "after_one",
			args:       []string{"arg1", "{{dash}}fruit", "apple"},
			wantArgs:   []string{"arg1"},
			wantFruits: []string{"apple"},
		},
		{
			name:       "after_two",
			args:       []string{"arg1", "{{dash}}fruit=apple", "{{dash}}fruit", "banana"},
			wantArgs:   []string{"arg1"},
			wantFruits: []string{"apple", "banana"},
		},
		{
			name:       "sandwich_arg",
			args:       []string{"{{dash}}fruit=apple", "arg1", "{{dash}}fruit", "banana"},
			wantArgs:   []string{"arg1"},
			wantFruits: []string{"apple", "banana"},
		},
		{
			name:       "zipper",
			args:       []string{"{{dash}}fruit=apple", "arg1", "{{dash}}fruit", "banana", "arg2"},
			wantArgs:   []string{"arg1", "arg2"},
			wantFruits: []string{"apple", "banana"},
		},
		{
			name:       "singleton_double_dash",
			args:       []string{"{{dash}}fruit=apple", "arg1", "--", "-fruit", "banana", "--fruit=pineapple"},
			wantFruits: []string{"apple"},
			wantArgs:   []string{"arg1", "-fruit", "banana", "--fruit=pineapple"},
		},
		{
			name:      "error",
			args:      []string{"{{dash}}invalid", "arg1"},
			wantArgs:  []string{"arg1"},
			wantError: "flag provided but not defined: -invalid",
		},
	}

	for _, tc := range cases {
		for dashName, dash := range map[string]string{
			"single_dash": "-",
			"double_dash": "--",
		} {
			dash, dashName := dash, dashName

			t.Run(dashName+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				fs := NewFlagSet()
				sec := fs.NewSection("my-section")

				var fruits []string
				sec.StringSliceVar(&StringSliceVar{
					Name:   "fruit",
					Usage:  "Pick a fruit.",
					Target: &fruits,
				})

				args := make([]string, 0, len(tc.args))
				for _, v := range tc.args {
					args = append(args, strings.ReplaceAll(v, "{{dash}}", dash))
				}

				err := fs.Parse(args)
				if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
					t.Error(diff)
				}

				if got, want := fs.Args(), tc.wantArgs; !reflect.DeepEqual(got, want) {
					t.Errorf("args: expected %q to be %q", got, want)
				}
				if got, want := fruits, tc.wantFruits; !reflect.DeepEqual(got, want) {
					t.Errorf("fruits: expected %q to be %q", got, want)
				}
			})
		}
	}
}

func TestFromFileParser(t *testing.T) {
	t.Parallel()

	goodFilePathAbs := filepath.Join(t.TempDir(), "good")
	if err := os.WriteFile(goodFilePathAbs, []byte("good\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	goodFilePathRel, err := filepath.Rel(t.TempDir(), goodFilePathAbs)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		input      string
		workingDir WorkingDirFunc
		want       string
		wantError  string
	}{
		{
			name: "working_dir_error_empty_input",
			workingDir: func() (string, error) {
				return "", fmt.Errorf("should not error")
			},
			want: "",
		},
		{
			name:  "working_dir_error_prefix_input",
			input: "@/foo/bar",
			workingDir: func() (string, error) {
				return "", fmt.Errorf("working dir error")
			},
			wantError: "working dir error",
		},
		{
			name:  "prefix_escaped",
			input: "\\@foo",
			want:  "@foo",
		},
		{
			name:  "normal_input_does_not_call_working_dir",
			input: "foo",
			workingDir: func() (string, error) {
				return "", fmt.Errorf("should not error")
			},
			want: "foo",
		},
		{
			name:  "relative_path",
			input: "@" + goodFilePathRel,
			workingDir: func() (string, error) {
				return t.TempDir(), nil
			},
			want: "good",
		},
		{
			name:  "absolute_path",
			input: "@" + goodFilePathAbs,
			want:  "good",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identityParser := func(val string) (string, error) { return val, nil }

			workingDir := func() (string, error) { return t.TempDir(), nil }
			if tc.workingDir != nil {
				workingDir = tc.workingDir
			}

			got, err := fromFileParser("flag", identityParser, workingDir)(tc.input)
			if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
				t.Error(diff)
			}

			if got, want := got, tc.want; got != want {
				t.Errorf("args: expected %q to be %q", got, want)
			}
		})
	}
}

func TestFromPromptParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		input        string
		promptResult string
		promptError  error
		want         string
		wantError    string
	}{
		{
			name:        "no_prompt_on_empty_input",
			input:       "",
			promptError: fmt.Errorf("impossible"),
		},
		{
			name:        "no_prompt_on_extra_characters",
			input:       "--foo",
			promptError: fmt.Errorf("impossible"),
			want:        "--foo",
		},
		{
			name:  "prompt_escaped",
			input: "\\-",
			want:  "-",
		},
		{
			name:        "prompt_error",
			input:       "-",
			promptError: fmt.Errorf("something happened"),
			wantError:   "something happened",
		},
		{
			name:         "prompt_result",
			input:        "-",
			promptResult: "hello there\n",
			want:         "hello there",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identityParser := func(val string) (string, error) { return val, nil }

			prompt := func(_ context.Context, msg string, args ...any) (string, error) {
				return tc.promptResult, tc.promptError
			}

			got, err := fromPromptParser("flag", identityParser, prompt)(tc.input)
			if diff := testutil.DiffErrString(err, tc.wantError); diff != "" {
				t.Error(diff)
			}

			if got, want := got, tc.want; got != want {
				t.Errorf("args: expected %q to be %q", got, want)
			}
		})
	}
}
