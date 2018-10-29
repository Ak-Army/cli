package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
	"time"
)

type Cli struct {
	Name           string
	Version        string
	Description    string
	Authors        []string
	Commands       map[string]Command
	commandSets    map[string]Command
	defaultCommand string
	template       *template.Template
}

func NewCli(name string, version string) *Cli {
	cli := &Cli{
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

func (cli *Cli) SetTemplate(temp string) error {
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

func (cli *Cli) Add(commands ...Command) {
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

func (cli *Cli) SetDefaults(command string) {
	cli.defaultCommand = command
}

func (cli *Cli) Run(args []string) {
	command := cli.defaultCommand
	if len(args) > 1 {
		command = args[1]
	}
	for name, c := range cli.commandSets {
		if name == command {
			cli.defineFlagSet(c)
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
				cli.Help(command)
				return
			}
			c.Run()
			return
		}
	}
	cli.Help("")
}

func (cli *Cli) Help(commandName string) {
	if commandName != "" {
		if command, ok := cli.commandSets[commandName]; ok {
			cli.Commands[commandName] = command
			cli.template.Execute(os.Stderr, cli)
		}
	} else {
		for _, command := range cli.commandSets {
			cli.defineFlagSet(command)
		}
		cli.Commands = cli.commandSets
		cli.template.Execute(os.Stderr, cli)
	}
}

func (cli *Cli) defineFlagSet(command Command) error {
	fs := command.GetFlagSet()
	st := reflect.ValueOf(command)
	if st.Kind() != reflect.Ptr {
		return errors.New("pointer expected")
	}
	st = reflect.Indirect(st)
	if !st.IsValid() || st.Type().Kind() != reflect.Struct {
		return errors.New("non-nil pointer to struct expected")
	}
	flagValueType := reflect.TypeOf((*flag.Value)(nil)).Elem()
	for i := 0; i < st.NumField(); i++ {
		typ := st.Type().Field(i)
		var name, usage string
		tag := typ.Tag.Get("flag")
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
