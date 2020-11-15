package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	cli "github.com/urfave/cli/v2"
)

var ReservedVarNames = map[string]int8{"$cnt": 0}

type Output struct {
	Raw            string
	VarCount       int
	CustomVarCount int
	Tokens         TokenStream
}

type Opts struct {
	DryRun  bool
	Verbose bool
}

func main() {
	app := &cli.App{
		Name:        "raf",
		Usage:       "Rename All Files",
		Description: "Rename all selected files based on the set of rules passed as options",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "prop",
				Aliases:     []string{"p"},
				Usage:       "Extract a portion of the file name and assign it to a varible",
				DefaultText: "title=/\\ \\-([A-Za-z0-9\\ ]+)\\_/",
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Define the output file name using string substitution",
				DefaultText: "\"MyVidew - $cnt - $title\"",
				Required:    true,
			},
			&cli.BoolFlag{
				Name:    "dryrun",
				Aliases: []string{"d"},
				Usage:   "Runs the command in dry run mode. When in dry run mode the log output is sent to stderr and the changed file names are sent to stdout, the files are not actually renamed",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Prints verbose output",
			},
		},
		Action: rename,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func rename(c *cli.Context) error {
	matches, err := validateMatcher(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	props, err := validateProps(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out, err := validateOutput(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(props) < out.CustomVarCount {
		fmt.Printf("The command declared %d properties but uses %d in the output formatter\n", len(props), out.CustomVarCount)
		os.Exit(1)
	}
	opts := Opts{
		DryRun:  c.Bool("dryrun"),
		Verbose: c.Bool("verbose"),
	}
	err = RenameAllFiles(props, out, matches, opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return nil
}

func validateMatcher(c *cli.Context) ([]string, error) {
	files := c.Args().Slice()
	if files == nil || len(files) == 0 {
		return nil, errors.New("No input files")
	}

	return files, nil
}

func validateProps(c *cli.Context) ([]Prop, error) {
	args := c.StringSlice("prop")
	props := make([]Prop, len(args))

	for idx, v := range args {
		prop, err := ParseProp(v)
		if err != nil {
			return nil, err
		}
		props[idx] = prop
	}

	return props, nil
}

func validateOutput(c *cli.Context) (*Output, error) {
	rawOutput := c.String("output")
	if rawOutput == "" {
		return nil, errors.New("Output formatter must be a valid string and cannot be empty")
	}

	tokens, err := ParseOutput(rawOutput)
	if err != nil {
		return nil, err
	}

	varCount := 0
	customVarCount := 0
	for _, t := range tokens {
		if t.Type != TokenTypeProperty {
			continue
		}
		varCount++
		if _, ok := ReservedVarNames[t.Value]; !ok {
			customVarCount++
		}
	}

	return &Output{
		Raw:            rawOutput,
		VarCount:       varCount,
		CustomVarCount: customVarCount,
		Tokens:         tokens,
	}, nil
}
