# cli
cli is a simple, fast package for building command line apps in Go. It's a wrapper around the "flag" package.

# Example usage
Declare a struct type along with a fields you want to capture as flags.
```Go
type Echo struct {
    Echoed string `flag:"echoed, echo this string"`
}
```
Package understands all basic types supported by flag's package xxxVar functions: int, int64, uint, uint64, float64, bool, string, time.Duration. Types implementing flag.Value interface are also supported.
```Go
type CustomDate string
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
    Echoed string `flag:"echoed, echo this string"`
    EchoWithDate CustomDate `flag:"echoDate, echo this date too"`
}
```
Positional arguments are bound with the `arg` tag, in field order, after the flags. They support the same types
as flags (and flag.Value). The last `arg` field may be a slice (`[]string`, or `[]T` where `*T` is a flag.Value):
it takes the rest of the arguments. Missing or extra arguments are reported as errors, and the help lists them under
"Arguments:". The `arg` fields of a parent command (one with sub commands) come first, followed by the ones of the
sub command, all given after the sub command's name and flags; use `--` before an argument that starts with `-`.
```Go
type Supersede struct {
    By  int64   `flag:"by, id of the observation that replaces them"`
    Old []int64 `arg:"old-id, ids of the replaced observations"`
}
```
Now we need to make our type implement the cli.Command interfacem, which requires three methods:
```Go
func (c *Echo) Synopsis() string {
	return "Echo the input string."
}
func (c *Echo) Run() {
	fmt.Println(c.Echoed)
}
```
Maybe write sample command runs:
```Go
func (c *Echo) Help() []string {
	return []string{"echoprogram -echoed=\"echo this\"",
	"echoprogram -echoed=\"or echo this\""}
}
```
We can set default command to run
```Go
c.SetDefault("echo")
```
After all of this, we can run them like this:
```Go
func main() {
	c := cli.New("echoer", "1.0.0")
	c.Authors = []string{"authors goes here"}
	c.Add(
		&Echo{
			Echoed: "default string",
		})
	c.Run(os.Args)
}

```

# Useful packages:
* <https://github.com/sgreben/flagvar>
