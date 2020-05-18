// cli is a simple, fast package for building command line apps in Go. It's a wrapper around the "flag" package.
//
// Example usage
//
// Declare a struct type which implement cli.Command interface.
//
//     type Echo struct {
//         Echoed string `flag:"echoed, echo this string"`
//     }
//
// Package understands all basic types supported by flag's package xxxVar functions:
// int, int64, uint, uint64, float64, bool, string, time.Duration.
// Types implementing flag.Value interface are also supported.
// (Useful package: https://github.com/sgreben/flagvar)
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
//         Echoed string `flag:"echoed, echo this string"`
//         EchoWithDate CustomDate `flag:"echoDate, echo this date too"`
//     }
//
// Now we need to make our type implement the cli.Command interface.
//
//     func (c *Echo) Help() string {
//         return "Echo the input string."
//     }
//
//     func (c *Echo) Synopsis() string {
//         return "Short one liner about the command"
//     }
//
// Maybe we write sample command runs:
//
//     func (c *Echo) Run(ctx_ context.Context) error {
//         return nil
//     }
//
// We can set default command to run
//
//     c.SetDefault("echo")
//
// After all of this, we can run them like this:
//
//     func main() {
//         c := cli.New("archiver", "1.0.0")
//         cli.RootCommand().Authors = []string{"authors goes here"}
//         cli.RootCommand().Description = `Lorem Ipsum is simply dummy text of the printing and typesetting industry.
// Lorem Ipsum has been the industry's standard dummy text ever since the 1500s`
//
//         cli.RootCommand().AddCommand("echo", &Echo{})
//         c.Run(context.Background(), os.Args)
//     }
package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	completeLine        = "COMP_LINE"
	completePoint       = "COMP_POINT"
	defaultHelpTemplate = `{{.Help}}
{{with $flags := flagSet .Command}}{{if ne $flags ""}}
Options:
{{$flags}}{{- end }}{{end}}{{if gt (len .SubCommands) 0}}
Commands:
{{- range $name, $value := .SubCommands }}
    {{$value.NameAligned}}    {{$value.Synopsis}}{{with $flags := flagSet $value.Command}}{{if ne $flags ""}}
        Options:
        {{replace $flags "\n" "\n        " -1}}{{- end }}{{end}}
{{- end }}{{end}}
`
)

// CLI defines a new command line interface
type CLI struct {
	// HelpWriter is used to print help text and version when requested.
	HelpWriter io.Writer
	// ErrorWriter used to output errors when a command can not be run.
	ErrorWriter io.Writer
	// AutoComplete used to handle autocomplete request from bash or zsh.
	AutoComplete     bool
	root             *Root
	defaultCommand   string
	flagSet          Flagger
	flagSetOut       bytes.Buffer
	template         string
	lastCommandsName []string
}

// New returns a new CLI struct
func New(name string, version string) *CLI {
	cli := &CLI{
		root:         RootCommand(),
		HelpWriter:   os.Stdout,
		AutoComplete: true,
		ErrorWriter:  os.Stderr,
		flagSet: &flag.FlagSet{
			Usage: func() {},
		},
		template: defaultHelpTemplate,
	}
	cli.flagSet.SetOutput(&cli.flagSetOut)
	cli.root.Name = name
	cli.root.Version = version

	return cli
}

// SetDefault sets default command
func (cli *CLI) SetDefault(command string) {
	cli.defaultCommand = command
}

// Run parses the arguments and runs the applicable command
func (cli *CLI) Run(ctx context.Context, args []string) {
	doComplete := false
	if line, ok := cli.isCompleteStarted(); ok {
		if !cli.AutoComplete {
			return
		}
		args = strings.Split(line, " ")
		doComplete = true
	}
	if len(args) == 1 {
		args = append(args, cli.defaultCommand)
	}
	c, err := cli.getSubCommand(cli.root, args[1:])
	if doComplete {
		lastArg := strings.TrimLeft(args[len(args)-1], "-")
		if len(cli.lastCommandsName) > 0 {
			for _, name := range cli.lastCommandsName {
				if strings.HasPrefix(name, lastArg) {
					cli.HelpWriter.Write([]byte(name + "\n"))
				}
			}
		} else {
			cli.flagSet.VisitAll(func(f *flag.Flag) {
				if strings.HasPrefix(f.Name, lastArg) {
					cli.HelpWriter.Write([]byte("-" + f.Name + "\n"))
				}
			})
		}
		return
	}
	if err != nil {
		cli.help(c, err)
		return
	}
	if c == nil {
		cli.help(cli.root, nil)
		return

	}
	if err := c.Run(ctx); err != nil {
		cli.help(c, err)
	}
}

// SetTemplate set a new template for commands
func (cli *CLI) SetTemplate(template string) {
	cli.template = template
}

// SetFlagSet set an different flag parser
func (cli *CLI) SetFlagSet(flagSet Flagger) {
	cli.flagSet = flagSet
	cli.flagSet.SetOutput(&cli.flagSetOut)
}

func (cli *CLI) getSubCommand(command SubCommands, args []string) (Command, error) {
	cli.lastCommandsName = []string{}
	for name, c := range command.SubCommands() {
		cli.lastCommandsName = append(cli.lastCommandsName, name)
		if name == args[0] {
			if err := cli.getFlagSet(c); err != nil {
				return c, err
			}
			if subC, ok := c.(SubCommands); ok {
				if len(args) <= 1 {
					return c, errors.New("missing sub command")
				}
				subC, err := cli.getSubCommand(subC, args[1:])
				if subC != nil {
					cli.lastCommandsName = []string{}
				}
				if err != nil {
					if subC == nil {
						subC = c
					}
					return subC, err
				}
				if subC == nil {
					return c, errors.New("wrong sub command")
				}
				return subC, nil
			}
			var parseArg []string
			if len(args) > 1 {
				parseArg = args[1:]
			}
			if err := cli.flagSet.Parse(parseArg); err != nil {
				return c, err
			}
			return c, nil
		}
	}
	return nil, nil
}

func (cli *CLI) help(c Command, err error) {
	output := cli.HelpWriter
	if err != nil {
		output = cli.ErrorWriter
		output.Write([]byte(err.Error() + "\n\n"))
	}
	t, err := template.New("root").Funcs(template.FuncMap{
		"replace": strings.Replace,
		"flagSet": func(c Command) string {
			fs := &flag.FlagSet{
				Usage: func() {},
			}
			var out bytes.Buffer
			fs.SetOutput(&out)
			st := reflect.ValueOf(c)
			if st.Kind() != reflect.Ptr {
				return err.Error()
			}
			if err := cli.defineFlagSet(fs, st, ""); err != nil {
				return err.Error()
			}
			fs.PrintDefaults()
			return out.String()
		}}).Parse(defaultHelpTemplate)
	if err != nil {
		cli.ErrorWriter.Write([]byte(fmt.Sprintf(
			"Internal error! Failed to parse command help template: %s\n", err)))
		return
	}
	s := struct {
		Command
		SubCommands map[string]interface{}
	}{
		Command:     c,
		SubCommands: make(map[string]interface{}),
	}
	if subCs, ok := c.(SubCommands); ok {
		longest := 0
		subC := subCs.SubCommands()
		for k, _ := range subC {
			if v := len(k); v > longest {
				longest = v
			}
		}
		for name, command := range subC {
			c := command
			s.SubCommands[name] = map[string]interface{}{
				"Command":     c,
				"Synopsis":    c.Synopsis(),
				"Help":        c.Help(),
				"NameAligned": name + strings.Repeat(" ", longest-len(name)),
			}
		}
	}
	t.Execute(output, s)
}

func (cli *CLI) getFlagSet(c Command) error {
	st := reflect.ValueOf(c)
	if st.Kind() != reflect.Ptr {
		return errors.New("pointer expected")
	}
	if err := cli.defineFlagSet(cli.flagSet, st, ""); err != nil {
		return err
	}
	return nil
}

func (cli *CLI) defineFlagSet(fs Flagger, st reflect.Value, subName string) error {
	st = reflect.Indirect(st)
	if !st.IsValid() || st.Type().Kind() != reflect.Struct {
		return errors.New("non-nil pointer for struct expected")
	}
	flagValueType := reflect.TypeOf((*flag.Value)(nil)).Elem()
	for i := 0; i < st.NumField(); i++ {
		typ := st.Type().Field(i)
		var name, usage string
		tag := typ.Tag.Get("flag")
		if tag == "" {
			if typ.Type.Kind() == reflect.Struct {
				if err := cli.defineFlagSet(fs, st.Field(i), ""); err != nil {
					return err
				}
				continue
			}
			continue
		}
		val := st.Field(i)
		if !val.CanInterface() {
			return errors.New("field is unexported")
		}
		if !val.CanAddr() {
			return errors.New("field is unsupported type")
		}
		flagData := strings.SplitN(tag, ",", 2)
		switch len(flagData) {
		case 1:
			name = flagData[0]
		case 2:
			name, usage = flagData[0], flagData[1]
		}
		if name == "-" {
			continue
		}
		if subName != "" {
			name = subName + "." + name
		}
		addr := val.Addr()
		if addr.Type().Implements(flagValueType) {
			fs.Var(addr.Interface().(flag.Value), name, usage)
			continue
		} else if typ.Type.Kind() == reflect.Struct {
			if err := cli.defineFlagSet(fs, st.Field(i), name); err != nil {
				return err
			}
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

func (cli *CLI) isCompleteStarted() (string, bool) {
	line := os.Getenv(completeLine)
	if line == "" {
		return "", false
	}
	point, err := strconv.Atoi(os.Getenv(completePoint))
	if err == nil && point > 0 && point < len(line) {
		line = line[:point]
	}
	return line, true
}
