package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	cli "github.com/urfave/cli/v2"
)

const rafStatusFile = ".raf"

// TODO: this is an ugly hack for the unit tests. We should formalize this
// as a parameter
var writeTestRLog = false

type output struct {
	Raw            string
	VarCount       int
	CustomVarCount int
	Tokens         TokenStream
}

// Opts contains the global settings for the library and it is passed to nearly
// all methods.
type Opts struct {
	DryRun  bool
	Verbose bool
}

func main() {
	app := getApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func undo(c *cli.Context) error {
	path := c.Args().First()
	opts := readOpts(c)
	return Undo(path, opts)
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
	opts := readOpts(c)
	rlog, err := RenameAllFiles(props, out.Tokens, matches, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !opts.DryRun || writeTestRLog {
		err = writeRenameLog(rlog, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write rename log %v\n", err)
			os.Exit(1)
		}
	}
	if opts.DryRun {
		// print out warnings
		yellow := color.New(color.FgYellow).SprintFunc()
		for _, e := range rlog {
			if e.Warnings != nil && len(e.Warnings) > 0 {
				for _, w := range e.Warnings {
					fmt.Fprintln(os.Stderr, yellow(w.String(e)))
				}
			}
		}
		// print out collisions
		printed := make([]bool, len(rlog))
		red := color.New(color.FgHiRed).SprintFunc()
		for logidx, e := range rlog {
			if e.Collisions != nil && !printed[logidx] && len(e.Collisions) > 0 {
				collisionLog := fmt.Sprintf("[ERROR] File %s would be rename to %s and would collide with: ", e.OriginalFileName, e.NewFileName)
				otherNames := make([]string, len(e.Collisions)-1) // -1 because it always includes itself
				for cidx, c := range e.Collisions {
					if c != logidx {
						otherNames[cidx-1] = rlog[c].OriginalFileName
						printed[c] = true
					}
				}
				collisionLog += strings.Join(otherNames, ", ")
				fmt.Fprintln(os.Stderr, red(collisionLog))
				printed[logidx] = true
			}
		}
	}
	return nil
}

func getApp() *cli.App {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print raf version",
	}
	return &cli.App{
		Name:        "raf",
		Usage:       "raf -p \"title=Video\\ \\d+\\ \\-\\ ([A-Za-z0-9\\ ]+)_\" -d -o 'UnionStudio - $cnt - $title.mkv' *",
		Description: cliDescription,
		Version:     "v0.1",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "prop",
				Aliases: []string{"p"},
				//Usage:   "-p \"title=Video\\ \\d+\\ \\-\\ ([A-Za-z0-9\\ ]+)_\"",
				Usage: propFlagDescription,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   outputFlagDescription,
			},
			&cli.BoolFlag{
				Name:    "dryrun",
				Aliases: []string{"d"},
				Usage:   dryRunFlagDescription,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Prints verbose output",
			},
		},
		Action: rename,
		Commands: []*cli.Command{
			{
				Name:   "undo",
				Usage:  undoCommandDescription,
				Action: undo,
			},
		},
	}
}

func readOpts(c *cli.Context) Opts {
	return Opts{
		DryRun:  c.Bool("dryrun"),
		Verbose: c.Bool("verbose"),
	}
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

func validateOutput(c *cli.Context) (*output, error) {
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

	return &output{
		Raw:            rawOutput,
		VarCount:       varCount,
		CustomVarCount: customVarCount,
		Tokens:         tokens,
	}, nil
}

func writeRenameLog(rlog RenameLog, c *cli.Context) error {
	// get folder
	absPath, err := filepath.Abs(c.Args().First())
	if err != nil {
		return err
	}
	statusFile := filepath.Dir(absPath) + string(os.PathSeparator) + rafStatusFile

	_, err = os.Stat(statusFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		err = os.Remove(statusFile)
		if err != nil {
			return err
		}

	}
	statusFileWriter, err := os.Create(statusFile)
	if err != nil {
		return err
	}
	defer statusFileWriter.Close()
	gobEncoder := gob.NewEncoder(statusFileWriter)
	err = gobEncoder.Encode(rlog)
	if err != nil {
		return err
	}
	return nil
}

func readRenameLog(path string) (RenameLog, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decoder := gob.NewDecoder(reader)
	var rlog RenameLog
	err = decoder.Decode(&rlog)
	if err != nil {
		return nil, err
	}
	return rlog, nil
}
