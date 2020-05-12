package cli

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type FlaggerTestSuite struct {
	suite.Suite
	cli *CLI
}

type TestCommand struct {
	*Flagger
	Verbose bool `flag:"v"`
}

func (c *TestCommand) Desc() string {
	return "Test description"
}
func (c *TestCommand) Run(ctx context.Context) {
	fmt.Println("its okey")
}
func (c *TestCommand) Samples() []string {
	return []string{"test samle"}
}

func (suite *FlaggerTestSuite) SetupTest() {
	suite.cli = New("test", "1.0.0")
}

func TestSafeList(t *testing.T) {
	suite.Run(t, new(FlaggerTestSuite))
}

func (suite *FlaggerTestSuite) TestEmptyFlag() {
	suite.cli.Add(&TestCommand{})
	suite.Nil(suite.cli.getFlagSet(suite.cli.commandSets["testcommand"]))
}

func (suite *FlaggerTestSuite) TestWrongFlag() {
	type testcommand2 struct {
		*TestCommand
		Apple *interface{} `flag:"apple"`
	}
	suite.cli.Add(&testcommand2{})
	suite.NotNil(suite.cli.getFlagSet(suite.cli.commandSets["testcommand2"]))
}

func (suite *FlaggerTestSuite) TestUnexportedFlag() {
	type testcommand2 struct {
		*TestCommand
		Apple *interface{} `flag:"apple"`
	}
	suite.cli.Add(&testcommand2{})
	suite.NotNil(suite.cli.getFlagSet(suite.cli.commandSets["testcommand2"]))
}

type CustomFlag []string

func (c *CustomFlag) String() string { return fmt.Sprint(*c) }
func (c *CustomFlag) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func (suite *FlaggerTestSuite) TestFlag() {
	type testcommand2 struct {
		TestCommand
		String   string        `flag:"string,string flag example"`
		Int      int           `flag:"int,int flag example"`
		Int64    int64         `flag:"int64,int64 flag example"`
		Uint     uint          `flag:"uint,uint flag example"`
		Uint64   uint64        `flag:"uint64"`
		Float64  float64       `flag:"float64"`
		Bool     bool          `flag:"bool"`
		Duration time.Duration `flag:"duration"`
		MySlice  CustomFlag    `flag:"slice"` // custom flag.Value implementation

		Empty      bool `flag:""` // empty flag definition
		NonExposed int  // does not have flag attached
	}
	reference := testcommand2{
		TestCommand: TestCommand{
			Verbose: true,
		},
		String:   "whales",
		Int:      42,
		Int64:    100 << 30,
		Uint:     7,
		Uint64:   24,
		Float64:  1.55,
		Bool:     true,
		Duration: 15 * time.Minute,
		MySlice:  CustomFlag{"a", "b"},
	}
	conf := testcommand2{}
	suite.cli.Add(&conf)
	suite.Nil(suite.cli.getFlagSet(&conf))

	args := []string{
		"-string", "whales", "-int", "42",
		"-int64", "107374182400", "-uint", "7",
		"-uint64", "24", "-float64", "1.55", "-bool",
		"-duration", "15m",
		"-slice", "a",
		"-slice", "b",
		"-v",
	}
	suite.Nil(conf.Parse(args))
	conf.Flagger = nil
	suite.Equal(reference, conf)
}

func (suite *FlaggerTestSuite) TestDefaultValueFlag() {
	type testcommand2 struct {
		TestCommand
		String   string        `flag:"string,string flag example"`
		Int      int           `flag:"int,int flag example"`
		Int64    int64         `flag:"int64,int64 flag example"`
		Uint     uint          `flag:"uint,uint flag example"`
		Uint64   uint64        `flag:"uint64"`
		Float64  float64       `flag:"float64"`
		Bool     bool          `flag:"bool"`
		Duration time.Duration `flag:"duration"`
		MySlice  CustomFlag    `flag:"slice"` // custom flag.Value implementation

		Empty      bool `flag:""` // empty flag definition
		NonExposed int  // does not have flag attached
	}
	reference := testcommand2{
		String:   "whales",
		Int:      42,
		Int64:    100 << 30,
		Uint:     7,
		Uint64:   24,
		Float64:  1.55,
		Bool:     true,
		Duration: 15 * time.Minute,
		MySlice:  CustomFlag{"a", "b"},
	}
	conf := testcommand2{
		String: "whales",
	}
	suite.cli.Add(&conf)
	suite.Nil(suite.cli.getFlagSet(&conf))

	args := []string{
		"-int", "42",
		"-int64", "107374182400", "-uint", "7",
		"-uint64", "24", "-float64", "1.55", "-bool",
		"-duration", "15m",
		"-slice", "a",
		"-slice", "b",
	}
	suite.Nil(conf.Parse(args))
	conf.Flagger = nil
	suite.Equal(reference, conf)
}

func (suite *FlaggerTestSuite) TestParseError() {
	type testcommand2 struct {
		TestCommand
		Int int `flag:"int,int flag example"`
	}

	conf := testcommand2{}
	suite.cli.Add(&conf)
	suite.Nil(suite.cli.getFlagSet(&conf))

	args := []string{
		"-int", "fasdf",
	}
	suite.NotNil(conf.Parse(args))
}

type CustomFlagError []string

func (c *CustomFlagError) String() string { return fmt.Sprint(*c) }
func (c *CustomFlagError) Set(value string) error {
	*c = append(*c, value)
	return errors.New("its a trap")
}

func (suite *FlaggerTestSuite) TestParseErrorCustom() {
	type testcommand2 struct {
		TestCommand
		Int CustomFlagError `flag:"int,int flag example"`
	}

	conf := testcommand2{}
	suite.cli.Add(&conf)
	suite.Nil(suite.cli.getFlagSet(&conf))

	args := []string{
		"-int", "fasdf",
	}
	suite.NotNil(conf.Parse(args))
}
