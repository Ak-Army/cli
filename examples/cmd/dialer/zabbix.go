package dialer

import (
	"context"
	"fmt"

	"github.com/Ak-Army/cli/examples/cmd/base"
)

type Zabbix struct {
	base.Base
}

func (d *Zabbix) Help() string {
	return ""
}

func (d *Zabbix) Synopsis() string {
	return "Get dialer info for zabbix"
}

func (d *Zabbix) Run(ctx context.Context) error {
	fmt.Println("Zabbix")
	return nil
}
