package main

import "unicode"

const (
	TokenTypeLiteral  = "literal"
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
	Formatter string
}

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
	str      string
	curValue string
}

func newParser(out string) statefulParser {
	return statefulParser{
		idx: 0,
		str: out,
		len: len(out),
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

func (p *statefulParser) isLast() bool {
	return p.idx >= p.len
}

func (p *statefulParser) parseProperty() (Token, error) {
	formatter := ""
	prop := ""
	for !p.isLast() {
		prop += string(p.nextChr())
		if !unicode.IsLetter(p.peek()) && !unicode.IsDigit(p.peek()) {
			break
		}
	}
	// next we have a formatter
	if p.peek() == '[' {
		for !p.isLast() {
			formatter += string(p.nextChr())
			if p.peek() == ']' {
				formatter += string(p.nextChr())
				break
			}
		}
	}
	// TODO: validate formatter
	return Token{
		Type:      TokenTypeProperty,
		Value:     prop,
		Formatter: formatter,
	}, nil
}

func (p *statefulParser) parseLiteral() (Token, error) {
	str := ""
	for !p.isLast() && p.peek() != '$' {
		str += string(p.nextChr())
	}

	return Token{
		Type:      TokenTypeLiteral,
		Value:     str,
		Formatter: "",
	}, nil
}
