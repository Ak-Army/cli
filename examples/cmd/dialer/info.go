package dialer

import (
	"context"
	"fmt"

	"github.com/Ak-Army/cli/examples/cmd/base"
)

type Info struct {
	base.Base `flag:"base"`
	Customer  string `flag:"customer, print just the customer info"`
	QueueId   int64  `flag:"queueId, print just one queue for a customer"`
}

func (d *Info) Help() string {
	return `Usage: archiver dialer info [options]`
}

func (d *Info) Synopsis() string {
	return "Get dialer projects info"
}

func (q *Info) Run(ctx context.Context) error {
	fmt.Printf("project info %#v", q)
	return nil
}
