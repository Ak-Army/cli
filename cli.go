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
	Name            string
	Version         string
	Description     string
	Authors         []string
	Commands        *commands
	commandSets     map[string]*commands
	defaultCommand  string
	template        *template.Template
	commandTemplate *template.Template
}

type commands struct {
	Command
	SubCommands map[string]*commands
	flagSetDone bool
}

// New returns new CLI struct
func New(name string, version string) *CLI {
	cli := &CLI{
		Name:           name,
		commandSets:    make(map[string]*commands),
		defaultCommand: "help",
		Version:        version,
	}
	cli.commandSets["root"] = &commands{
		SubCommands: make(map[string]*commands),
		flagSetDone: true,
	}
	cli.SetTemplate(`USAGE:
   {{.Name}}{{if .Commands}} command [command options]{{end}}
VERSION:
   {{.Version}}{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}
AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .Commands.Command}}
COMMAND:
   {{.Name}}: {{printCommand .Commands}}
{{with $subCommands := .Commands.SubCommands}}{{with $subCommandsLength := len $subCommands}}{{if gt $subCommandsLength 0}}SUB COMMANDS DESCRIPTION{{if gt $subCommandsLength 1}}S{{end}}:{{range $subName, $subCommand := $subCommands}}
   {{$subName}}: {{printCommand $subCommand}}
{{end}}{{end}}{{end}}{{end}}{{else}}
COMMAND{{with $length := len .Commands.SubCommands}}{{if ne 1 $length}}S{{end}}{{end}}:{{range $subName, $subCommands := .Commands.SubCommands}}
    {{$subName}}: {{printCommand $subCommands}}
{{end}}{{end}}
`, `{{.Command.Desc}}{{with $help := .Command.Help}}{{if ne $help ""}}
      Options:
        {{replace $help "\n"  "\n        " -1}}{{end}}{{end}}{{with $samples := .Command.Samples}}{{with $sampleLen := len $samples}}{{if gt $sampleLen 1}}
      Samples:
          {{join $samples "\n          "}}{{end}}{{end}}{{end}}{{with $subCommands := .SubCommands}}{{with $subCommandsLength := len $subCommands}}{{if gt $subCommandsLength 0}}
      SUB COMMAND{{if ne 1 $subCommandsLength}}S{{end}}:{{range $subName, $subCommand := $subCommands}}
          {{$subName}}: {{$subCommand.Desc}}{{end}}{{end}}{{end}}{{end}}`)
	return cli
}

// SetTemplate sets the template for the console output
func (cli *CLI) SetTemplate(temp string, commandTemp string) error {
	funcs := template.FuncMap{
		"join":    strings.Join,
		"replace": strings.Replace,
		"printSubCommands": func(commands *commands) string {
			err := cli.commandTemplate.Execute(os.Stderr, commands)
			if err != nil {
				return err.Error()
			}
			return ""
		},
		"printCommand": func(commands *commands) string {
			err := cli.commandTemplate.Execute(os.Stderr, commands)
			if err != nil {
				return err.Error()
			}
			return ""
		},
	}
	ct, err := template.New("command").Funcs(funcs).Parse(commandTemp)
	if err != nil {
		return err
	}
	t, err := template.New("help").Funcs(funcs).Parse(temp)
	if err != nil {
		return err
	}
	cli.template = t
	cli.commandTemplate = ct
	return nil
}

// Add commands
func (cli *CLI) Add(commands ...Command) {
	cli.addTo(commands, cli.commandSets["root"])
}

// SetDefault sets default command
func (cli *CLI) SetDefault(command string) {
	cli.defaultCommand = command
}

// Run parses the arguments and runs the applicable command
func (cli *CLI) Run(ctx context.Context, args []string) {
	if len(args) == 1 {
		args = append(args, cli.defaultCommand)
	}
	c, err := cli.getSubCommand(cli.commandSets["root"], args[1:])
	if err != nil {
		fmt.Println("INVALID PARAMETER:")
		fmt.Println("  ", err)
		fmt.Println()
		cli.help(c)
		return
	}
	if c == nil {
		cli.help(cli.commandSets["root"])
		return

	}
	c.Command.Run(ctx)
}

func (cli *CLI) getSubCommand(commandSet *commands, args []string) (*commands, error) {
	for name, c := range commandSet.SubCommands {
		if name == args[0] {
			cli.getFlagSet(c)
			if len(c.SubCommands) > 0 {
				if len(args) <= 1 {
					return c, errors.New("missing sub command")
				}
				subC, err := cli.getSubCommand(c, args[1:])
				if err != nil {
					return c, err
				}
				if subC == nil {
					return c, errors.New("missing sub command")
				}
				return subC, nil
			}
			if len(args) > 1 {
				if err := c.Command.Parse(args[2:]); err != nil {
					return c, err
				}
			} else {
				if err := c.Command.Parse([]string{}); err != nil {
					return c, err
				}
			}
			return c, nil
		}
	}
	return nil, nil
}

func (cli *CLI) addTo(commandList []Command, commandSet *commands) {
	for _, c := range commandList {
		if reflect.TypeOf(c).Kind() != reflect.Ptr {
			continue
		}
		t := reflect.TypeOf(c).Elem()
		v := reflect.ValueOf(c).Elem()
		flagger := v.FieldByNameFunc(func(s string) bool { return strings.Contains(s, "Flagger") })
		if !flagger.CanSet() {
			continue
		}
		flagger.Set(reflect.ValueOf(&Flagger{FlagSet: &flag.FlagSet{}}))
		name := strings.ToLower(t.Name())

		commandSet.SubCommands[name] = &commands{
			Command:     c,
			SubCommands: make(map[string]*commands),
		}
		if subCommands := c.SubCommands(); len(subCommands) > 0 {
			cli.addTo(subCommands, commandSet.SubCommands[name])
		}
	}
}

func (cli *CLI) help(command *commands) {
	cli.Commands = command
	cli.getFlagSet(command)
	for _, c := range command.SubCommands {
		cli.getFlagSet(c)
	}
	cli.template.Execute(os.Stderr, cli)
}

func (cli *CLI) getFlagSet(c *commands) error {
	if c.flagSetDone {
		return nil
	}
	c.flagSetDone = true
	fs := c.Command.getFlagSet()
	st := reflect.ValueOf(c.Command)
	if st.Kind() != reflect.Ptr {
		return errors.New("pointer expected")
	}
	if err := cli.defineFlagSet(fs, st); err != nil {
		return err
	}
	return nil
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
