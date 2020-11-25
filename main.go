package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	cli "github.com/urfave/cli/v2"
)

const rafStatusFile = ".raf"
const rafVersion = "v0.3.0"

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
		Version:     rafVersion,
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
			{
				Name:   "man",
				Usage:  "Show man page for raf",
				Action: man,
			},
		},
	}
}

func undo(c *cli.Context) error {
	path := c.Args().First()
	opts := readOpts(c)
	rlog, err := Undo(path, opts)
	if err != nil {
		return err
	}
	return Apply(rlog, path, opts)

}

func rename(c *cli.Context) error {
	matches, err := validateMatcher(c)
	if err != nil {
		return err
	}
	props, err := validateProps(c)
	if err != nil {
		return err
	}
	out, err := validateOutput(c)
	if err != nil {
		return err
	}
	if len(props) < out.CustomVarCount {
		return fmt.Errorf("The command declared %d properties but uses %d in the output formatter\n", len(props), out.CustomVarCount)
	}
	opts := readOpts(c)
	rlog, err := RenameAllFiles(props, out.Tokens, matches, opts)
	if err != nil {
		return err
	}
	path := filepath.Dir(matches[0])
	if !opts.DryRun {
		return Apply(rlog, path, opts)
	}
	if opts.DryRun {
		for _, e := range rlog {
			dryRunPrint(e.OriginalFileName, e.NewFileName)
		}
		fmt.Fprintln(os.Stderr)

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
				collisionLog := fmt.Sprintf("[ERROR] File \"%s\" would be renamed to \"%s\" and would collide with: ", e.OriginalFileName, e.NewFileName)
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

	if writeTestRLog {
		err = writeRenameLog(rlog, path)
		if err != nil {
			return fmt.Errorf("Could not write rename log %v\n", err)

		}
	}
	return nil
}

func man(c *cli.Context) error {
	manUri := "https://raw.githubusercontent.com/sapessi/raf/" + rafVersion + "/raf.1"
	manWebMessage := fmt.Sprintf("The latest version of the documentation is available in the man page at %s", manUri)
	_, manErr := exec.LookPath("man")
	usr, userErr := user.Current()
	if runtime.GOOS == "windows" || manErr != nil || userErr != nil {
		fmt.Println(manWebMessage)
		return nil
	}

	// crete folder for raf man page
	rafHomeDir := usr.HomeDir + string(os.PathSeparator) + ".raf"
	rafManPath := rafHomeDir + string(os.PathSeparator) + "raf.1"
	if _, err := os.Stat(rafManPath); os.IsNotExist(err) {
		if os.Mkdir(rafHomeDir, 0700) != nil {
			fmt.Println(manWebMessage)
			return err
		}
		// download man page from url
		// Get the data
		resp, err := http.Get(manUri)
		if err != nil {
			fmt.Println(manWebMessage)
			return err
		}

		// Create the file
		out, err := os.Create(rafManPath)
		if err != nil {
			fmt.Println(manWebMessage)
			return err
		}

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Println(manWebMessage)
			return err
		}
		resp.Body.Close()
		out.Close()
	}

	cmd := exec.Command("man", rafManPath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(manWebMessage)
		return err
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

func dryRunPrint(from, to string) {
	red := color.New(color.FgHiRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("File %s -> %s\n", red(from), green(to))
}
