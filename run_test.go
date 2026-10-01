package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type errCmd struct {
	Name string `arg:"name"`
	err  error
}

func (c *errCmd) Help() string                { return "Usage: test run <name>" }
func (c *errCmd) Synopsis() string            { return "run" }
func (c *errCmd) Run(_ context.Context) error { return c.err }

func runErr(t *testing.T, c Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	SetRoot(&Root{Name: "test", subCommands: map[string]Command{"run": c}})
	t.Cleanup(func() { SetRoot(&Root{subCommands: make(map[string]Command)}) })
	var out, errOut bytes.Buffer
	cli := New("test", "1")
	cli.HelpWriter, cli.ErrorWriter = &out, &errOut
	err = cli.Run(context.Background(), append([]string{"test"}, args...))
	return out.String(), errOut.String(), err
}

func TestRunRunError(t *testing.T) {
	want := errors.New("boom")
	stdout, stderr, err := runErr(t, &errCmd{err: want}, "run", "x")
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("runtime error printed: stdout %q stderr %q", stdout, stderr)
	}
}

func TestRunUsageError(t *testing.T) {
	for name, tc := range map[string]struct {
		cmd  *errCmd
		args []string
		msg  string
	}{
		"missing arg":     {&errCmd{}, []string{"run"}, "missing argument"},
		"bad flag":        {&errCmd{}, []string{"run", "-x"}, "flag provided but not defined"},
		"from Run":        {&errCmd{err: Usagef("bad name %q", "x")}, []string{"run", "x"}, `bad name "x"`},
		"unknown command": {&errCmd{}, []string{"nope"}, `unknown command "nope"`},
	} {
		t.Run(name, func(t *testing.T) {
			_, stderr, err := runErr(t, tc.cmd, tc.args...)
			var ue *UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("err = %v (%T), want *UsageError", err, err)
			}
			if !strings.Contains(stderr, tc.msg) || !strings.Contains(stderr, "Usage: test") {
				t.Fatalf("stderr misses %q or the help:\n%s", tc.msg, stderr)
			}
		})
	}
}

func TestRunHelpFlag(t *testing.T) {
	stdout, stderr, err := runErr(t, &errCmd{}, "run", "-h")
	if err != nil || stderr != "" || !strings.Contains(stdout, "Usage: test run <name>") {
		t.Fatalf("err %v, stdout %q, stderr %q", err, stdout, stderr)
	}
}

func TestUseOrder(t *testing.T) {
	SetRoot(&Root{Name: "test", subCommands: map[string]Command{"run": &errCmd{}}})
	t.Cleanup(func() { SetRoot(&Root{subCommands: make(map[string]Command)}) })
	var got []string
	mw := func(name string) Middleware {
		return func(c Command, next RunFunc) RunFunc {
			if _, ok := c.(*errCmd); !ok {
				t.Errorf("middleware got %T", c)
			}
			return func(ctx context.Context) error {
				got = append(got, name+">")
				err := next(ctx)
				got = append(got, "<"+name)
				return err
			}
		}
	}
	cli := New("test", "1")
	cli.Use(mw("a"))
	cli.Use(mw("b"))
	if err := cli.Run(context.Background(), []string{"test", "run", "x"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "a> b> <b <a" {
		t.Fatalf("order %v", got)
	}
}

func TestUseSkipsOnUsageError(t *testing.T) {
	SetRoot(&Root{Name: "test", subCommands: map[string]Command{"run": &errCmd{}}})
	t.Cleanup(func() { SetRoot(&Root{subCommands: make(map[string]Command)}) })
	cli := New("test", "1")
	cli.ErrorWriter = &bytes.Buffer{}
	cli.Use(func(c Command, next RunFunc) RunFunc {
		t.Error("middleware ran for a usage error")
		return next
	})
	if err := cli.Run(context.Background(), []string{"test", "run"}); err == nil {
		t.Fatal("no error for a missing argument")
	}
}
