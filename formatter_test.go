package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZeroPadding(t *testing.T) {
	formatter := NewPaddingFormatter('0', 3)
	formatted, err := formatter.Format("1", renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "001", formatted)
}

func TestValueLongerThanPadding(t *testing.T) {
	formatter := NewPaddingFormatter('0', 3)
	formatted, err := formatter.Format("123", renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "123", formatted)

	formatter = NewPaddingFormatter('0', 3)
	formatted, err = formatter.Format("1234", renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "1234", formatted)
}
