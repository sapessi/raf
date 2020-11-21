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

const (
	// RenameWarningTypePropertyValueEmpty is used when the renamer could not extract the
	// value for the property from the original file name. The Value property of the
	// RenameWarning will be populated with the name of the requested property.
	RenameWarningTypePropertyValueEmpty = iota
	// RenameWarningtypePropertyMissing is used when the property requested by the output
	// name generator was not declared as a property to be extracted from the input file
	// name or is not a valid intrinsic property. The Value property of the RenameWarning
	// wil be populated with the name of the requested property.
	RenameWarningtypePropertyMissing
)

// RenameWarning contains information about potential name generation issues. For example,
// when the output name requires a property that is either not declared or whose value is
// empty.
type RenameWarning struct {
	Type  int
	Value string
}

// String returns the warning message ready to be printed in the log/stdout
func (w *RenameWarning) String(entry RenameLogEntry) string {
	switch w.Type {
	case RenameWarningTypePropertyValueEmpty:
		return fmt.Sprintf("WARNING: Could not extract property %s from original file name: %s ", w.Value, entry.OriginalFileName)
	case RenameWarningtypePropertyMissing:
		return fmt.Sprintf("WARNING: Output file name asks for property %s which is not delcared", w.Value)
	}
	return ""
}

// RenameLogEntry records an operation performed in a file and can be used to undo
// the rename
type RenameLogEntry struct {
	OriginalFileName string
	// OriginalFileChecksum is not used at the moment, we may enable it through a separate
	// options since it could have a significant impact on performance
	OriginalFileChecksum string
	NewFileName          string
	// Warnings lists potential issues found while renaming the file
	Warnings []RenameWarning
	// Collisions points to other entries in the log that the new generated name for this
	// entry collides with
	Collisions []int
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
	collisions := make(map[string][]int)
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

		outName, warnings, err := GenerateName(varValues, tokens, state, opts)
		if err != nil {
			return rlog[:idx], err
		}

		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Renaming \"%s\" to \"%s\"\n", fileName, outName)
		}

		c, ok := collisions[outName]
		if !ok {
			collisions[outName] = make([]int, 1)
			collisions[outName][0] = idx
		} else {
			collisions[outName] = append(c, idx)
		}

		rlog[idx] = RenameLogEntry{
			OriginalFileName: fileName,
			NewFileName:      outName,
			Warnings:         warnings,
			// we'll append the collisions at teh end, once we have a fully populated map
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

			fmt.Println(outName)
		} else {
			dryRunPrint(fileName, outName)
		}
	}

	// populate collisions
	// TODO: Validate output file name
	for _, v := range collisions {
		if len(v) > 1 {
			for _, idx := range v {
				rlog[idx].Collisions = v
			}
		}
	}
	return rlog, nil
}

// GenerateName uses the variable values to generate a string based on the input TokenStream
func GenerateName(varValues VarValues, out TokenStream, rstate renamerState, opts Opts) (string, []RenameWarning, error) {
	outName := ""
	warnings := make([]RenameWarning, 0)
	for _, t := range out {
		if t.Type == TokenTypeLiteral {
			outName += t.Value
			continue
		}
		if t.Type == TokenTypeProperty {

			propValue, ok := varValues[t.Value]
			if !ok {
				fmt.Fprintf(os.Stderr, "WARNING: Output asks for value %s that is not declared as a property\n", t.Value)
				warnings = append(warnings, RenameWarning{
					Type:  RenameWarningtypePropertyMissing,
					Value: t.Value,
				})
				continue
			}

			if propValue == "" {
				if opts.Verbose {
					fmt.Fprintf(os.Stderr, "WARNING: Value for property %s is empty\n", t.Value)
				}
				warnings = append(warnings, RenameWarning{
					Type:  RenameWarningTypePropertyValueEmpty,
					Value: t.Value,
				})
			}

			if t.Formatter != nil {
				formattedValue := propValue
				for _, f := range t.Formatter {
					fout, err := f.Format(formattedValue, rstate)
					if err != nil {
						return "", warnings, err
					}
					formattedValue = fout
				}
				outName += formattedValue
			} else {
				outName += propValue
			}
		}
	}
	return outName, warnings, nil
}

// Undo looks for a rename log file in the given folder and reverses the change to the files listed in the log
func Undo(cwd string, opts Opts) error {
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
			dryRunPrint(entry.NewFileName, entry.OriginalFileName)
		}
	}

	err = os.Remove(rafFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not remove raf status file %s. This is not a critical issue since the file will be overwritten automatically if raf is executed again in this folder", rafFilePath)
	}
	return nil
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
