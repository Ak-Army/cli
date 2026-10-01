// cli is a simple, fast package for building command line apps in Go. It's a wrapper around the "flag" package.
//
// # Example usage
//
// Declare a struct type which implement cli.Command interface.
//
//	type Echo struct {
//	    Echoed string `flag:"echoed, echo this string"`
//	}
//
// Package understands all basic types supported by flag's package xxxVar functions:
// int, int64, uint, uint64, float64, bool, string, time.Duration.
// Types implementing flag.Value interface are also supported.
// (Useful package: https://github.com/sgreben/flagvar)
//
//	type CustomDate string
//
//	func (c *CustomDate) String() string {
//	    return fmt.Sprint(*c)
//	}
//
//	func (c *CustomDate) Set(value string) error {
//	    dateRegex := `^20\d{2}(\/|-)(0[1-9]|1[0-2])(\/|-)(0[1-9]|[12][0-9]|3[01])$`
//	    if ok, err := regexp.MatchString(dateRegex, value); err != nil || !ok {
//	        return errors.New("from parameter is not a valid date")
//	    }
//	    *c = CustomDate(value)
//	    return nil
//	}
//
//	type EchoWithDate struct {
//	    Echoed string `flag:"echoed, echo this string"`
//	    EchoWithDate CustomDate `flag:"echoDate, echo this date too"`
//	}
//
// Now we need to make our type implement the cli.Command interface.
//
//	func (c *Echo) Help() string {
//	    return "Echo the input string."
//	}
//
//	func (c *Echo) Synopsis() string {
//	    return "Short one liner about the command"
//	}
//
// Maybe we write sample command runs:
//
//	func (c *Echo) Run(ctx_ context.Context) error {
//	    return nil
//	}
//
// We can set default command to run
//
//	c.SetDefault("echo")
//
// After all of this, we can run them like this:
//
//	func main() {
//	    c := cli.New("archiver", "1.0.0")
//	    cli.RootCommand().Authors = []string{"authors goes here"}
//	    cli.RootCommand().Description = `Lorem Ipsum is simply dummy text of the printing and typesetting industry.
//
// Lorem Ipsum has been the industry's standard dummy text ever since the 1500s`
//
//	    cli.RootCommand().AddCommand("echo", &Echo{})
//	    c.Run(context.Background(), os.Args)
//	}
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
{{$flags}}{{- end }}{{end}}{{with $args := argList .Command}}{{if ne $args ""}}
Arguments:
{{$args}}{{- end }}{{end}}{{if gt (len .SubCommands) 0}}
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
	arguments        arguments
	flagSetOut       bytes.Buffer
	template         string
	lastCommandsName []string
	middlewares      []Middleware
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

// Run parses the arguments, runs the applicable command and returns its
// error. A *UsageError is printed with the help of its command before it is
// returned; any other error is returned unprinted, for the caller to report.
// A help request (-h) prints the help and returns nil.
func (cli *CLI) Run(ctx context.Context, args []string) error {
	c, err := cli.execute(ctx, args)
	if errors.Is(err, flag.ErrHelp) {
		cli.help(c, nil)
		return nil
	}
	var ue *UsageError
	if errors.As(err, &ue) {
		cli.help(c, err)
	}
	return err
}

// Use adds a middleware around the Run of every command. The first one added
// is the outermost.
func (cli *CLI) Use(mw Middleware) {
	cli.middlewares = append(cli.middlewares, mw)
}

// execute resolves and runs the command of args. It returns the command the
// error belongs to; argument errors are wrapped in a *UsageError.
func (cli *CLI) execute(ctx context.Context, args []string) (Command, error) {
	doComplete := false
	if line, ok := cli.isCompleteStarted(); ok {
		if !cli.AutoComplete {
			return nil, nil
		}
		args = strings.Split(line, " ")
		doComplete = true
	}
	if len(args) == 1 {
		args = append(args, cli.defaultCommand)
	}
	if strings.HasPrefix(args[1], "-") {
		args = append([]string{args[0], cli.defaultCommand}, args[1:]...)
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
		return nil, nil
	}
	if err != nil {
		var ue *UsageError
		if !errors.As(err, &ue) && !errors.Is(err, flag.ErrHelp) {
			err = &UsageError{Err: err}
		}
		return c, err
	}
	if c == nil {
		if args[1] != "" {
			return cli.root, Usagef("unknown command %q", args[1])
		}
		cli.help(cli.root, nil)
		return nil, nil
	}
	run := RunFunc(c.Run)
	for i := len(cli.middlewares) - 1; i >= 0; i-- {
		run = cli.middlewares[i](c, run)
	}
	return c, run(ctx)
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

// getSubCommand resolves the command of args. The arg fields of the parent
// commands come before the ones of the sub command, all set after its name.
func (cli *CLI) getSubCommand(command SubCommands, args []string) (Command, error) {
	cli.lastCommandsName = []string{}
	for name, c := range command.SubCommands() {
		cli.lastCommandsName = append(cli.lastCommandsName, name)
		if name == args[0] {
			if err := cli.defineCommand(cli.flagSet, c, &cli.arguments); err != nil {
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
			if err := cli.arguments.set(cli.flagSet.Args()); err != nil {
				return c, err
			}
			if p, ok := c.(ParseHelper); ok {
				if err := p.Parse(cli.flagSet.Args()); err != nil {
					return c, err
				}
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
			flags, _ := cli.usage(c)
			return flags
		},
		"argList": func(c Command) string {
			_, args := cli.usage(c)
			return args
		},
	}).Parse(defaultHelpTemplate)
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
		for k := range subC {
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

// usage returns the flag and the positional argument help of c.
func (cli *CLI) usage(c Command) (flags, args string) {
	fs := &flag.FlagSet{
		Usage: func() {},
	}
	var out bytes.Buffer
	fs.SetOutput(&out)
	var arguments arguments
	if err := cli.defineCommand(fs, c, &arguments); err != nil {
		return err.Error(), ""
	}
	fs.PrintDefaults()
	return out.String(), arguments.String()
}

// defineCommand defines the flag fields of c on fs and appends its arg fields
// to args.
func (cli *CLI) defineCommand(fs Flagger, c Command, args *arguments) error {
	st := reflect.ValueOf(c)
	if st.Kind() != reflect.Pointer {
		return fmt.Errorf("%T: pointer expected", c)
	}
	if err := cli.defineFlagSet(fs, st, "", args); err != nil {
		return fmt.Errorf("%T: %w", c, err)
	}
	return nil
}

func (cli *CLI) defineFlagSet(fs Flagger, st reflect.Value, subName string, args *arguments) error {
	st = reflect.Indirect(st)
	if !st.IsValid() || st.Type().Kind() != reflect.Struct {
		return errors.New("non-nil pointer for struct expected")
	}
	for i := 0; i < st.NumField(); i++ {
		typ := st.Type().Field(i)
		var name, usage string
		tag := typ.Tag.Get("flag")
		val := st.Field(i)
		if !val.CanInterface() {
			// field is unexported
			continue
		}
		if argTag := typ.Tag.Get("arg"); argTag != "" {
			if err := args.add(val, argTag, typ.Name); err != nil {
				return err
			}
			continue
		}
		if tag == "" {
			switch typ.Type.Kind() {
			case reflect.Struct:
				if err := cli.defineFlagSet(fs, val, "", args); err != nil {
					return err
				}
			case reflect.Pointer:
				if reflect.ValueOf(val).Kind() == reflect.Struct {
					if err := cli.defineFlagSet(fs, val, "", args); err != nil {
						return err
					}
				}
			}
			continue
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
		if typ.Type.Kind() == reflect.Struct && !val.Addr().Type().Implements(flagValueType) {
			if err := cli.defineFlagSet(fs, val, name, args); err != nil {
				return err
			}
			continue
		}
		if err := bindValue(fs, val, name, usage); err != nil {
			return fmt.Errorf("flag %q (field %s, type %s): %w", name, typ.Name, typ.Type, err)
		}
	}
	return nil
}

var (
	flagValueType      = reflect.TypeOf((*flag.Value)(nil)).Elem()
	errUnsupportedType = errors.New("unsupported type")
)

// bindValue defines val on fs as the flag name. It is shared by the flag and
// the arg fields.
func bindValue(fs Flagger, val reflect.Value, name, usage string) error {
	addr := val.Addr()
	if v, ok := addr.Interface().(flag.Value); ok {
		fs.Var(v, name, usage)
		return nil
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
		return errUnsupportedType
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
