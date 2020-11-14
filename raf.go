package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RenameAllFile(p []Prop, out *Output, files []string, opts Opts) error {
	for idx, f := range files {
		cnt := idx + 1
		fileName := filepath.Base(f)

		varValues := make(map[string]string)
		for _, prop := range p {
			matches := prop.Regex.FindAllString(fileName, -1)
			if matches == nil {
				logOut(fmt.Sprintf("WARNING: the matcher %s does not match any string on file %s", prop.Matcher, fileName), opts)
			}
			if len(matches) > 1 {
				logOut(fmt.Sprintf("WARNING: the matcher %s matches multiple parts on file %s, only the leftmost is available", prop.Matcher, fileName), opts)
			}
			varValues[prop.Name] = matches[0]
		}

		outName := out.Raw
		for _, v := range out.Vars {
			if _, ok := ReservedVarNames[v]; ok {
				switch v {
				case "$cnt":
					outName = strings.ReplaceAll(outName, "$cnt", strconv.Itoa(cnt))
				}
				continue
			}
			outName = strings.ReplaceAll(outName, v, varValues[v])
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

func logOut(str string, opt Opts) {
	if opt.DryRun {
		fmt.Fprintln(os.Stderr, str)
		return
	}
	fmt.Println(str)
}
