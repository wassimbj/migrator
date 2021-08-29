package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	// "github.com/jackc/pgx/pgtype"

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

type SchemaInfo struct {
	ColName    interface{} `json:"column_name"`
	DataType   interface{} `json:"data_type"`
	CharMaxLen interface{} `json:"character_maximum_length"`
	ColDefault interface{} `json:"column_default"`
	IsNullable interface{} `json:"is_nullable"`
}

func getChangedFields(fields []Field, schema []SchemaInfo) {

}

/*
	TODO:
	- get table schema (cols, types, len....)
	- compare to the given fields
	- update changes
*/
// add columns, modify size, type... add constrains...
func (m *Migrator) UpdateTable(name string, fields []Field) {
	schema, _ := m.conn.Query(
		context.Background(), `
		SELECT column_name, data_type, character_maximum_length, column_default, is_nullable
		FROM information_schema.columns WHERE table_name = $1`, name,
	)
	var schemaFields []SchemaInfo
	for schema.Next() {
		val, _ := schema.Values()
		schemaFields = append(schemaFields, SchemaInfo{
			ColName:    val[0],
			DataType:   val[1],
			CharMaxLen: val[2],
			ColDefault: val[3],
			IsNullable: val[4],
		})
	}

	schemaLen := len(schemaFields)
	fieldsLen := len(fields)

	// new column to add
	if fieldsLen > schemaLen {
		// m.conn.QueryRow(
		// 	context.Background(),
		// 	`
		// 	ALTER TABLE $1
		// 	ADD COLUMN contact_name VARCHAR NOT NULL;
		// 	`,
		// )
	}
	// fmt.Println(len(schema.RawValues()), )
	// for _, field := range fields {

	// }

	for schema.Next() {
		val, _ := schema.Values()
		fmt.Println(val[0], val[1], val[2], val[3], val[4])
	}

}

// https://www.postgresql.org/docs/9.1/sql-createtable.html
func (m *Migrator) CreateTable(name string, fields []Field) string {
	/*
	 types that can have size
	 varbit [ (n) ], char [ (n) ], varchar [ (n) ], decimal [ (p, s) ]
	*/

	// generate create table sql
	// ------------------------
	sqlStr := fmt.Sprintf("CREATE TABLE %s ( \n", name)
	for i, field := range fields {
		switch field.Ftype {
		// types that can have size
		case "varbit", "char", "varchar", "decimal":
			size := field.Size
			if size == "" {
				size = "255" // default size
			}
			sqlStr += fmt.Sprintf("%s %s(%s) NOT NULL", field.Name, field.Ftype, size)
		default:
			sqlStr += fmt.Sprintf("%s %s NOT NULL", field.Name, field.Ftype)
		}

		if i < len(fields)-1 {
			sqlStr += ",\n"
		}
	}
	sqlStr += "\n)"
	// ------------------------

	err := m.conn.QueryRow(context.Background(), sqlStr).Scan()

	if err != nil {
		fmt.Println("CREATE ERROR :", err)
		m.UpdateTable(name, fields)
	}

	return sqlStr

}

// return a nice table name
// e.g: UserPayment => user_payments
func NiceTableName(name string) string {
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

func (m *Migrator) Migrate(schema interface{}) {
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

	tbl := NiceTableName(tableName)

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

	m.Migrate(&Testo{})

}
