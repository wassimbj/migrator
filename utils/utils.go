package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// convert Camel to snake case
// e.g: UserPayment => user_payment(s)
func CamelToSnakeCase(name string, addPl bool) string {
	s := strings.ToLower(string(name[0]))

	for i := 1; i < len(name); i++ {
		lowerCh := strings.ToLower(string(name[i]))
		if unicode.IsUpper(rune(name[i])) {
			s += "_" + lowerCh
		} else {
			s += lowerCh
		}
	}

	if addPl {
		// plural prefix
		var pl string

		switch string(s[len(name)-1]) {
		case "s", "ss", "sh", "ch", "x", "z":
			pl = "es"
		default:
			pl = "s"
		}
		return s + pl
	}

	return s
}

/*
 types that can have size
 varbit [ (n) ], char [ (n) ], varchar [ (n) ], decimal [ (p, s) ]
*/
func GetDataTypeWithSize(dataType string, size string) string {
	// var x Field
	switch dataType {
	case "varbit", "char", "varchar", "decimal":
		s := size
		if s == "" {
			s = "255" // default size
		}
		return fmt.Sprintf("%s(%s)", dataType, s)
	default:
		return fmt.Sprintf("%s", dataType)
	}

}

// https://www.postgresql.org/docs/9.5/datatype.html#DATATYPE-TABLE
// convert Golang types to Postgres types;
// e.g: string => varchar
func GoToPgType(mtype string) string {
	switch mtype {
	case "string":
		return "varchar"
	case "int64":
		return "bigint"
	case "int", "int8", "int16", "int32":
		return "int"
	case "bool":
		return "bool"
	case "float32", "float64":
		return "numeric"
	case "net.IP":
		return "inet"
	case "time.Time":
		return "timestamp"
	case "time.Duration":
		return "time"
	default:
		return ""
	}
}

// convert data type name to alias (e.g: character varying => varchar)
func PgTypeToAlias(mtype string) string {
	switch mtype {
	case "character":
		return "char"
	case "bit varying":
		return "varbit"
	case "boolean":
		return "bool"
	case "character varying":
		return "varchar"
	case "double precision":
		return "float8"
	case "integer":
		return "int"
	default:
		return mtype
	}
}
