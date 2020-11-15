package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const titlePropRegexGroup = "\\ \\-\\ ([A-Za-z0-9\\ ]+)\\ \\-"

func TestExtractPropGroup(t *testing.T) {
	p, err := NewProp("title", titlePropRegexGroup)
	assert.Nil(t, err)

	vals := extractVarValues("wedding - chapel first - video01", []Prop{p}, Opts{})
	assert.NotNil(t, vals)
	title, ok := vals["$title"]
	assert.True(t, ok)
	assert.Equal(t, "chapel first", title)
}

func TestExtractProp(t *testing.T) {
	p, err := NewProp("title", "\\ \\-\\ [A-Za-z0-9\\ ]+\\ \\-")
	assert.Nil(t, err)

	vals := extractVarValues("wedding - chapel first - video01", []Prop{p}, Opts{})
	assert.NotNil(t, vals)
	title, ok := vals["$title"]
	assert.True(t, ok)
	assert.Equal(t, " - chapel first -", title)
}

func TestExtractPropNoMatch(t *testing.T) {
	p, err := NewProp("title", titlePropRegexGroup)
	assert.Nil(t, err)

	vals := extractVarValues("wedding_chapel first - video01", []Prop{p}, Opts{})
	assert.NotNil(t, vals)
	title, ok := vals["$title"]
	assert.True(t, ok)
	assert.Equal(t, "", title)
}

func TestRenameStringSimple(t *testing.T) {
	values := make(map[string]string)
	values["$title"] = "test title"
	out, err := ParseOutput("rip - $title.mkv")
	assert.Nil(t, err)

	renamed, err := GenerateName(values, out, Opts{})
	assert.Nil(t, err)
	assert.Equal(t, "rip - test title.mkv", renamed)
}

func TestRenameStringMissingProperty(t *testing.T) {
	values := make(map[string]string)
	values["$title"] = "test title"
	out, err := ParseOutput("rip - $title2.mkv")
	assert.Nil(t, err)

	renamed, err := GenerateName(values, out, Opts{})
	assert.Nil(t, err)
	assert.Equal(t, "rip - .mkv", renamed)
}
