package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func RenameAllFiles(p []Prop, out *Output, files []string, opts Opts) error {
	for idx, f := range files {
		cnt := idx + 1
		fileName := filepath.Base(f)

		varValues := extractVarValues(fileName, p, opts)
		varValues["$cnt"] = strconv.Itoa(cnt)

		outName, err := renameString(fileName, varValues, out, opts)
		if err != nil {
			return err
		}

		logOut(fmt.Sprintf("Renaming \"%s\" to \"%s\"", fileName, outName), opts)

		if !opts.DryRun {
			err := os.Rename(fileName, outName)
			if err != nil {
				return fmt.Errorf("Error while renaming %s to %s: %v", fileName, outName, err)
			}
		} else {
			fmt.Println(outName)
		}
	}
	return nil
}

func renameString(from string, varValues map[string]string, out *Output, opts Opts) (string, error) {
	outName := ""
	for _, t := range out.Tokens {
		if t.Type == TokenTypeLiteral {
			outName += t.Value
			continue
		}
		if t.Type == TokenTypeProperty {

			propValue, ok := varValues[t.Value]
			if !ok {
				logOut(fmt.Sprintf("WARNING: Output asks for value %s that is not declared as a property", t.Value), opts)
				continue
			}

			if propValue == "" {
				logOut(fmt.Sprintf("WARNING: Value for property %s is empty", t.Value), opts)
			}
			formattedValue, err := formatValue(propValue, t.Formatter)
			if err != nil {
				return "", err // we treat a formatting error as fatal
			}
			outName += formattedValue
		}
	}
	return outName, nil
}

func extractVarValues(fname string, p []Prop, opts Opts) map[string]string {
	varValues := make(map[string]string)
	for _, prop := range p {
		matches := prop.Regex.FindAllStringSubmatch(fname, -1) //prop.Regex.FindAllString(fname, -1)
		if matches == nil {
			logOut(fmt.Sprintf("WARNING: the matcher %s does not match any string on file %s", prop.Matcher, fname), opts)
			varValues["$"+prop.Name] = ""
			continue
		}
		if len(matches) > 1 {
			logOut(fmt.Sprintf("WARNING: the matcher %s matches multiple parts on file %s, only the leftmost is available", prop.Matcher, fname), opts)
		}
		varValues["$"+prop.Name] = matches[0][len(matches[0])-1]
	}

	return varValues
}

// TODO: implement formatting
func formatValue(val, formatter string) (string, error) {
	return val, nil
}

func logOut(str string, opt Opts) {
	if opt.DryRun {
		fmt.Fprintln(os.Stderr, str)
		return
	}
	fmt.Println(str)
}
