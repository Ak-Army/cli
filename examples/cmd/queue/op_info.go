package queue

import (
	"context"
	"fmt"

	"github.com/Ak-Army/cli/examples/cmd/base"
)

type OpInfo struct {
	base.Base
	Customer string `flag:"customer, print just the customer info"`
	UserId   int64  `flag:"userId, print just one user for a customer"`
}

func (d *OpInfo) Help() string {
	return ""
}

func (d *OpInfo) Synopsis() string {
	return "Get queue operator info"
}

func (d *OpInfo) Run(ctx context.Context) error {
	fmt.Println("queue OpInfo")
	return nil
}
