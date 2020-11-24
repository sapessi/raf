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

func TestSliceFormatter(t *testing.T) {
	testStr := "testing new string"
	formatter := NewSliceFormatter(-1, 10)
	formatted, err := formatter.Format(testStr, renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "testing ne", formatted)

	formatter = NewSliceFormatter(-1, 50)
	formatted, err = formatter.Format(testStr, renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, testStr, formatted)

	formatter = NewSliceFormatter(10, 50)
	formatted, err = formatter.Format(testStr, renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "w string", formatted)
}

func TestValueShorterThanSlice(t *testing.T) {
	formatter := NewSliceFormatter(5, 10)
	formatted, err := formatter.Format("tst", renamerState{})
	assert.Nil(t, err)
	assert.Equal(t, "", formatted)
}
