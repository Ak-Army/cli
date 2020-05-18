package cli

import (
	"context"
	"strings"
)

type Root struct {
	Name        string
	Version     string
	Description string
	Authors     []string
	subCommands map[string]Command
}

var defaultRoot = &Root{
	subCommands: make(map[string]Command),
}

// RootCommand returns the default Root command
func RootCommand() *Root {
	return defaultRoot
}

// SetRoot should set default root command
func SetRoot(root *Root) {
	defaultRoot = root
}

// AddCommand add a main command
func (r *Root) AddCommand(name string, command Command) bool {
	if _, ok := r.subCommands[name]; ok {
		return false
	}
	r.subCommands[name] = command
	return true
}

// SubCommands return the main commands
func (r *Root) SubCommands() map[string]Command {
	return r.subCommands
}

func (r *Root) Help() string {
	buff := strings.Builder{}
	buff.WriteString("Usage: " + r.Name)
	if len(r.SubCommands()) > 0 {
		buff.WriteString(" command [command options]")
	}
	buff.WriteString("\n")
	if r.Version != "" {
		buff.WriteString("Version: " + r.Version + "\n")
	}
	authorsLen := len(r.Authors)
	if authorsLen > 0 {
		if authorsLen > 1 {
			buff.WriteString("Authors: ")
		} else {
			buff.WriteString("Author: ")
		}
		buff.WriteString(strings.Join(r.Authors, ", ") + "\n")
	}
	if r.Description != "" {
		buff.WriteString("Description:\n" + r.Description + "\n")
	}
	return buff.String()
}

func (r *Root) Synopsis() string {
	buff := strings.Builder{}
	buff.WriteString(r.Name)
	if len(r.SubCommands()) > 0 {
		buff.WriteString(" command [command options]")
	}
	buff.WriteString("\n")
	return buff.String()
}

func (r *Root) Run(_ context.Context) error {
	return nil
}
