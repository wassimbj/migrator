package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	// "github.com/jackc/pgx/pgtype"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Migrator struct {
	conn *pgxpool.Pool
}

func Init(conn *pgxpool.Pool) *Migrator {
	// conn, err := pgxpool.Connect(context.Background(), "postgres://root:12S34@localhost:5432/testdb")
	// if err != nil {
	// 	fmt.Println("DB_ERROR: ", err)
	// }
	return &Migrator{
		conn: conn,
	}
}

/*
	TODO: complete the  logic
*/
// https://www.postgresql.org/docs/9.5/datatype.html#DATATYPE-TABLE
func GetPgType(goType string) string {
	if goType == "string" {
		return "varchar"
	} else if goType == "int" || goType == "int8" || goType == "int32" {
		return "int"
	} else if goType == "int64" {
		return "bigint"
	} else if goType == "struct{}" {
		return "json"
	}

	return ""
}

type Field struct {
	Name  string
	Ftype string
	Size  string
}

func (m *Migrator) TableAlreadyExist(name string) (bool, error) {
	var count int
	err := m.conn.QueryRow(
		context.Background(),
		"SELECT count(*) FROM information_schema.columns WHERE table_name=$1",
		name,
	).Scan(&count)

	return count > 0, err
}

// will run if we faced an error when creating the table
func (m *Migrator) UpdateTable(name string, fields []Field) {

	/*
		TODO:
		- get table schema (cols, types, len....)
		- compare to the given fields
		- update changes
	*/
	var detils []struct {
		ColName    string
		DataType   string
		CharMaxLen string `json:"character_maximum_length"`
		ColDefault string `json:"column_default"`
		IsNullable string `json:"is_nullable"`
	}
	pgxscan.Select(context.Background(), m.conn, &detils, `
		SELECT column_name, data_type, character_maximum_length, column_default, is_nullable
		FROM information_schema.columns WHERE table_name = $1
	`, name)

	fmt.Println(detils)
	// m.conn.Query(
	// 	context.Background(),
	// 	``,
	// 	name,
	// )

}

// https://www.postgresql.org/docs/9.1/sql-createtable.html
func (m *Migrator) CreateTable(name string, fields []Field) string {
	/*
	 types that can have size
	 varbit [ (n) ], char [ (n) ], varchar [ (n) ], decimal [ (p, s) ]
	*/

	// generate create table sql
	sqlStr := fmt.Sprintf("CREATE TABLE %s ( \n", name)

	for i, field := range fields {
		canHaveSize := field.Ftype == "varbit" || field.Ftype == "char" || field.Ftype == "varchar" || field.Ftype == "decimal"
		if canHaveSize {
			size := field.Size
			if size == "" {
				size = "255" // default size
			}
			sqlStr += fmt.Sprintf("%s %s(%s)", field.Name, field.Ftype, size)
		} else {
			sqlStr += fmt.Sprintf("%s %s", field.Name, field.Ftype)
		}

		if i < len(fields)-1 {
			sqlStr += ",\n"
		}
	}

	sqlStr += "\n)"

	err := m.conn.QueryRow(context.Background(), sqlStr).Scan()

	if err != nil {
		fmt.Println("CREATE ERROR :", err)
		m.UpdateTable(name, fields)
	}
	// fmt.Println(sqlStr)

	return sqlStr

}

func NiceTableName(name string) string {
	// e.g: TestTable => test_tables
	s := strings.ToLower(string(name[0]))

	for i := 1; i < len(name); i++ {
		lowerCh := strings.ToLower(string(name[i]))
		if unicode.IsUpper(rune(name[i])) {
			s += "_" + lowerCh
		} else {
			s += lowerCh
		}
	}

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

func (m *Migrator) AutoMigrate(schema interface{}) {
	elem := reflect.ValueOf(schema).Elem()
	tableName := reflect.TypeOf(schema).Elem().Name()
	var fieldNames []Field
	fmt.Println("Tbl: ", NiceTableName(tableName))

	for i := 0; i < elem.NumField(); i++ {
		fieldName := elem.Type().Field(i).Name
		field, _ := reflect.TypeOf(schema).Elem().FieldByName(fieldName)

		colName := field.Tag.Get("col")
		colSize := field.Tag.Get("size")
		colType := field.Tag.Get("type")

		fieldNames = append(fieldNames, Field{
			Name:  colName,
			Ftype: colType,
			Size:  colSize,
		})
	}

	tbl := strings.ToLower(tableName) + "s"
	exists, err := m.TableAlreadyExist(tbl)
	if err != nil {
		panic(err)
	}
	if exists {
		m.UpdateTable(tbl, fieldNames)
	} else {
		m.CreateTable(tbl, fieldNames)
	}

}

type Testo struct {
	Id   int    `col:"id" type:"int"`
	Name string `col:"name" type:"varchar" size:"30"`
	// Email       string `col:"email" type:"varchar" size:"100"`
	Password    string `col:"password" type:"varchar" size:"20"`
	ConfirmCode string `col:"confirm_code" type:"varchar" size:"6"`
}

func main() {

	conn, err := pgxpool.Connect(context.Background(), "postgres://root:1234@localhost:5432/testdb")
	if err != nil {
		fmt.Println("DB_CONN_ERROR ", err)
	}

	m := Init(conn)

	m.AutoMigrate(&Testo{})

}
