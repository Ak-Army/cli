package cmd

import (
	"context"
	"errors"

	"github.com/Ak-Army/cli"
	"github.com/Ak-Army/cli/examples/cmd/dialer"
)

func init() {
	cli.RootCommand().AddCommand("dialer", &Dialer{})
}

type Dialer struct{}

func (d *Dialer) Help() string {
	return `
Usage: archiver dialer <command> [command options]
`
}

func (d *Dialer) Synopsis() string {
	return "Interact with the dialer service"
}

func (d *Dialer) SubCommands() map[string]cli.Command {
	return map[string]cli.Command{
		"info":    &dialer.Info{},
		"op_info": &dialer.OpInfo{},
		"zabbix":  &dialer.Zabbix{},
	}
}
func (d *Dialer) Run(ctx context.Context) error {
	return errors.New("Select a sub command")
}
