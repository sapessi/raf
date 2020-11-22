// This file contains integration tests for the entire CLI as well as utility methods
// to generate tempoary files for the tests
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	rc := m.Run()

	if rc == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.7 {
			fmt.Println("Tests passed but coverage failed at", c)
			rc = -1
		}
	}
	os.Exit(rc)
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

func TestUndo(t *testing.T) {
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

	args = []string{"raf", "undo", testCtx.filesDir}
	err = app.Run(args)
	assert.Nil(t, err)
	files, err = testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(files))
	assert.Equal(t, "[UnionVideos] Wedding - 2 - Chapel.mkv", files[1])
}

func TestCollisions(t *testing.T) {
	writeTestRLog = true
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "Home", "Chapel", "_Church", "_Reception", "Party")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9]+)\\.mkv", "--output", "test - $title.avi", "-d"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.Nil(t, err)

	log, err := testCtx.RLog()
	assert.Nil(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, 5, len(log))

	// we expect a conflict between entries 2 and 3 in the log
	churchEntry := log[2]
	assert.NotNil(t, churchEntry)
	assert.Equal(t, "test - .avi", churchEntry.NewFileName)
	// we shouldn't be able to extract the title because the name starts with _
	assert.Equal(t, 1, len(churchEntry.Warnings))
	assert.Equal(t, RenameWarningTypePropertyValueEmpty, churchEntry.Warnings[0].Type)

	// I should have a collission between the two entries
	receptionEntry := log[3]
	assert.Equal(t, 2, len(receptionEntry.Collisions))
	assert.Equal(t, 2, receptionEntry.Collisions[0])
	assert.Equal(t, 3, receptionEntry.Collisions[1])
	writeTestRLog = false
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
						idx:       idx,
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

func (t *integTestContext) RLog() (RenameLog, error) {
	return ReadRenameLog(t.filesDir + string(os.PathSeparator) + rafStatusFile)
}
