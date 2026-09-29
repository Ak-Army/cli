package cli

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
)

type argField struct {
	name     string
	usage    string
	val      reflect.Value
	variadic bool
}

type arguments []argField

func (a *arguments) add(val reflect.Value, tag, fieldName string) error {
	name, usage, _ := strings.Cut(tag, ",")
	f := argField{name: strings.TrimSpace(name), usage: strings.TrimSpace(usage), val: val}
	if f.name == "-" {
		return nil
	}
	if n := len(*a); n > 0 && (*a)[n-1].variadic {
		return fmt.Errorf("variadic argument %q must be the last one", (*a)[n-1].name)
	}
	f.variadic = val.Kind() == reflect.Slice && !val.Addr().Type().Implements(flagValueType)
	if _, err := f.bind(flag.NewFlagSet("", flag.ContinueOnError)); err != nil {
		return fmt.Errorf("argument %q (field %s, type %s): %w", f.name, fieldName, val.Type(), err)
	}
	*a = append(*a, f)
	return nil
}

func (a *arguments) set(args []string) error {
	for _, f := range *a {
		if f.variadic {
			f.val.Set(reflect.MakeSlice(f.val.Type(), 0, len(args)))
			for _, s := range args {
				if err := f.set(s); err != nil {
					return err
				}
			}
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("missing argument: %s", f.name)
		}
		if err := f.set(args[0]); err != nil {
			return err
		}
		args = args[1:]
	}
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument: %s", args[0])
	}
	return nil
}

func (a *arguments) String() string {
	var b strings.Builder
	for _, f := range *a {
		name := f.name
		if f.variadic {
			name += "..."
		}
		fmt.Fprintf(&b, "  %s\n    \t%s\n", name, f.usage)
	}
	return b.String()
}

func (f argField) bind(fs Flagger) (reflect.Value, error) {
	v := f.val
	if f.variadic {
		v = reflect.New(v.Type().Elem()).Elem()
	}
	return v, bindValue(fs, v, f.name, f.usage)
}

func (f argField) set(s string) error {
	fs := flag.NewFlagSet(f.name, flag.ContinueOnError)
	v, err := f.bind(fs)
	if err == nil {
		err = fs.Set(f.name, s)
	}
	if err != nil {
		return fmt.Errorf("invalid value %q for argument %s: %w", s, f.name, err)
	}
	if f.variadic {
		f.val.Set(reflect.Append(f.val, v))
	}
	return nil
}
