package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Ak-Army/cli"
)

type Archive struct {
	*cli.Flagger
	DryRun bool `flag:"dry-run, downloaded voice files are not marked as archived"`
}

func (c *Archive) Desc() string {
	return "Run  archiver"
}
func (c *Archive) Run(ctx context.Context) {
	fmt.Println("Archive na ez lefutott", c.Args(), c)
}

type CustomDate time.Time

func (c *CustomDate) String() string {
	return fmt.Sprint(*c)
}
func (c *CustomDate) Set(value string) error {
	t, err := time.Parse("2006/01/02", value)
	if err != nil {
		return err
	}
	*c = CustomDate(t)
	return nil
}

type Cdr struct {
	*cli.Flagger
	Format    string     `flag:"format, the output format: csv/tsv"`
	Output    string     `flag:"output, the output format default STDOUT"`
	Fields    string     `flag:"fields, filds wich will be exported"`
	From      CustomDate `flag:"from, download voice files from (YYYY/MM/DD)"`
	To        string     `flag:"to, download voice files until (YYYY/MM/DD)"`
	ProjectID int        `flag:"projectId, download voice files only from the given project"`
}

func (c *Cdr) Desc() string {
	return "Echo the input string."
}
func (c *Cdr) Run(ctx context.Context) {
	fmt.Println("Cdr na ez lefutott", c)
}
func (c *Cdr) Samples() []string {
	return []string{"./build/vccla cdr -fields=alma,bela,aaaa -from 2017/01/01 -to 2017/02/01",
		"./build/vccla cdr -formatcsv -from 2017/01/01 -to 2017/02/01"}
}

type Db struct {
	*cli.Flagger
	Format    string `flag:"format, the output format: csv/tsv"`
	Output    string `flag:"output, the output format default STDOUT"`
	Fields    string `flag:"fields, filds wich will be exported"`
	ProjectId int    `flag:"projectId, download voice files only from the given project"`
}

func (c *Db) Desc() string {
	return "Run  archiver"
}
func (c *Db) Run(ctx context.Context) {
	fmt.Println("db na ez lefutott", c.Args())
}
func (c *Db) Samples() []string {
	return []string{"./build/vccla cdr -fields=alma,bela,aaaa -from 2017/01/01 -to 2017/02/01",
		"./build/vccla cdr -formatcsv -from 2017/01/01 -to 2017/02/01"}
}

type Download struct {
	*cli.Flagger
	Parts     bool   `flag:"parts, download voice files parts"`
	Overwrite bool   `flag:"overwrite, overwrite the file if it already exists"`
	From      string `flag:"from, download voice files from (YYYY/MM/DD)"`
	To        string `flag:"to, download voice files until (YYYY/MM/DD)"`
	ProjectId int    `flag:"projectId, download voice files only from the given project"`
}

func (c *Download) Parse(args []string) error {
	if err := c.FlagSet.Parse(args); err != nil {
		return err
	}
	dateRegex := `^20\d{2}(\/|-)(0[1-9]|1[0-2])(\/|-)(0[1-9]|[12][0-9]|3[01])$`
	if ok, err := regexp.MatchString(dateRegex, c.From); err != nil || !ok {
		return errors.New("from parameter is not a valid date")
	}
	if c.ProjectId == 0 {
		return errors.New("projectId is required")
	}

	return nil
}
func (c *Download) Desc() string {
	return "Echo the input string."
}
func (c *Download) Run(ctx context.Context) {
	fmt.Println("na ez lefutott", c.Parsed(), c.Args())
}
func (c *Download) Samples() []string {
	return []string{"./build/vccla -download -from 2017/01/01 -to 2017/02/01"}
}

func main() {
	c := cli.New("archiver", "1.0.0")
	c.Authors = []string{"authors goes here"}
	c.Add(
		&Archive{
			DryRun: true,
		},
		&Download{
			From: time.Now().AddDate(0, 0, -1).Format("2006/01/02"),
			To:   time.Now().Format("2006/01/02"),
		},
		&Db{
			Format: "csv",
		},
		&Cdr{
			Format: "csv",
			From:   CustomDate(time.Now().AddDate(0, 0, -1)),
			To:     time.Now().Format("2006/01/02"),
		})
	c.SetDefault("archive")
	c.Run(context.Background(), os.Args)
}
