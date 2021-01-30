package cmd

import (
	"context"
	"errors"

	"github.com/Ak-Army/cli"
	"github.com/Ak-Army/cli/examples/cmd/base"
	"github.com/Ak-Army/cli/examples/cmd/queue"
)

func init() {
	cli.RootCommand().AddCommand("queue", &Queue{})
}

type Queue struct {
	base.Base
}

func (d *Queue) Help() string {
	return `Usage: archiver queue <command> [options]`
}

func (d *Queue) Synopsis() string {
	return "Interact with the queue service"
}

func (d *Queue) SubCommands() map[string]cli.Command {
	return map[string]cli.Command{
		"info":    &queue.Info{},
		"op_info": &queue.OpInfo{},
	}
}

func (d *Queue) Parse([]string) error {
	return nil
}

func (d *Queue) Run(ctx context.Context) error {
	return errors.New("Select a sub command")
}
