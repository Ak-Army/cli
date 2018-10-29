# cli
cli is a simple, fast package for building command line apps in Go. It's a wrapper around the "flag" package.

# Example usage
Declare a struct type that embeds *cli.Flagger, along with an fields you want to capture as flags.
```
type Echo struct {
	*cli.Flagger
    Echoed string `flag:"echoed, echo this string"`
}
```
Now we need to make our type implement the cli.Command interface. That requires three methods that aren't already provided by *cli.Flagger:
```
func (c *Echo) Desc() string {
	return "Echo the input string."
}
func (c *Echo) Run() {
	fmt.Println(c.Echoed)
}
```
Maybe we write sample command runs:
```
func (c *Echo) Samples() []string {
	return []string{"echoprogram -echoed=\"echo this\"",
	"echoprogram -echoed=\"or echo this\""}
}
```


