package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const singleLiteralStr = "one literal string"
const literalPlusParam = "literal-$second"
const literalPlusParamWithFormatter = "literal_$cnt[000] - ext.mkv"
const literalSquareBrackets = "fileName [divx] - episode $cnt[000].$ext"
const paramAndEscapedBracket = "t - $title\\[2020]"

func TestZeroLen(t *testing.T) {
	parser := newParser("")
	tokens, err := parser.parse()
	assert.Nil(t, err)
	assert.Nil(t, tokens)
}

func TestSingleLiteral(t *testing.T) {
	parser := newParser(singleLiteralStr)
	tokens, err := parser.parse()

	assert.Nil(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, 1, len(tokens))
	assert.Equal(t, singleLiteralStr, tokens[0].Value)
	assert.Equal(t, "", tokens[0].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
}

func TestParamAndEscapeBracket(t *testing.T) {
	parser := newParser(paramAndEscapedBracket)
	tokens, err := parser.parse()

	assert.Nil(t, err)
	assert.Equal(t, 3, len(tokens))
	assert.Equal(t, "t - ", tokens[0].Value)
	assert.Equal(t, "", tokens[0].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
	assert.Equal(t, "$title", tokens[1].Value)
	assert.Equal(t, TokenTypeProperty, tokens[1].Type)
	assert.Equal(t, "", tokens[1].Formatter)
	assert.Equal(t, "[2020]", tokens[2].Value)
	assert.Equal(t, "", tokens[2].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[2].Type)
}

func TestLiteralAndParam(t *testing.T) {
	parser := newParser(literalPlusParam)
	tokens, err := parser.parse()

	assert.Nil(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, 2, len(tokens))
	assert.Equal(t, "literal-", tokens[0].Value)
	assert.Equal(t, "", tokens[0].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
	assert.Equal(t, "$second", tokens[1].Value)
	assert.Equal(t, TokenTypeProperty, tokens[1].Type)
	assert.Equal(t, "", tokens[1].Formatter)
}

func TestLiteralAndParamWithFormatter(t *testing.T) {
	parser := newParser(literalPlusParamWithFormatter)
	tokens, err := parser.parse()

	assert.Nil(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, 3, len(tokens))
	assert.Equal(t, "literal_", tokens[0].Value)
	assert.Equal(t, "", tokens[0].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
	assert.Equal(t, "$cnt", tokens[1].Value)
	assert.Equal(t, TokenTypeProperty, tokens[1].Type)
	assert.Equal(t, "[000]", tokens[1].Formatter)
	assert.Equal(t, " - ext.mkv", tokens[2].Value)
	assert.Equal(t, "", tokens[2].Formatter)
	assert.Equal(t, TokenTypeLiteral, tokens[2].Type)
}

func TestLiteralSquareBrackets(t *testing.T) {
	parser := newParser(literalSquareBrackets)
	tokens, err := parser.parse()

	assert.Nil(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, 4, len(tokens))
	assert.Equal(t, "fileName [divx] - episode ", tokens[0].Value)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
	assert.Equal(t, "$cnt", tokens[1].Value)
	assert.Equal(t, TokenTypeProperty, tokens[1].Type)
	assert.Equal(t, "[000]", tokens[1].Formatter)
	assert.Equal(t, ".", tokens[2].Value)
	assert.Equal(t, TokenTypeLiteral, tokens[2].Type)
	assert.Equal(t, "$ext", tokens[3].Value)
	assert.Equal(t, TokenTypeProperty, tokens[3].Type)
}

func TestLiteralSquareBracketsPublicParse(t *testing.T) {
	tokens, err := ParseOutput(literalSquareBrackets)

	assert.Nil(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, 4, len(tokens))
	assert.Equal(t, "fileName [divx] - episode ", tokens[0].Value)
	assert.Equal(t, TokenTypeLiteral, tokens[0].Type)
	assert.Equal(t, "$cnt", tokens[1].Value)
	assert.Equal(t, TokenTypeProperty, tokens[1].Type)
	assert.Equal(t, "[000]", tokens[1].Formatter)
	assert.Equal(t, ".", tokens[2].Value)
	assert.Equal(t, TokenTypeLiteral, tokens[2].Type)
	assert.Equal(t, "$ext", tokens[3].Value)
	assert.Equal(t, TokenTypeProperty, tokens[3].Type)
}
