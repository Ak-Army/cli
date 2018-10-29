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
Package understands all basic types supported by flag's package xxxVar functions: int, int64, uint, uint64, float64, bool, string, time.Duration. Types implementing flag.Value interface are also supported.
```type CustomDate string
func (c *CustomDate) String() string {
	return fmt.Sprint(*c)
}
func (c *CustomDate) Set(value string) error {
	dateRegex := `^20\d{2}(\/|-)(0[1-9]|1[0-2])(\/|-)(0[1-9]|[12][0-9]|3[01])$`
	if ok, err := regexp.MatchString(dateRegex, value); err != nil || !ok {
		return errors.New("from parameter is not a valid date")
	}
	*c = CustomDate(value)
	return nil
}
type EchoWithDate struct {
    *cli.Flagger
    Echoed string `flag:"echoed, echo this string"`
    EchoWithDate CustomDate `flag:"echoDate, echo this date too"`
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
We can set default command to run
```
c.SetDefaults("echo")
```
After all of this, we can run them like this:
```
func main() {
	c := cli.NewCli("echoer", "1.0.0")
	c.Authors = []string{"authors goes here"}
	c.Add(
		&Echo{
			Echoed: "default string",
		})
	//c.SetDefaults("echo")
	c.Run(os.Args)
}

```

