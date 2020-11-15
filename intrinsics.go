package main

import "strconv"

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
