package queue

import (
	"context"
	"fmt"

	"github.com/Ak-Army/cli/examples/cmd/base"

	"github.com/sgreben/flagvar"
)

type Info struct {
	base.Base `flag:"base"`
	Customer  flagvar.Strings `flag:"customer, print just the customer info"`
	QueueId   int64           `flag:"queueId, print just one queue for a customer"`
}

func (d *Info) Help() string {
	return `Usage: archiver queue info [options]`
}

func (d *Info) Synopsis() string {
	return "Get queue projects info"
}

func (q *Info) Run(ctx context.Context) error {
	fmt.Printf("queue info %#v", q)
	return nil
}
