package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	cli "github.com/urfave/cli/v2"
)

const rafStatusFile = ".raf"

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
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print raf version",
	}
	app := &cli.App{
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
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func undo(c *cli.Context) error {
	opts := readOpts(c)
	cwd := c.Args().First()
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	rafPath, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !rafPath.IsDir() {
		return fmt.Errorf("%s is not a valid directory. The undo command receives the path to a directory containing a %s file", rafPath, rafStatusFile)
	}
	rafFilePath := abs + string(os.PathSeparator) + rafStatusFile
	if _, err = os.Stat(rafFilePath); os.IsNotExist(err) {
		return fmt.Errorf("The directory %s does not contain a valid raf status file (%s)", abs, rafStatusFile)
	}
	rlog, err := readRenameLog(rafFilePath)
	if err != nil {
		return err
	}
	if len(rlog) == 0 {
		return fmt.Errorf("The raf status file %s does not contain any log entries", rafFilePath)
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Beginning raf undo in folder %s", abs)
	}

	for _, entry := range rlog {
		curFilePath := abs + string(os.PathSeparator) + entry.NewFileName
		newFilePath := abs + string(os.PathSeparator) + entry.OriginalFileName
		// new file must exists
		if _, err = os.Stat(curFilePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARNING: File %s from raf log not found", entry.NewFileName)
			continue
		}
		// original file must not
		if _, err = os.Stat(newFilePath); !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARNING: Another file is already using the name %s preventing raf from resting %s to its original name", entry.OriginalFileName, entry.NewFileName)
			continue
		}

		if !opts.DryRun {
			err = os.Rename(curFilePath, newFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: Could not rename file %s to %s: %v", curFilePath, newFilePath, err)
			}
			fmt.Println(entry.OriginalFileName)
		} else {
			fmt.Printf("File %s -> %s\n", entry.NewFileName, entry.OriginalFileName)
		}
	}

	err = os.Remove(rafFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not remove raf status file %s. This is not a critical issue since the file will be overwritten automatically if raf is executed again in this folder", rafFilePath)
	}
	return nil
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
	if !opts.DryRun {
		err = writeRenameLog(rlog, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write rename log %v\n", err)
			os.Exit(1)
		}
	}
	return nil
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
