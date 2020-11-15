package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Prop defines a property that can be used in the RenameAllFiles to extract
// values from the original file name.
type Prop struct {
	Name    string
	Matcher string
	Regex   *regexp.Regexp
}

// ParseProp populates a Prop object based on the string format passed to the cli: "propName=/regex/"
func ParseProp(v string) (Prop, error) {
	kv := strings.Split(v, "=")
	if len(kv) != 2 {
		return Prop{}, fmt.Errorf("Invalid property definition %s. Property definitions must contain a name and a matcher: name=/matcher/", v)
	}
	if _, ok := ReservedVarNames[kv[0]]; ok {
		return Prop{}, fmt.Errorf("The property name %s is reserved", kv[0])
	}
	return NewProp(kv[0], kv[1])
}

// NewProp creates a new Prop object for the given name and matcher regex. The regex string is compiled
// into a RegEx struct.
func NewProp(name, matcher string) (Prop, error) {
	regex, err := regexp.Compile(matcher)
	if err != nil {
		return Prop{}, fmt.Errorf("Invalid matcher for pattern %s: Selector must be valid regular expressions. %v", matcher, err)
	}
	return Prop{
		Name:    name,
		Matcher: matcher,
		Regex:   regex,
	}, nil
}
