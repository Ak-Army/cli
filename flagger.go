package cli

import (
	"bufio"
	"bytes"
	"flag"
)

type Command interface {
	Desc() string
	Samples() []string
	Run()
	Parse([]string) error
	GetFlagSet() *flag.FlagSet
	Help() string
}

type Flagger struct {
	*flag.FlagSet
	Output bytes.Buffer
}

func (f *Flagger) GetFlagSet() *flag.FlagSet {
	f.FlagSet.Usage = func() {}
	f.FlagSet.SetOutput(bufio.NewWriter(&f.Output))
	return f.FlagSet
}

func (f *Flagger) Help() string {
	b := &bytes.Buffer{}
	f.FlagSet.SetOutput(b)
	f.PrintDefaults()
	f.FlagSet.SetOutput(bufio.NewWriter(&f.Output))
	return b.String()
}

func (f *Flagger) Samples() []string {
	return []string{}
}
