package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
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
	// RenameWarningTypeFileDoesNotExist is used to report the fact that the original
	// file raf has been asked to rename does not exist in the file system. The Value
	// property of the RenameWarning will be populated with the original file name
	RenameWarningTypeFileDoesNotExist
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
// The output RenameLog file can be passed to the Apply() function to perform the changes.
func RenameAllFiles(p []Prop, tokens TokenStream, files []string, opts Opts) (RenameLog, error) {
	rlog := make([]RenameLogEntry, len(files))
	collisions := make(map[string][]int)
	for idx, f := range files {
		_, err := filepath.Abs(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not determine absolute path for %s: %s\n", f, err)
			return rlog[:idx], err
		}
		//baseDir := filepath.Dir(absPath)

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

// Undo looks for a rename log file in the given folder and reverses the change to the files listed in the log.
// Returns a flipped RenameLog that can be passed to the Apply() function
func Undo(cwd string, opts Opts) (RenameLog, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	rafPath, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !rafPath.IsDir() {
		return nil, fmt.Errorf("%s is not a valid directory. The undo command receives the path to a directory containing a %s file", rafPath, rafStatusFile)
	}
	rafFilePath := abs + string(os.PathSeparator) + rafStatusFile
	if _, err = os.Stat(rafFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("The directory %s does not contain a valid raf status file (%s)", abs, rafStatusFile)
	}
	rlog, err := ReadRenameLog(rafFilePath)
	if err != nil {
		return nil, err
	}
	if len(rlog) == 0 {
		return nil, fmt.Errorf("The raf status file %s does not contain any log entries", rafFilePath)
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Beginning raf undo in folder %s", abs)
	}

	flipRlog := make([]RenameLogEntry, len(rlog))
	collisions := make(map[string][]int)
	for idx, entry := range rlog {
		warnings := make([]RenameWarning, 0)
		curFilePath := abs + string(os.PathSeparator) + entry.NewFileName
		newFilePath := abs + string(os.PathSeparator) + entry.OriginalFileName
		// new file must exists
		if _, err = os.Stat(curFilePath); os.IsNotExist(err) {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "WARNING: File %s from raf log not found", entry.NewFileName)
			}
			warnings = append(warnings, RenameWarning{
				Type:  RenameWarningTypeFileDoesNotExist,
				Value: entry.NewFileName,
			})
			continue
		}
		// original file must not
		if _, err = os.Stat(newFilePath); !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARNING: Another file is already using the name %s preventing raf from resting %s to its original name", entry.OriginalFileName, entry.NewFileName)
			continue
		}

		c, ok := collisions[entry.OriginalFileName]
		if !ok {
			collisions[entry.OriginalFileName] = make([]int, 1)
			collisions[entry.OriginalFileName][0] = idx
		} else {
			collisions[entry.OriginalFileName] = append(c, idx)
		}

		flipRlog[idx] = RenameLogEntry{
			OriginalFileName: entry.NewFileName,
			NewFileName:      entry.OriginalFileName,
			Warnings:         warnings,
		}
	}

	// populate collisions
	// TODO: Validate output file name
	for _, v := range collisions {
		if len(v) > 1 {
			for _, idx := range v {
				flipRlog[idx].Collisions = v
			}
		}
	}
	return flipRlog, nil
}

// Apply makes the changes outlined by the given RenameLog in the given path. Apply will not handle
// collisions or warnings in the log, the function will attempt to use the os.Rename method to
// perform the action and if an error is thrown return the os error.
func Apply(rlog RenameLog, path string, opts Opts) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	pathStat, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if !pathStat.IsDir() {
		return fmt.Errorf("The given path %s is not a directory", path)
	}

	for idx, e := range rlog {
		origFile := absPath + string(os.PathSeparator) + e.OriginalFileName
		newFile := absPath + string(os.PathSeparator) + e.NewFileName

		err = os.Rename(origFile, newFile)
		if err != nil {
			if writeErr := writeRenameLog(rlog[:idx], absPath); writeErr != nil {
				fmt.Fprintf(os.Stderr, "FATAL: Could not write rename log after rename error: %s", writeErr)
			}
			return err
		}
	}

	// write new log file
	return writeRenameLog(rlog, absPath)
}

// ReadRenameLog parses a RenameLog file at the given path and unmarshals it into a
// RenameLog object (slice of RenameLogEntry)
func ReadRenameLog(path string) (RenameLog, error) {
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

func writeRenameLog(rlog RenameLog, absPath string) error {
	statusFile := absPath + string(os.PathSeparator) + rafStatusFile

	_, err := os.Stat(statusFile)
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
