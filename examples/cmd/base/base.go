package base

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Base struct {
	Verbose bool `flag:"v, print debug and info messages"`
}

func (b *Base) PrettyPrint(resp interface{}) error {
	r, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	defer out.Reset()
	json.Indent(&out, r, "", "\t")
	fmt.Println(out.String())

	return nil
}
