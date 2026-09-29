package cli

import (
	"bytes"
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type id int64

func (i *id) String() string { return strconv.FormatInt(int64(*i), 10) }
func (i *id) Set(s string) error {
	if _, after, ok := strings.Cut(s, "#"); ok {
		s = after
	}
	v, err := strconv.ParseInt(s, 10, 64)
	*i = id(v)
	return err
}

type argCmd struct {
	Verbose bool          `flag:"v, verbose"`
	Name    string        `arg:"name, who"`
	Count   int           `arg:"count, how many"`
	Wait    time.Duration `arg:"wait, how long"`
	IDs     []id          `arg:"ids, observation ids"`
	ran     bool
}

func (c *argCmd) Help() string                { return "Usage: test run [-v] <name> <count> <wait> [ids...]" }
func (c *argCmd) Synopsis() string            { return "run" }
func (c *argCmd) Run(_ context.Context) error { c.ran = true; return nil }

type badVariadicCmd struct {
	Files []string `arg:"files"`
	Name  string   `arg:"name"`
}

func (c *badVariadicCmd) Help() string                { return "" }
func (c *badVariadicCmd) Synopsis() string            { return "" }
func (c *badVariadicCmd) Run(_ context.Context) error { return nil }

func run(t *testing.T, c Command, args ...string) (stdout, stderr string) {
	t.Helper()
	SetRoot(&Root{Name: "test", subCommands: map[string]Command{"run": c}})
	t.Cleanup(func() { SetRoot(&Root{subCommands: make(map[string]Command)}) })
	var out, errOut bytes.Buffer
	cli := New("test", "1")
	cli.HelpWriter, cli.ErrorWriter = &out, &errOut
	cli.Run(context.Background(), append([]string{"test", "run"}, args...))
	return out.String(), errOut.String()
}

func TestArgs(t *testing.T) {
	c := &argCmd{}
	_, stderr := run(t, c, "-v", "alice", "3", "2s", "obs#1", "2")
	if stderr != "" {
		t.Fatalf("stderr: %s", stderr)
	}
	want := &argCmd{Verbose: true, Name: "alice", Count: 3, Wait: 2 * time.Second, IDs: []id{1, 2}, ran: true}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func TestArgsVariadicEmpty(t *testing.T) {
	c := &argCmd{}
	if _, stderr := run(t, c, "bob", "1", "0s"); stderr != "" || !c.ran || len(c.IDs) != 0 {
		t.Fatalf("got %+v, stderr %q", c, stderr)
	}
}

func TestArgsErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"bob"}, "missing argument: count"},
		{[]string{"bob", "x", "1s"}, `invalid value "x" for argument count`},
		{[]string{"bob", "1", "1s", "obs#x"}, `invalid value "obs#x" for argument ids`},
	} {
		c := &argCmd{}
		_, stderr := run(t, c, tc.args...)
		if !strings.HasPrefix(stderr, tc.want) || c.ran {
			t.Errorf("%v: stderr %q, want prefix %q (ran %v)", tc.args, stderr, tc.want, c.ran)
		}
	}
}

type oneArgCmd struct {
	Name string `arg:"name"`
}

func (c *oneArgCmd) Help() string                { return "" }
func (c *oneArgCmd) Synopsis() string            { return "" }
func (c *oneArgCmd) Run(_ context.Context) error { return nil }

func TestArgsUnexpected(t *testing.T) {
	if _, stderr := run(t, &oneArgCmd{}, "a", "b"); !strings.HasPrefix(stderr, "unexpected argument: b") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestArgsVariadicNotLast(t *testing.T) {
	_, stderr := run(t, &badVariadicCmd{}, "a")
	if !strings.HasPrefix(stderr, `*cli.badVariadicCmd: variadic argument "files" must be the last one`) {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestArgsHelp(t *testing.T) {
	_, stderr := run(t, &argCmd{}, "-h")
	for _, want := range []string{"Options:", "-v\t", "Arguments:", "  name\n    \twho",
		"  ids...\n    \tobservation ids"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("help misses %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "-name") {
		t.Errorf("arg field listed as flag:\n%s", stderr)
	}
}

type parentCmd struct {
	Project string `arg:"project, project name"`
	sub     Command
}

func (c *parentCmd) Help() string                    { return "" }
func (c *parentCmd) Synopsis() string                { return "" }
func (c *parentCmd) Run(_ context.Context) error     { return nil }
func (c *parentCmd) SubCommands() map[string]Command { return map[string]Command{"sub": c.sub} }

type variadicParentCmd struct {
	Projects []string `arg:"projects"`
	sub      Command
}

func (c *variadicParentCmd) Help() string                    { return "" }
func (c *variadicParentCmd) Synopsis() string                { return "" }
func (c *variadicParentCmd) Run(_ context.Context) error     { return nil }
func (c *variadicParentCmd) SubCommands() map[string]Command { return map[string]Command{"sub": c.sub} }

func TestArgsParentFirst(t *testing.T) {
	child := &argCmd{}
	parent := &parentCmd{sub: child}
	if _, stderr := run(t, parent, "sub", "-v", "vcc", "alice", "3", "2s", "7"); stderr != "" {
		t.Fatalf("stderr: %s", stderr)
	}
	want := &argCmd{Verbose: true, Name: "alice", Count: 3, Wait: 2 * time.Second, IDs: []id{7}, ran: true}
	if parent.Project != "vcc" || !reflect.DeepEqual(child, want) {
		t.Fatalf("got project %q, %+v", parent.Project, child)
	}
}

func TestArgsParentVariadic(t *testing.T) {
	_, stderr := run(t, &variadicParentCmd{sub: &oneArgCmd{}}, "sub", "a", "b")
	if !strings.HasPrefix(stderr, `*cli.oneArgCmd: variadic argument "projects" must be the last one`) {
		t.Fatalf("stderr %q", stderr)
	}
}

type unsupportedArgCmd struct {
	Date *time.Time `arg:"date"`
}

func (c *unsupportedArgCmd) Help() string                { return "" }
func (c *unsupportedArgCmd) Synopsis() string            { return "" }
func (c *unsupportedArgCmd) Run(_ context.Context) error { return nil }

type unsupportedFlagCmd struct {
	Date *time.Time `flag:"date"`
}

func (c *unsupportedFlagCmd) Help() string                { return "" }
func (c *unsupportedFlagCmd) Synopsis() string            { return "" }
func (c *unsupportedFlagCmd) Run(_ context.Context) error { return nil }

func TestUnsupportedType(t *testing.T) {
	for c, want := range map[Command]string{
		&unsupportedArgCmd{}:  `*cli.unsupportedArgCmd: argument "date" (field Date, type *time.Time): unsupported type`,
		&unsupportedFlagCmd{}: `*cli.unsupportedFlagCmd: flag "date" (field Date, type *time.Time): unsupported type`,
	} {
		_, stderr := run(t, c)
		if !strings.HasPrefix(stderr, want+"\n") || strings.Count(stderr, want) != 2 {
			t.Errorf("stderr %q, want %q once plus in the help", stderr, want)
		}
	}
}
