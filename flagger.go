package cli

import (
	"bytes"
	"flag"
)

// Command is an interface for cli commands
type Command interface {
	Desc() string
	Samples() []string
	Run()
	Parse([]string) error
	GetFlagSet() *flag.FlagSet
	Help() string
}

// Flagger is a helper struct for commands
type Flagger struct {
	*flag.FlagSet
	Output bytes.Buffer
}

func (f *Flagger) GetFlagSet() *flag.FlagSet {
	f.FlagSet.Usage = func() {}
	f.FlagSet.SetOutput(&f.Output)
	return f.FlagSet
}

func (f *Flagger) Help() string {
	f.PrintDefaults()
	return f.Output.String()
}

func (f *Flagger) Samples() []string {
	return []string{}
}
