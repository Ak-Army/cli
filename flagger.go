package cli

import (
	"bytes"
	"context"
	"flag"
)

// Command is an interface for cli commands
type Command interface {
	Desc() string
	Samples() []string
	Run(ctx context.Context)
	Parse([]string) error
	Help() string
	SubCommands() []Command
	getFlagSet() *flag.FlagSet
}

// Flagger is a helper struct for commands
type Flagger struct {
	*flag.FlagSet
	out bytes.Buffer
}

func (f *Flagger) getFlagSet() *flag.FlagSet {
	f.FlagSet.Usage = func() {}
	f.FlagSet.SetOutput(&f.out)
	return f.FlagSet
}

func (f *Flagger) Help() string {
	f.PrintDefaults()
	return f.out.String()
}

func (f *Flagger) Samples() []string {
	return []string{}
}

func (f *Flagger) SubCommands() []Command {
	return nil
}
