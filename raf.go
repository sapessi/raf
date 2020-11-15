package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type VarValues = map[string]string

type renamerState struct {
	idx       int
	fileName  string
	extension string
}

func RenameAllFiles(p []Prop, out *Output, files []string, opts Opts) error {
	for idx, f := range files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			logOut(fmt.Sprintf("Could not determine absolute path for %s: %s", f, err), opts)
			return err
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

		outName, err := RenameString(varValues, out.Tokens, opts)
		if err != nil {
			return err
		}

		logOut(fmt.Sprintf("Renaming \"%s\" to \"%s\"", fileName, outName), opts)

		if !opts.DryRun {
			outPath := baseDir + string(os.PathSeparator) + outName
			if _, err := os.Stat(outPath); err == nil {
				return fmt.Errorf("The file %s already exists", outPath)
			}
			err := os.Rename(baseDir+string(os.PathSeparator)+fileName, baseDir+string(os.PathSeparator)+outName)
			if err != nil {
				return fmt.Errorf("Error while renaming %s to %s: %v", fileName, outName, err)
			}
		} else {
			fmt.Println(outName)
		}
	}
	return nil
}

func RenameString(varValues VarValues, out TokenStream, opts Opts) (string, error) {
	outName := ""
	for _, t := range out {
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

func extractVarValues(fname string, p []Prop, opts Opts) VarValues {
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
