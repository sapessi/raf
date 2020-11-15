package main

import "strconv"

// ReservedVarNames is a map with the variable name - such as $cnt - as its key and a function
// that extracts the correct value from the renamer state as its value.
var ReservedVarNames = map[string]func(renamerState) string{
	"$cnt": func(rs renamerState) string {
		return strconv.Itoa(rs.idx + 1)
	},
	"$ext": func(rs renamerState) string {
		return rs.extension
	},
	"$fname": func(rs renamerState) string {
		return rs.fileName
	},
}
