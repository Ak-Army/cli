package cli

import (
	"context"
	"flag"
	"io"
	"time"
)

// A command is a runnable command of a CLI.
type Command interface {
	// Help should return long-form help text that includes the command-line
	// usage, a brief few sentences explaining the function of the command.
	Help() string

	// Synopsis should return a one-line, short synopsis of the command.
	Synopsis() string

	// Run should run the actual command with the given Context
	Run(ctx context.Context) error
}

type SubCommands interface {
	// SubCommand should return a list of sub commands
	SubCommands() map[string]Command
}

type ParseHelper interface {
	// Parse should help to validate flags, and add extra options
	Parse([]string) error
}

// Flagger is an interface satisfied by flag.FlagSet and other implementations
// of flags.
type Flagger interface {
	Parse([]string) error
	StringVar(p *string, name string, value string, usage string)
	IntVar(p *int, name string, value int, usage string)
	Int64Var(p *int64, name string, value int64, usage string)
	BoolVar(p *bool, name string, value bool, usage string)
	UintVar(p *uint, name string, value uint, usage string)
	Uint64Var(p *uint64, name string, value uint64, usage string)
	Float64Var(p *float64, name string, value float64, usage string)
	DurationVar(p *time.Duration, name string, value time.Duration, usage string)
	Set(name string, value string) error
	SetOutput(output io.Writer)
	Var(value flag.Value, name string, usage string)
	VisitAll(fn func(*flag.Flag))
}
