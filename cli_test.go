// This file contains integration tests for the entire CLI as well as utility methods
// to generate tempoary files for the tests
package main

import (
	"bytes"
	"fmt"
	"io"
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

func TestNameStdout(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "Home", "Chapel", "Church", "Reception", "Party")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9]+)\\.mkv", "--output", "test - $cnt - $title.avi"}
	args = append(args, testCtx.Files(true)...)

	// capture stdout
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err = app.Run(args)
	assert.Nil(t, err)

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()
	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	files, err := testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(files))
	assert.Equal(t, "test - 2 - Chapel.avi", files[1])

	outFiles := strings.Split(out, "\n")
	assert.Equal(t, 6, len(outFiles))
	assert.Equal(t, "test - 1 - Home.avi", outFiles[0])
	assert.Equal(t, "test - 2 - Chapel.avi", outFiles[1])
	assert.Equal(t, "test - 3 - Church.avi", outFiles[2])
	assert.Equal(t, "test - 4 - Reception.avi", outFiles[3])
	assert.Equal(t, "test - 5 - Party.avi", outFiles[4])
	assert.Equal(t, "", outFiles[5])
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

func TestTitleWithSliceFormatter(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "Home", "Chapel", "Church", "Reception", "Party")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9]+)\\.mkv", "--output", "test - $cnt - $title[>:3].avi"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.Nil(t, err)

	files, err := testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(files))
	assert.Equal(t, "test - 1 - Hom.avi", files[0])
	assert.Equal(t, "test - 2 - Cha.avi", files[1])
	assert.Equal(t, "test - 3 - Chu.avi", files[2])
	assert.Equal(t, "test - 4 - Rec.avi", files[3])
	assert.Equal(t, "test - 5 - Par.avi", files[4])
}

func TestTitleWithReplacingFormatter(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "my.home.video")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9\\.]+)\\.mkv", "--output", "test - $cnt - $title[/\\./ /].avi"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.Nil(t, err)
	files, err := testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "test - 1 - my home video.avi", files[0])
}

func TestMultipleFormatters(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "my.home.video")
	assert.Nil(t, err)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9\\.]+)\\.mkv", "--output", "test - $cnt - $title[/\\./ /,>:8].avi"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.Nil(t, err)
	files, err := testCtx.ListFilesInWorkingDir(false, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "test - 1 - my home .avi", files[0])
}

func TestPartialRenameLog(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	err = testCtx.CreateFiles("[UnionVideos] Wedding - $cnt - $title.mkv", "Home", "Chapel", "_Church", "_Reception", "Party")
	assert.Nil(t, err)
	os.Mkdir(testCtx.filesDir+string(os.PathSeparator)+"test - .avi", os.ModeAppend)

	app := getApp()
	args := []string{"raf", "--prop", "title=\\d\\ \\-\\ ([A-Za-z0-9]+)\\.mkv", "--output", "test - $title.avi"}
	args = append(args, testCtx.Files(true)...)
	err = app.Run(args)
	assert.NotNil(t, err)

	log, err := testCtx.RLog()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(log))
}

func TestManDownload(t *testing.T) {
	testCtx, err := createIntegTestContext(t)
	assert.Nil(t, err)

	rafUri := "https://raw.githubusercontent.com/sapessi/raf/v-1/raf.1"
	manFile, err := getManPage(rafUri, testCtx.filesDir, "-1")
	assert.NotNil(t, err)
	assert.Errorf(t, err, "Could not retrieve raf man page from repository (%s): %s", rafUri, "404 Not Found")
	assert.Equal(t, "", manFile)

	rafUri = "https://raw.githubusercontent.com/sapessi/raf/v0.3.0/raf.1"
	rafHome := testCtx.filesDir + string(os.PathSeparator) + ".raf" // make sure we can create the dir
	manPath := rafHome + string(os.PathSeparator) + "v0.3.0_raf.1"
	manFile, err = getManPage(rafUri, rafHome, "v0.3.0")
	assert.Nil(t, err)
	assert.Equal(t, manPath, manFile)
	assert.FileExists(t, manPath)
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
