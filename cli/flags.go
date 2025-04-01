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

//nolint:wrapcheck // These functions intentionally just wrap flag.Flag.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/kr/text"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/timeutil"
)

const maxLineLength = 80

// var unescapedCommas = regexp.MustCompile(`(?m)(?<!\\),`)
var unescapedCommas = regexp.MustCompile(`[^\\](,)`)

type (
	// LookupEnvFunc is the signature of a function for looking up environment
	// variables. It makes that of [os.LookupEnv].
	LookupEnvFunc = func(string) (string, bool)

	// WorkingDirFunc is the signature of a function for getting the current
	// working directory.
	WorkingDirFunc = func() (string, error)

	// PromptAllFunc is the signature for a function that prompts or reads from a
	// pipe.
	PromptAllFunc = func(ctx context.Context, msg string, args ...any) (string, error)
)

// MapLookuper returns a LookupEnvFunc that reads from a map instead of the
// environment. This is mostly used for testing.
func MapLookuper(m map[string]string) LookupEnvFunc {
	return func(s string) (string, bool) {
		if m == nil {
			return "", false
		}

		v, ok := m[s]
		return v, ok
	}
}

// MultiLookuper accepts multiple [LookupEnvFunc]. It runs them in order on the
// environment key, returning the first entry that reports found.
func MultiLookuper(fns ...LookupEnvFunc) LookupEnvFunc {
	return func(s string) (string, bool) {
		for _, fn := range fns {
			if fn == nil {
				continue
			}

			v, ok := fn(s)
			if ok {
				return v, ok
			}
		}

		return "", false
	}
}

// AfterParseFunc is the type signature for functions that are called after
// flags have been parsed.
type AfterParseFunc func(existingErr error) error

// FlagSet is the root flag set for creating and managing flag sections.
type FlagSet struct {
	flagSet  *flag.FlagSet
	sections []*FlagSection

	lookupEnv  LookupEnvFunc
	workingDir WorkingDirFunc
	promptAll  PromptAllFunc

	afterParseFuncs []AfterParseFunc
	args            []string
}

// Option is an option to the flagset.
type Option func(fs *FlagSet) *FlagSet

// WithLookupEnv defines a custom function for looking up environment variables.
// This is mostly useful for testing.
//
// To bind to a CLI's lookup function:
//
//	func (c *CountCommand) Flags() *cli.FlagSet {
//		set := cli.NewFlagSet(cli.WithLookupEnv(c.LookupEnv))
//	}
//
// Alternatively use [BaseCommand.NewFlagSet].
func WithLookupEnv(fn LookupEnvFunc) Option {
	return func(fs *FlagSet) *FlagSet {
		if fn != nil {
			fs.lookupEnv = fn
		}
		return fs
	}
}

// WithWorkingDir sets the working directory function.
func WithWorkingDir(fn WorkingDirFunc) Option {
	return func(fs *FlagSet) *FlagSet {
		if fn != nil {
			fs.workingDir = fn
		}
		return fs
	}
}

// WithWorkingDir sets the prompt function.
func WithPromptAll(fn PromptAllFunc) Option {
	return func(fs *FlagSet) *FlagSet {
		if fn != nil {
			fs.promptAll = fn
		}
		return fs
	}
}

// NewFlagSet creates a new root flag set.
func NewFlagSet(opts ...Option) *FlagSet {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	// Errors and usage are controlled by the writer.
	f.Usage = func() {}
	f.SetOutput(io.Discard)

	fs := &FlagSet{
		flagSet:    f,
		lookupEnv:  os.LookupEnv,
		workingDir: workingDir,
		promptAll: func(ctx context.Context, msg string, args ...any) (string, error) {
			var val string
			fmt.Printf(msg, args...)
			_, err := fmt.Scanf("%s", &val)
			return val, err
		},
	}

	for _, opt := range opts {
		fs = opt(fs)
	}

	return fs
}

// FlagSection represents a group section of flags. The flags are actually
// "flat" in memory, but maintain a structure for better help output and alias
// matching.
type FlagSection struct {
	name      string
	flagNames []string

	// fields inherited from the parent
	flagSet *flag.FlagSet

	lookupEnv  LookupEnvFunc
	workingDir WorkingDirFunc
	promptAll  PromptAllFunc
}

// NewSection creates a new flag section. By convention, section names should be
// all capital letters (e.g. "MY SECTION"), but this is not strictly enforced.
func (f *FlagSet) NewSection(name string) *FlagSection {
	fs := &FlagSection{
		name:       name,
		flagSet:    f.flagSet,
		lookupEnv:  f.lookupEnv,
		workingDir: f.workingDir,
		promptAll:  f.promptAll,
	}
	f.sections = append(f.sections, fs)
	return fs
}

// AfterParse defines a post-parse function. This can be used to set flag
// defaults that should not be set until after parsing, such as defaulting flag
// values to the value of other flags. These functions are called after flags
// have been parsed by the flag library, but before [Parse] returns.
func (f *FlagSet) AfterParse(fn AfterParseFunc) {
	if fn == nil {
		return
	}

	f.afterParseFuncs = append(f.afterParseFuncs, fn)
}

// Arg implements flag.FlagSet#Arg.
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// Args implements flag.FlagSet#Args.
func (f *FlagSet) Args() []string {
	cp := append([]string{}, f.args...)
	return cp
}

// Lookup implements flag.FlagSet#Lookup.
func (f *FlagSet) Lookup(name string) *flag.Flag {
	return f.flagSet.Lookup(name)
}

// Args implements flag.FlagSet#Parse.
func (f *FlagSet) Parse(args []string) error {
	// Call the normal parse function first, so that Args and everything are
	// properly set for any after functions.
	merr := f.flagSet.Parse(args)

	// "Recursively" parse flags. By default, Go stops parsing after the first
	// non-flag argument.
	var finalArgs []string
	for i := len(args) - len(f.flagSet.Args()) + 1; i < len(args); {
		// Stop parsing if we hit an actual "stop parsing"
		if i > 1 && args[i-2] == "--" {
			break
		}
		finalArgs = append(finalArgs, f.flagSet.Arg(0))
		merr = errors.Join(merr, f.flagSet.Parse(args[i:]))
		i += 1 + len(args[i:]) - len(f.flagSet.Args())
	}
	finalArgs = append(finalArgs, f.flagSet.Args()...)

	f.args = finalArgs

	for _, fn := range f.afterParseFuncs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					merr = errors.Join(merr, fmt.Errorf("panic: %v", r))
				}
			}()

			merr = errors.Join(merr, fn(merr))
		}()
	}

	return merr
}

// Args implements flag.FlagSet#Parsed.
func (f *FlagSet) Parsed() bool {
	return f.flagSet.Parsed()
}

// Args implements flag.FlagSet#Visit.
func (f *FlagSet) Visit(fn func(*flag.Flag)) {
	f.flagSet.Visit(fn)
}

// Args implements flag.FlagSet#VisitAll.
func (f *FlagSet) VisitAll(fn func(*flag.Flag)) {
	f.flagSet.VisitAll(fn)
}

// Help returns formatted help output.
func (f *FlagSet) Help() string {
	var b strings.Builder

	for _, set := range f.sections {
		sort.Strings(set.flagNames)

		fmt.Fprint(&b, set.name)
		fmt.Fprint(&b, "\n\n")

		for _, name := range set.flagNames {
			sub := set.flagSet.Lookup(name)
			if sub == nil {
				panic("inconsistency between flag structure and help")
			}

			typ, ok := sub.Value.(Value)
			if !ok {
				panic(fmt.Sprintf("flag is incorrect type %T", sub.Value))
			}

			// Do not process hidden flags.
			if typ.Hidden() {
				continue
			}

			// Incorporate aliases.
			aliases := typ.Aliases()
			sort.Slice(aliases, func(i, j int) bool {
				return len(aliases[i]) < len(aliases[j])
			})
			all := make([]string, 0, len(aliases)+1)
			for _, v := range aliases {
				all = append(all, "-"+v)
			}
			all = append(all, "-"+sub.Name)

			// Handle boolean flags
			if typ.IsBoolFlag() {
				fmt.Fprintf(&b, "    %s\n", strings.Join(all, ", "))
			} else {
				fmt.Fprintf(&b, "    %s=%q\n", strings.Join(all, ", "), typ.Example())
			}

			indented := wrapAtLengthWithPadding(sub.Usage, 8)
			fmt.Fprint(&b, indented)
			fmt.Fprint(&b, "\n\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// GetEnv is a convenience function for looking up an environment variable. By
// default, it is the same as [os.GetEnv], but the lookup function can be
// overridden.
func (f *FlagSet) GetEnv(k string) string {
	v, _ := f.LookupEnv(k)
	return v
}

// LookupEnv is a convenience function for looking up an environment variable.
// By default, it is the same as [os.LookupEnv], but the lookup function can be
// overridden.
func (f *FlagSet) LookupEnv(k string) (string, bool) {
	return f.lookupEnv(k)
}

// Value is an extension of [flag.Value] which adds additional fields for
// setting examples and defining aliases. All flags with this package must
// statisfy this interface.
type Value interface {
	flag.Value

	// Get returns the value. Even though we know the concrete type with generics,
	// this returns [any] to match the standard library.
	Get() any

	// Aliases returns any defined aliases of the flag.
	Aliases() []string

	// Example returns an example input for the flag. For example, if the flag was
	// accepting a URL, this could be "https://example.com". This is largely meant
	// as a hint to the CLI user and only affects help output.
	//
	// If there is a default value, the example value should be different from the
	// default value.
	Example() string

	// Hidden returns true if the flag is hidden, false otherwise.
	Hidden() bool

	// IsBoolFlag returns true if the flag accepts no arguments, false otherwise.
	IsBoolFlag() bool

	// Predictor returns a completion predictor. All flags have a default
	// predictor, but they can be further customized by the developer when
	// instantiating the flag.
	Predictor() complete.Predictor
}

// ParserFunc is a function that parses a value into T, or returns an error.
type ParserFunc[T any] func(val string) (T, error)

// PrinterFunc is a function that pretty-prints T.
type PrinterFunc[T any] func(cur T) string

// SetterFunc is a function that sets *T to T.
type SetterFunc[T any] func(cur *T, val T)

type Var[T any] struct {
	Name    string
	Aliases []string
	Usage   string
	Example string
	Default T
	Hidden  bool
	IsBool  bool
	EnvVar  string
	Target  *T

	// AllowFromFile allows the flag contents to be read from a file by starting
	// the input value with an "@" sign.
	AllowFromFile bool

	// AllowFromPrompt allows the flag contents to be read from a prompt or a pipe
	// like stdin by setting the input value to "-".
	AllowFromPrompt bool

	// Parser and Printer are the generic functions for converting string values
	// to/from the target value. These are populated by the individual flag
	// helpers.
	Parser  ParserFunc[T]
	Printer PrinterFunc[T]

	// Predict is the completion predictor. If no predictor is defined, it
	// defaults to predicting something (waiting for a value) for all flags except
	// boolean flags (which have no value). Callers are encouraged to customize
	// the predictions.
	Predict complete.Predictor

	// Setter defines the function that sets the variable into the target. If nil,
	// it uses a default setter which overwrites the entire value of the Target.
	// Implementations that do special processing (such as appending to a slice),
	// may override this to customize the behavior.
	Setter SetterFunc[T]
}

// Flag is a lower-level API for creating a flag on a flag section. Callers
// should use this for defining new flags as it sets defaults and provides more
// granular usage details.
//
// It panics if any of the target, parser, or printer are nil.
func Flag[T any](f *FlagSection, i *Var[T]) {
	if i.Target == nil {
		panic("missing target")
	}

	parser := i.Parser
	if parser == nil {
		panic("missing parser func")
	}

	printer := i.Printer
	if printer == nil {
		panic("missing printer func")
	}

	predictor := i.Predict
	if predictor == nil {
		if i.IsBool {
			predictor = predict.Nothing
		} else {
			predictor = predict.Something
		}
	}

	usage := i.Usage

	if i.AllowFromFile {
		parser = fromFileParser(i.Name, parser, f.workingDir)
		usage += " This can be read from a file on disk by setting the value " +
			"to \"@\" followed by the filepath."
	}

	if i.AllowFromPrompt {
		parser = fromPromptParser(i.Name, parser, f.promptAll)
		usage += " This value be read from a prompt or pipe by setting the value " +
			"to \"-\"."
	}

	setter := i.Setter
	if setter == nil {
		setter = func(cur *T, val T) { *cur = val }
	}

	initial := i.Default
	if v, ok := f.lookupEnv(i.EnvVar); ok {
		if t, err := parser(v); err == nil {
			initial = t
		}
	}

	// Set a default value.
	setter(i.Target, initial)

	// Compute a sane default if one was not given.
	example := i.Example
	if example == "" {
		example = fmt.Sprintf("%T", *new(T))
	}

	if v := printer(i.Default); v != "" {
		usage += fmt.Sprintf(" The default value is %q.", v)
	}

	if v := i.EnvVar; v != "" {
		usage += fmt.Sprintf(" This option can also be specified with the %s "+
			"environment variable.", v)
	}

	fv := &flagValue[T]{
		target:    i.Target,
		hidden:    i.Hidden,
		isBool:    i.IsBool,
		example:   example,
		parser:    parser,
		printer:   printer,
		predictor: predictor,
		setter:    setter,
		aliases:   i.Aliases,
	}
	f.flagNames = append(f.flagNames, i.Name)
	f.flagSet.Var(fv, i.Name, usage)

	// Since aliases are not added as a flag name, we can safely add them to the
	// main flag set. Our custom help will skip them.
	for _, alias := range i.Aliases {
		f.flagSet.Var(fv, alias, "")
	}
}

var _ Value = (*flagValue[any])(nil)

type flagValue[T any] struct {
	target  *T
	hidden  bool
	isBool  bool
	example string

	parser    ParserFunc[T]
	printer   PrinterFunc[T]
	setter    SetterFunc[T]
	predictor complete.Predictor
	aliases   []string
}

func (f *flagValue[T]) Set(s string) error {
	v, err := f.parser(s)
	if err != nil {
		return err
	}
	f.setter(f.target, v)
	return nil
}

func (f *flagValue[T]) Get() any                      { return *f.target }
func (f *flagValue[T]) Aliases() []string             { return f.aliases }
func (f *flagValue[T]) String() string                { return f.printer(*f.target) }
func (f *flagValue[T]) Example() string               { return f.example }
func (f *flagValue[T]) Hidden() bool                  { return f.hidden }
func (f *flagValue[T]) IsBoolFlag() bool              { return f.isBool }
func (f *flagValue[T]) Predictor() complete.Predictor { return f.predictor }

type BoolVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         bool
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *bool
	AllowFromFile   bool
	AllowFromPrompt bool
}

// BoolVar creates a new boolean variable (true/false). By convention, the
// default value should always be false. For example:
//
//	Bad: -enable-cookies (default: true)
//	Good: -disable-cookies (default: false)
//
// Consider naming your flags to match this convention.
func (f *FlagSection) BoolVar(i *BoolVar) {
	Flag(f, &Var[bool]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		IsBool:          true,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          strconv.ParseBool,
		Printer:         strconv.FormatBool,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type DurationVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         time.Duration
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *time.Duration
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) DurationVar(i *DurationVar) {
	Flag(f, &Var[time.Duration]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          time.ParseDuration,
		Printer:         timeutil.HumanDuration,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type Float64Var struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         float64
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *float64
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) Float64Var(i *Float64Var) {
	parser := func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}
	printer := func(v float64) string {
		return strconv.FormatFloat(v, 'e', -1, 64)
	}

	Flag(f, &Var[float64]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type IntVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         int
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *int
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) IntVar(i *IntVar) {
	parser := func(s string) (int, error) {
		v, err := strconv.ParseInt(s, 10, strconv.IntSize)
		return int(v), err
	}
	printer := func(v int) string { return strconv.FormatInt(int64(v), 10) }

	Flag(f, &Var[int]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type Int64Var struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         int64
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *int64
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) Int64Var(i *Int64Var) {
	parser := func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }
	printer := func(v int64) string { return strconv.FormatInt(v, 10) }

	Flag(f, &Var[int64]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type StringVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         string
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *string
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) StringVar(i *StringVar) {
	parser := func(s string) (string, error) { return s, nil }
	printer := func(v string) string { return v }

	Flag(f, &Var[string]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type StringMapVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         map[string]string
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *map[string]string
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) StringMapVar(i *StringMapVar) {
	parser := func(s string) (map[string]string, error) {
		idx := strings.Index(s, "=")
		if idx == -1 {
			return nil, fmt.Errorf("missing = in KV pair %q", s)
		}

		m := make(map[string]string, 1)
		m[s[0:idx]] = s[idx+1:]
		return m, nil
	}

	printer := func(m map[string]string) string {
		list := make([]string, 0, len(m))
		for k, v := range m {
			list = append(list, k+"="+v)
		}
		sort.Strings(list)
		return strings.Join(list, ",")
	}

	var setDefault *bool
	setter := func(cur *map[string]string, val map[string]string) {
		if setDefault == nil {
			setDefault = ptr(true)
		} else if *setDefault {
			*cur = make(map[string]string)
			setDefault = ptr(false)
		}

		if *cur == nil {
			*cur = make(map[string]string)
		}
		for k, v := range val {
			(*cur)[k] = v
		}
	}

	Flag(f, &Var[map[string]string]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		Setter:          setter,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type StringSliceVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         []string
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *[]string
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) StringSliceVar(i *StringSliceVar) {
	parser := func(s string) ([]string, error) {
		final := make([]string, 0)
		indices := unescapedCommas.FindAllStringSubmatchIndex(s, -1)
		lastMatch := 0
		for _, indexPair := range indices {
			part := s[lastMatch : indexPair[0]+1]
			parsed := strings.TrimSpace(escapeComma(part))
			if parsed != "" {
				final = append(final, parsed)
			}
			lastMatch = indexPair[1]
		}
		remainder := s[lastMatch:len(s)]
		final = append(final, strings.TrimSpace(escapeComma(remainder)))
		return final, nil
	}

	printer := func(v []string) string {
		return strings.Join(v, ",")
	}

	var setDefault *bool
	setter := func(cur *[]string, val []string) {
		if setDefault == nil {
			setDefault = ptr(true)
		} else if *setDefault {
			*cur = []string{}
			setDefault = ptr(false)
		}

		*cur = append(*cur, val...)
	}

	Flag(f, &Var[[]string]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		Setter:          setter,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

func escapeComma(v string) string {
	return strings.ReplaceAll(v, `\,`, ",")
}

type TimeVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         time.Time
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *time.Time
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) TimeVar(layout string, i *TimeVar) {
	parser := func(s string) (time.Time, error) {
		return time.Parse(layout, s)
	}
	printer := func(v time.Time) string {
		return v.Format(layout)
	}

	Flag(f, &Var[time.Time]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type UintVar struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         uint
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *uint
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) UintVar(i *UintVar) {
	parser := func(s string) (uint, error) {
		v, err := strconv.ParseUint(s, 10, strconv.IntSize)
		return uint(v), err
	}
	printer := func(v uint) string { return strconv.FormatUint(uint64(v), 10) }

	Flag(f, &Var[uint]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type Uint64Var struct {
	Name            string
	Aliases         []string
	Usage           string
	Example         string
	Default         uint64
	Hidden          bool
	EnvVar          string
	Predict         complete.Predictor
	Target          *uint64
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) Uint64Var(i *Uint64Var) {
	parser := func(s string) (uint64, error) { return strconv.ParseUint(s, 10, 64) }
	printer := func(v uint64) string { return strconv.FormatUint(v, 10) }

	Flag(f, &Var[uint64]{
		Name:            i.Name,
		Aliases:         i.Aliases,
		Usage:           i.Usage,
		Example:         i.Example,
		Default:         i.Default,
		Hidden:          i.Hidden,
		EnvVar:          i.EnvVar,
		Predict:         i.Predict,
		Target:          i.Target,
		Parser:          parser,
		Printer:         printer,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

type LogLevelVar struct {
	Logger          *slog.Logger
	AllowFromFile   bool
	AllowFromPrompt bool
}

func (f *FlagSection) LogLevelVar(i *LogLevelVar) {
	parser := func(s string) (slog.Level, error) {
		v, err := logging.LookupLevel(s)
		if err != nil {
			return 0, err
		}
		return v, nil
	}

	printer := func(v slog.Level) string { return logging.LevelString(v) }

	setter := func(_ *slog.Level, val slog.Level) { logging.SetLevel(i.Logger, val) }

	// trick the CLI into thinking we need a value to set.
	var fake slog.Level

	levelNames := logging.LevelNames()

	Flag(f, &Var[slog.Level]{
		Name:    "log-level",
		Aliases: []string{"l"},
		Usage: `Sets the logging verbosity. Valid values include: ` +
			strings.Join(levelNames, ",") + `.`,
		Example:         "warn",
		Default:         slog.LevelInfo,
		Predict:         predict.Set(levelNames),
		Target:          &fake,
		Parser:          parser,
		Printer:         printer,
		Setter:          setter,
		AllowFromFile:   i.AllowFromFile,
		AllowFromPrompt: i.AllowFromPrompt,
	})
}

// wrapAtLengthWithPadding wraps the given text at the maxLineLength, taking
// into account any provided left padding.
func wrapAtLengthWithPadding(s string, pad int) string {
	wrapped := text.Wrap(s, maxLineLength-pad)
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", pad) + line
	}
	return strings.Join(lines, "\n")
}

func ptr[T any](i T) *T {
	return &i
}

// fromFileParser wraps an existing parser function and adds the ability for the
// input to be sourced from a file using the "@" prefix syntax.
func fromFileParser[T any](name string, inner ParserFunc[T], workingDir WorkingDirFunc) ParserFunc[T] {
	return func(val string) (T, error) {
		if len(val) > 0 && val[0] == '@' {
			root, err := workingDir()
			if err != nil {
				var nilT T
				return nilT, err
			}

			pth := filepath.Clean(val[1:])
			if !filepath.IsAbs(pth) {
				pth = filepath.Clean(filepath.Join(root, pth))
			}

			b, err := os.ReadFile(pth)
			if err != nil {
				var nilT T
				return nilT, fmt.Errorf("failed to set %q: failed to read %s: %w", name, pth, err)
			}

			val = strings.TrimRightFunc(string(b), unicode.IsSpace)
		} else if len(val) > 1 && val[0] == '\\' && val[1] == '@' {
			// Unescape \@foo to "@foo"
			val = val[1:]
		}

		return inner(val)
	}
}

// fromPromptParser wraps an existing parser function and adds the ability for the
// input to be sourced from a prompt or pipe using the "-" value.
func fromPromptParser[T any](name string, inner ParserFunc[T], promptAll PromptAllFunc) ParserFunc[T] {
	return func(val string) (T, error) {
		if len(val) == 1 && val[0] == '-' {
			ctx, done := signal.NotifyContext(context.Background(),
				syscall.SIGINT, syscall.SIGTERM)
			defer done()

			s, err := promptAll(ctx, "Provide a value for %q: ", name)
			if err != nil {
				var nilT T
				return nilT, fmt.Errorf("failed to set %q: %w", name, err)
			}
			val = strings.TrimRightFunc(s, unicode.IsSpace)
		} else if len(val) == 2 && val[0] == '\\' && val[1] == '-' {
			// Unescape \- to "-" to support the literal value "-".
			val = val[1:]
		}

		return inner(val)
	}
}
