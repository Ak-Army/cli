// cli is a simple, fast package for building command line apps in Go. It's a wrapper around the "flag" package.
//
// Example usage
//
// Declare a struct type that embeds *cli.Flagger, along with an fields you want to capture as flags.
//
//     type Echo struct {
//         *cli.Flagger
//         Echoed string `flag:"echoed, echo this string"`
//     }
//
// Package understands all basic types supported by flag's package xxxVar functions: int, int64, uint, uint64, float64, bool, string, time.Duration. Types implementing flag.Value interface are also supported.
//
//     type CustomDate string
//
//     func (c *CustomDate) String() string {
//         return fmt.Sprint(*c)
//     }
//
//     func (c *CustomDate) Set(value string) error {
//         dateRegex := `^20\d{2}(\/|-)(0[1-9]|1[0-2])(\/|-)(0[1-9]|[12][0-9]|3[01])$`
//         if ok, err := regexp.MatchString(dateRegex, value); err != nil || !ok {
//             return errors.New("from parameter is not a valid date")
//         }
//         *c = CustomDate(value)
//         return nil
//     }
//
//     type EchoWithDate struct {
//         *cli.Flagger
//         Echoed string `flag:"echoed, echo this string"`
//         EchoWithDate CustomDate `flag:"echoDate, echo this date too"`
//     }
//
// Now we need to make our type implement the cli.Command interface. That requires three methods that aren't already provided by *cli.Flagger:
//
//     func (c *Echo) Desc() string {
//         return "Echo the input string."
//     }
//
//     func (c *Echo) Run() {
//         fmt.Println(c.Echoed)
//     }
//
// Maybe we write sample command runs:
//
//     func (c *Echo) Samples() []string {
//         return []string{"echoprogram -echoed=\"echo this\"",
//         "echoprogram -echoed=\"or echo this\""}
//     }
//
// We can set default command to run
//
//     c.SetDefault("echo")
//
// After all of this, we can run them like this:
//
//     func main() {
//         c := cli.New("echoer", "1.0.0")
//         c.Authors = []string{"authors goes here"}
//         c.Add(
//             &Echo{
//                 Echoed: "default string",
//             })
//         //c.SetDefaults("echo")
//         c.Run(os.Args)
//     }
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
	"time"
)

// CLI defines a new command line interface
type CLI struct {
	Name           string
	Version        string
	Description    string
	Authors        []string
	Commands       map[string]Command
	commandSets    map[string]Command
	defaultCommand string
	template       *template.Template
}

// New returns new CLI struct
func New(name string, version string) *CLI {
	cli := &CLI{
		Name:           name,
		Commands:       make(map[string]Command),
		commandSets:    make(map[string]Command),
		defaultCommand: "help",
		Version:        version,
	}
	cli.SetTemplate(`USAGE:
   {{.Name}}{{if .Commands}} command [command options]{{end}}
VERSION:
   {{.Version}}{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}
AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .Commands}}
COMMAND{{with $length := len .Commands}}{{if ne 1 $length}}S{{end}}{{end}}:{{range $name, $command := .Commands}}
   {{$name}}: {{$command.Desc}}
      Options:
        {{replace $command.Help "\n"  "\n        " -1}}{{if $command.Samples}}
      Samples:
          {{join $command.Samples "\n          "}}{{end}}
{{end}}{{end}}
`)
	return cli
}

// SetTemplate sets the template for the console output
func (cli *CLI) SetTemplate(temp string) error {
	t, err := template.New("help").Funcs(template.FuncMap{
		"join":    strings.Join,
		"replace": strings.Replace,
	}).Parse(temp)
	if err != nil {
		return err
	}
	cli.template = t
	return nil
}

// Add add commands
func (cli *CLI) Add(commands ...Command) {
	for _, c := range commands {
		t := reflect.TypeOf(c).Elem()
		v := reflect.ValueOf(c).Elem()
		flagger := v.FieldByNameFunc(func(s string) bool { return strings.Contains(s, "Flagger") })
		if !flagger.CanSet() {
			continue
		}
		flagger.Set(reflect.ValueOf(&Flagger{FlagSet: &flag.FlagSet{}}))
		name := strings.ToLower(t.Name())
		cli.commandSets[name] = c

	}
}

// SetDefault sets default command
func (cli *CLI) SetDefault(command string) {
	cli.defaultCommand = command
}

// Run parses the arguments and runs the applicable command
func (cli *CLI) Run(ctx context.Context, args []string) {
	command := cli.defaultCommand
	if len(args) > 1 {
		command = args[1]
	}
	for name, c := range cli.commandSets {
		if name == command {
			cli.getFlagSet(c)
			var err error
			if len(args) > 1 {
				err = c.Parse(args[2:])
			} else {
				err = c.Parse([]string{})
			}
			if err != nil {
				fmt.Println("INVALID PARAMETER:")
				fmt.Println("  ", err)
				fmt.Println()
				cli.help(command)
				return
			}
			c.Run(ctx)
			return
		}
	}
	cli.help("")
}

func (cli *CLI) help(commandName string) {
	if commandName != "" {
		if command, ok := cli.commandSets[commandName]; ok {
			cli.Commands[commandName] = command
			cli.template.Execute(os.Stderr, cli)
		}
	} else {
		for _, command := range cli.commandSets {
			cli.getFlagSet(command)
		}
		cli.Commands = cli.commandSets
		cli.template.Execute(os.Stderr, cli)
	}
}

func (cli *CLI) getFlagSet(command Command) error {
	fs := command.GetFlagSet()
	st := reflect.ValueOf(command)
	if st.Kind() != reflect.Ptr {
		return errors.New("pointer expected")
	}
	return cli.defineFlagSet(fs, st)
}

func (cli *CLI) defineFlagSet(fs *flag.FlagSet, st reflect.Value) error {
	st = reflect.Indirect(st)
	if !st.IsValid() || st.Type().Kind() != reflect.Struct {
		return errors.New("non-nil pointer to struct expected")
	}
	flagValueType := reflect.TypeOf((*flag.Value)(nil)).Elem()
	for i := 0; i < st.NumField(); i++ {
		typ := st.Type().Field(i)
		var name, usage string
		tag := typ.Tag.Get("flag")
		if typ.Type.Kind() == reflect.Struct {
			if err := cli.defineFlagSet(fs, st.Field(i)); err != nil {
				return err
			}
			continue
		}
		if tag == "" {
			continue
		}
		val := st.Field(i)
		if !val.CanInterface() {
			return errors.New("field is unexported")
		}
		if !val.CanAddr() {
			return errors.New("field is of unsupported type")
		}
		flagData := strings.SplitN(tag, ",", 2)
		switch len(flagData) {
		case 1:
			name = flagData[0]
		case 2:
			name, usage = flagData[0], flagData[1]
		}
		addr := val.Addr()
		if addr.Type().Implements(flagValueType) {
			fs.Var(addr.Interface().(flag.Value), name, usage)
			continue
		}
		switch d := val.Interface().(type) {
		case int:
			fs.IntVar(addr.Interface().(*int), name, d, usage)
		case int64:
			fs.Int64Var(addr.Interface().(*int64), name, d, usage)
		case uint:
			fs.UintVar(addr.Interface().(*uint), name, d, usage)
		case uint64:
			fs.Uint64Var(addr.Interface().(*uint64), name, d, usage)
		case float64:
			fs.Float64Var(addr.Interface().(*float64), name, d, usage)
		case bool:
			fs.BoolVar(addr.Interface().(*bool), name, d, usage)
		case string:
			fs.StringVar(addr.Interface().(*string), name, d, usage)
		case time.Duration:
			fs.DurationVar(addr.Interface().(*time.Duration), name, d, usage)
		default:
			return errors.New(fmt.Sprintf("field with flag tag value %q is of unsupported type", name))
		}
	}
	return nil
}
