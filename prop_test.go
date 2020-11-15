package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropParseSimple(t *testing.T) {
	prop, err := ParseProp("title=\\ \\- ([A-Za-z0-9\\ ]+)\\ \\-")
	assert.Nil(t, err)
	assert.Equal(t, "title", prop.Name)
	assert.Equal(t, "\\ \\- ([A-Za-z0-9\\ ]+)\\ \\-", prop.Matcher)
}
