package main

import (
	"fmt"
	"strconv"
	"unicode"
)

const (
	// TokenTypeLiteral is used for literal strings in the output
	TokenTypeLiteral = "literal"
	// TokenTypeProperty is used for variables to be replaced in the output
	TokenTypeProperty = "property"
)

// TokenStream is a slice of tokens produced by parsing an output string
type TokenStream = []Token

// TokenType tells the RenameAllFiles function whether this token is of type literal or property
type TokenType = string

// Token represents a potion of the parsed output format. Tokens have a type (literal or property),
// a value, and a formatter. The value is used to store the string literal for tokens of type literal
// or the name of the property for tokens of type property. The formatter field is only populated
// for tokens of type property and includes the additional formatting information for the text content
// of the property. For example, "literal" This is a string literal; "$cnt" this is a property token;
// "$cnt[000]" and this is a property token with formatting information.
type Token struct {
	Type      TokenType
	Value     string
	Formatter FormattingPipeline
}

// FormattingPipeline is a slice of formatters associated with a property. Formatters are executed in
// order and operate on each others output
type FormattingPipeline = []Formatter

// ParseOutput takes a string declaration of the output format - for example "fileName [divx] - episode $cnt[000].$ext"
// and parses it into a sequence of tokens. Tokens can be of type literal or property: Literals are string literals that
// will be appended to the output file name and properties are variables for substitution.
//
// The token stream returned by this method can be passed to the RenameAllFiles method.
func ParseOutput(out string) (TokenStream, error) {
	parser := newParser(out)
	return parser.parse()
}

type statefulParser struct {
	idx      int
	len      int
	str      []rune
	curValue string
}

func newParser(out string) statefulParser {
	runeSlice := []rune(out)
	return statefulParser{
		idx: 0,
		str: runeSlice,
		len: len(runeSlice),
	}
}

func (p *statefulParser) parse() (TokenStream, error) {
	if p.len == 0 {
		return nil, nil
	}
	tokens := make([]Token, 0)
	for !p.isLast() {
		if p.peek() == '$' {
			prop, err := p.parseProperty()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, prop)
		} else {
			literal, err := p.parseLiteral()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, literal)
		}
	}
	return tokens, nil
}

func (p *statefulParser) nextChr() rune {
	chr := rune(p.str[p.idx])
	p.idx++
	return chr
}

func (p *statefulParser) peek() rune {
	if p.isLast() {
		return ' '
	}
	// at this point we have already incremented idx in the nextChr function
	return rune(p.str[p.idx])
}

func (p *statefulParser) cur() rune {
	if p.idx == 0 {
		return ' '
	}
	return rune(p.str[p.idx-1])
}

func (p *statefulParser) isLast() bool {
	return p.idx >= p.len
}

func (p *statefulParser) parseProperty() (Token, error) {
	prop := ""
	for !p.isLast() {
		prop += string(p.nextChr())
		if !unicode.IsLetter(p.peek()) && !unicode.IsDigit(p.peek()) {
			break
		}
	}

	// next we have a formatter
	if p.peek() == '[' {
		formatters, err := p.parseFormatters()
		if err != nil {
			return Token{}, err
		}
		// advance to skip the closing ]
		p.nextChr()

		return Token{
			Type:      TokenTypeProperty,
			Value:     prop,
			Formatter: formatters,
		}, nil
	}

	return Token{
		Type:      TokenTypeProperty,
		Value:     prop,
		Formatter: nil,
	}, nil
}

func (p *statefulParser) parseFormatters() (FormattingPipeline, error) {
	// at this point we should be just before the [, skip to next char
	p.nextChr()
	formatters := make([]Formatter, 0)

	// switch by formatter type
	for !p.isLast() && p.peek() != ']' {
		chr := p.peek()
		var formatter Formatter
		var err error
		switch chr {
		case '%': // padding
			formatter, err = p.parsePaddingFormatter()
		case '>': //slice
			formatter, err = p.parseSliceFormatter()
		case '/': // replacing
			formatter, err = p.parseReplacingFormatter()
		default:
			return nil, fmt.Errorf("Unknown formatter type %s", string(chr))
		}
		if err != nil {
			return nil, err
		}
		formatters = append(formatters, formatter)
	}

	return formatters, nil
}

func (p *statefulParser) parsePaddingFormatter() (*PaddingFormatter, error) {
	// at this point we should be peeking at the %
	p.nextChr() // %

	padChar := p.nextChr()

	padLength := ""
	for !p.isLast() && (p.peek() != ']' && p.peek() != ',') {
		padLength += string(p.nextChr())
	}
	padLengthInt, err := strconv.Atoi(padLength)
	if err != nil {
		return &PaddingFormatter{}, err
	}
	padder := NewPaddingFormatter(padChar, padLengthInt)
	return &padder, nil
}

func (p *statefulParser) parseSliceFormatter() (*SliceFormatter, error) {
	p.nextChr() // skip the >

	startPos := -1
	startPosStr := ""
	for !p.isLast() && p.peek() != ':' {
		if !unicode.IsDigit(p.peek()) {
			return &SliceFormatter{}, fmt.Errorf("Found %s in beginning position of slice formatter, only numeric values are allowed", string(p.peek()))
		}
		startPosStr += string(p.nextChr())
	}
	if startPosStr != "" {
		startPosTmp, err := strconv.Atoi(startPosStr)
		if err != nil {
			return &SliceFormatter{}, err
		}
		startPos = startPosTmp
	}

	p.nextChr() // skip the :
	endPos := -1
	endPosStr := ""
	for !p.isLast() && (p.peek() != ']' && p.peek() != ',') {
		if !unicode.IsDigit(p.peek()) {
			return &SliceFormatter{}, fmt.Errorf("Found %s in end position of slice formatter, only numeric values are allowed", string(p.peek()))
		}
		endPosStr += string(p.nextChr())
	}
	if endPosStr != "" {
		endPosTmp, err := strconv.Atoi(endPosStr)
		if err != nil {
			return &SliceFormatter{}, err
		}
		endPos = endPosTmp
	}
	slicer := NewSliceFormatter(startPos, endPos)
	return &slicer, nil
}

func (p *statefulParser) parseReplacingFormatter() (*ReplacingFormatter, error) {
	p.nextChr() // skip the /

	find := ""
	for !p.isLast() {
		if p.peek() == '/' && p.cur() != '\\' {
			break
		}
		find += string(p.nextChr())
	}

	p.nextChr() // skip the /

	replace := ""
	for !p.isLast() {
		if p.peek() == '/' && p.cur() != '\\' {
			break
		}
		replace += string(p.nextChr())
	}

	p.nextChr() // skip the final /

	formatter, err := NewReplacingFormatter(find, replace)
	return &formatter, err
}

func (p *statefulParser) parseLiteral() (Token, error) {
	str := ""
	for !p.isLast() && p.peek() != '$' {
		nextChr := p.nextChr()
		// remove escapes
		if nextChr == '\\' {
			nextChr = p.nextChr()
		}
		str += string(nextChr)
	}

	return Token{
		Type:      TokenTypeLiteral,
		Value:     str,
		Formatter: nil,
	}, nil
}
