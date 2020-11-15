package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Prop struct {
	Name    string
	Matcher string
	Regex   *regexp.Regexp
}

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
