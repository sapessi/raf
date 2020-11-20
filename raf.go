package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

// VarValues stores the values parsed from the original name of the file based on the
// Prop configured.
type VarValues = map[string]string

// RenameLogEntry records an operation performed in a file and can be used to undo
// the rename
type RenameLogEntry struct {
	OriginalFileName string
	// OriginalFileChecksum is not used at the moment, we may enable it through a separate
	// options since it could have a significant impact on performance
	OriginalFileChecksum string
	NewFileName          string
}

// RenameLog is a slice of RenameLogEntry objects that record all of the opertaions
// performed by the RenameAllFiles function
type RenameLog = []RenameLogEntry

// RenameAllFiles iterates over the files passed as input and for each one, extracts the
// property values, populates the intrinsic properties, and calls the GenerateName function.
//
// If the DryRun property of the Opts object is set to true the files are not actually
// renamed and the output, generated name is printed out
func RenameAllFiles(p []Prop, tokens TokenStream, files []string, opts Opts) (RenameLog, error) {
	rlog := make([]RenameLogEntry, len(files))
	for idx, f := range files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not determine absolute path for %s: %s\n", f, err)
			return rlog[:idx], err
		}
		baseDir := filepath.Dir(absPath)

		fileName := filepath.Base(f)
		state := renamerState{
			idx:       idx,
			fileName:  fileName,
			extension: filepath.Ext(fileName),
		}

		varValues := extractVarValues(fileName, p, opts)
		for k, v := range ReservedVarNames {
			varValues[k] = v(state)
		}

		outName, err := GenerateName(varValues, tokens, state, opts)
		if err != nil {
			return rlog[:idx], err
		}

		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Renaming \"%s\" to \"%s\"\n", fileName, outName)
		}

		if !opts.DryRun {
			outPath := baseDir + string(os.PathSeparator) + outName
			if _, err := os.Stat(outPath); err == nil {
				return rlog[:idx], fmt.Errorf("The file %s already exists", outPath)
			}
			err := os.Rename(baseDir+string(os.PathSeparator)+fileName, baseDir+string(os.PathSeparator)+outName)
			if err != nil {
				return rlog[:idx], fmt.Errorf("Error while renaming %s to %s: %v", fileName, outName, err)
			}
			rlog[idx] = RenameLogEntry{
				OriginalFileName: fileName,
				NewFileName:      outName,
			}
			fmt.Println(outName)
		} else {
			dryRunPrint(fileName, outName)
		}
	}
	return rlog, nil
}

// GenerateName uses the variable values to generate a string based on the input TokenStream
func GenerateName(varValues VarValues, out TokenStream, rstate renamerState, opts Opts) (string, error) {
	outName := ""
	for _, t := range out {
		if t.Type == TokenTypeLiteral {
			outName += t.Value
			continue
		}
		if t.Type == TokenTypeProperty {

			propValue, ok := varValues[t.Value]
			if !ok {
				fmt.Fprintf(os.Stderr, "WARNING: Output asks for value %s that is not declared as a property\n", t.Value)
				continue
			}

			if propValue == "" && opts.Verbose {
				fmt.Fprintf(os.Stderr, "WARNING: Value for property %s is empty\n", t.Value)
			}

			if t.Formatter != nil {
				formattedValue := propValue
				for _, f := range t.Formatter {
					fout, err := f.Format(formattedValue, rstate)
					if err != nil {
						return "", err
					}
					formattedValue = fout
				}
				outName += formattedValue
			} else {
				outName += propValue
			}
		}
	}
	return outName, nil
}

func dryRunPrint(from, to string) {
	red := color.New(color.FgHiRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("File %s -> %s\n", red(from), green(to))
}

type renamerState struct {
	idx       int
	fileName  string
	extension string
}

func extractVarValues(fname string, p []Prop, opts Opts) VarValues {
	varValues := make(map[string]string)
	for _, prop := range p {
		matches := prop.Regex.FindAllStringSubmatch(fname, -1) //prop.Regex.FindAllString(fname, -1)
		if matches == nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "WARNING: the matcher %s does not match any string on file %s\n", prop.Matcher, fname)
			}
			varValues["$"+prop.Name] = ""
			continue
		}
		if len(matches) > 1 && opts.Verbose {
			fmt.Fprintf(os.Stderr, "WARNING: the matcher %s matches multiple parts on file %s, only the leftmost is available\n", prop.Matcher, fname)
		}
		varValues["$"+prop.Name] = matches[0][len(matches[0])-1]
	}

	return varValues
}
