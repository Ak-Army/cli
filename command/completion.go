package command

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

type Completion struct {
	UnInstall bool `flag:"uninstall, uninstall auto complete functionality"`
	Bash      bool `flag:"bash, add auto complete for bash"`
	Zsh       bool `flag:"zsh, add auto complete for zsh"`
	name      string
	binPath   string
	bashCmd   string
	zshCmd    string
	homeDir   string
}

func New(name string) *Completion {
	return &Completion{
		name:    name,
		bashCmd: "complete -C %s %s",
		zshCmd:  "complete -o nospace -C %s %s",
	}
}

func (c *Completion) Help() string {
	return `Install auto complete ability for your bash and zsh terminal.
(un)install in bash basically adds/remove from .bashrc:
    complete -C </path/to/completion/command> <command>

(un)install in zsh basically adds/remove from .zshrc:
    autoload -U +X bashcompinit && bashcompinit"
    complete -C </path/to/completion/command> <command>
`
}

func (c *Completion) Synopsis() string {
	return `Install/uninstall auto complete ability for your bash, fish or zsh terminal.`
}

func (c *Completion) Run(_ context.Context) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	c.binPath, err = filepath.Abs(bin)
	if err != nil {
		return err
	}

	u, err := user.Current()
	if err != nil {
		return err
	}
	c.homeDir = u.HomeDir

	functs := []func() error{c.installBash, c.installZsh}
	if c.UnInstall {
		functs = []func() error{c.unInstallBash, c.unInstallZsh}
	}
	for _, fn := range functs {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Completion) installBash() error {
	cmd := fmt.Sprintf(c.bashCmd, c.binPath, c.name)
	file := filepath.Join(c.homeDir, ".bashrc")
	if c.isLineExists(file, cmd) {
		return errors.New("already installed")
	}
	err := c.appendFile(file, cmd)
	if err == nil {
		fmt.Println("Bash install done")
	}
	return err
}

func (c *Completion) installZsh() error {
	cmd := fmt.Sprintf(c.zshCmd, c.binPath, c.name)
	file := filepath.Join(c.homeDir, ".zshrc")
	if c.isLineExists(file, cmd) {
		return errors.New("already installed")
	}
	cmdAutoload := "autoload -U +X bashcompinit && bashcompinit"
	if c.isLineExists(file, cmdAutoload) {
		if err := c.appendFile(".zshrc", cmdAutoload); err != nil {
			return err
		}
	}
	err := c.appendFile(file, cmd)
	if err == nil {
		fmt.Println("Zsh install done")
	}
	return err
}

func (c *Completion) unInstallBash() error {
	cmd := fmt.Sprintf(c.bashCmd, c.binPath, c.name)
	file := filepath.Join(c.homeDir, ".bashrc")
	if c.isLineExists(file, cmd) {
		return errors.New("already installed")
	}
	return c.appendFile(file, cmd)
}

func (c *Completion) unInstallZsh() error {
	cmd := fmt.Sprintf(c.zshCmd, c.binPath, c.name)
	file := filepath.Join(c.homeDir, ".zshrc")
	if c.isLineExists(file, cmd) {
		return errors.New("already installed")
	}
	return c.appendFile(file, cmd)
}

func (c *Completion) isLineExists(file string, line string) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if scanner.Text() == line {
			return true
		}
	}
	return false
}

func (c *Completion) appendFile(file string, cmd string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("\n%s\n", cmd))
	return err
}

func (c *Completion) removeFromFile(file string, cmd []string) error {
	wf, err := os.OpenFile(file+".tmp", os.O_WRONLY|os.O_CREATE, 0)
	if err != nil {
		return err
	}
	defer wf.Close()
	rf, err := os.Open(file)
	if err != nil {
		return err
	}
	defer rf.Close()
	scanner := bufio.NewScanner(rf)
	for scanner.Scan() {
		text := scanner.Text()
		found := false
		for _, s := range cmd {
			if text == s {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if _, err := wf.WriteString(fmt.Sprintf("%s\n", text)); err != nil {
			return err
		}
	}
	return os.Rename(file+".tmp", file)
}
