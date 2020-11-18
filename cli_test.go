// This file contains integration tests for the entire CLI as well as utility methods
// to generate tempoary files for the tests
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	make := exec.Command("go", "build", "-o", "./raf")
	out, err := make.CombinedOutput()
	if err != nil {
		fmt.Printf("could not make binary for raf: %v\n%s", err, string(out))
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSimple(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "Home", "Chapel", "Church", "Reception", "Party")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9]+)\\.mkv", "--output", "test - $cnt - $title.avi"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.Nil(t, err)

	files, err := testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(files))
	assert.Equal(t, "test - 2 - Chapel.avi", files[1])
}

type integTestContext struct {
	filesDir string
	files    []string
	ctx      *testing.T
}

func createIntegTestContext(t *testing.T) (*integTestContext, error) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), t.Name())
	if err != nil {
		return nil, err
	}
	return &integTestContext{
		filesDir: tmpDir,
		files:    make([]string, 0),
		ctx:      t,
	}, nil
}

func (t *integTestContext) CreateFile(name string) error {
	fpath := t.filesDir + string(os.PathSeparator) + name
	file, err := os.Create(fpath)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		t.ctx.Logf("WARNING: could not close temporary file %s: %v", fpath, err)
	}
	t.files = append(t.files, name)
	return nil
}

func (t *integTestContext) ListFilesInWorkingDir(includeStatusFile, useFullPath bool) ([]string, error) {
	files := make([]string, 0)
	filepath.Walk(t.filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.ctx.Log(err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".raf") && !includeStatusFile {
			return nil
		}

		if useFullPath {
			files = append(files, path)
		} else {
			files = append(files, filepath.Base(path))
		}

		return nil
	})
	return files, nil
}

func (t *integTestContext) Files(useFullPath bool) []string {
	if !useFullPath {
		return t.files
	}
	files := make([]string, len(t.files))
	for idx, f := range t.files {
		files[idx] = t.filesDir + string(os.PathSeparator) + f
	}
	return files
}

func (t *integTestContext) CreateFiles(pattern string, titles ...string) error {
	parser := newParser(pattern)
	tokens, err := parser.parse()
	if err != nil {
		return err
	}
	for idx, str := range titles {
		fname := ""
		hasIdx := false
		for _, t := range tokens {
			if t.Type == TokenTypeLiteral {
				fname += t.Value
			}
			if t.Type == TokenTypeProperty {
				if f, ok := ReservedVarNames[t.Value]; ok {
					rstate := renamerState{
						idx:       idx + 1,
						fileName:  "",
						extension: "",
					}
					fname += f(rstate)

					if t.Value == "$cnt" {
						hasIdx = true
					}
				} else {
					fname += str
				}
			}
		}
		if !hasIdx {
			fname = strconv.Itoa(idx) + " - " + fname
		}
		err = t.CreateFile(fname)
		if err != nil {
			return err
		}
	}
	return nil
}
