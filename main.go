package main

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	// "github.com/jackc/pgx/pgtype"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Migrator struct {
	conn *pgxpool.Pool
}

type Field struct {
	Name       string
	DataType   string
	Size       string
	DefaultVal string
	IsNullable bool
}

// type SchemaInfo struct {
// 	ColName    interface{}
// 	DataType   interface{}
// 	Size       interface{}
// 	DefaultVal interface{}
// 	IsNullable interface{}
// }

func Init(conn *pgxpool.Pool) *Migrator {
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

func (m *Migrator) TableAlreadyExist(name string) (bool, error) {
	var count int
	err := m.conn.QueryRow(
		context.Background(),
		"SELECT count(*) FROM information_schema.columns WHERE table_name=$1",
		name,
	).Scan(&count)

	return count > 0, err
}

type ChangedField struct {
	f     Field
	isNew bool
}

// check and return new and old fields
func getChangedFields(fields []Field, schema []Field) []ChangedField {
	var changedFields []ChangedField
	var colExist bool = false

	// check if fields exist on the database tables
	for _, f := range fields {
		for _, s := range schema {
			if f.Name == s.Name {
				colExist = true
				break
			}
		}
		changedFields = append(changedFields, ChangedField{
			f:     f,
			isNew: !colExist,
		})
		colExist = false
	}

	return changedFields
}

func (m *Migrator) AddCol(tbl string, f Field) error {
	err := m.conn.QueryRow(
		context.Background(),
		"ALTER TABLE "+tbl+" ADD COLUMN "+colFmt(f),
	).Scan()

	if err != pgx.ErrNoRows {
		return err
	}
	return nil
}

func (m *Migrator) EditCol(tbl string, f Field) error {
	err := m.conn.QueryRow(
		context.Background(),
		"ALTER TABLE "+tbl+" ADD COLUMN "+colFmt(f),
	).Scan()

	if err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// add columns, modify size, type... add constrains...
func (m *Migrator) UpdateTable(name string, fields []Field) {
	schema, _ := m.conn.Query(
		context.Background(), `
		SELECT column_name, data_type, character_maximum_length, column_default, is_nullable
		FROM information_schema.columns WHERE table_name = $1`, name,
	)
	var schemaFields []Field
	for schema.Next() {
		val, _ := schema.Values()
		size := ""
		defVal := ""
		if val[2] != nil {
			size = strconv.Itoa(int(val[2].(int32)))
		} else if val[3] != nil {
			defVal = val[3].(string)
		}

		schemaFields = append(schemaFields, Field{
			Name:       val[0].(string),
			DataType:   val[1].(string),
			Size:       size,
			DefaultVal: defVal,
			IsNullable: val[4] == "YES",
		})
	}

	for _, cf := range getChangedFields(fields, schemaFields) {
		fmt.Println(cf)
		if cf.isNew {
			m.AddCol(name, cf.f)
		} else {
			fmt.Println("Editing col...")
		}

	}

}

// format col
func colFmt(field Field) string {
	/*
	 types that can have size
	 varbit [ (n) ], char [ (n) ], varchar [ (n) ], decimal [ (p, s) ]
	*/
	switch field.DataType {
	// types that can have size
	case "varbit", "char", "varchar", "decimal":
		size := field.Size
		if size == "" {
			size = "255" // default size
		}
		return fmt.Sprintf("%s %s(%s) NOT NULL", field.Name, field.DataType, size)
	default:
		return fmt.Sprintf("%s %s NOT NULL", field.Name, field.DataType)
	}
}

// https://www.postgresql.org/docs/9.1/sql-createtable.html
func (m *Migrator) CreateTable(name string, fields []Field) string {

	// generate create table sql
	// ------------------------
	sqlStr := fmt.Sprintf("CREATE TABLE %s ( \n", name)
	for i, field := range fields {
		sqlStr += colFmt(field)
		// switch field.DataType {
		// // types that can have size
		// case "varbit", "char", "varchar", "decimal":
		// 	size := field.Size
		// 	if size == "" {
		// 		size = "255" // default size
		// 	}
		// 	sqlStr += fmt.Sprintf("%s %s(%s) NOT NULL", field.Name, field.DataType, size)
		// default:
		// 	sqlStr += fmt.Sprintf("%s %s NOT NULL", field.Name, field.DataType)
		// }

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

func (m *Migrator) Migrate(schema interface{}) {
	elem := reflect.ValueOf(schema).Elem()
	tableName := reflect.TypeOf(schema).Elem().Name()
	var fieldNames []Field

	for i := 0; i < elem.NumField(); i++ {
		fieldName := elem.Type().Field(i).Name
		field, _ := reflect.TypeOf(schema).Elem().FieldByName(fieldName)

		colName := field.Tag.Get("col")
		colSize := field.Tag.Get("size")
		colType := field.Tag.Get("type")

		fieldNames = append(fieldNames, Field{
			Name:     colName,
			DataType: colType,
			Size:     colSize,
		})
	}

	tbl := CamelToSnakeCase(tableName, true)

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
	Id          int    `col:"id" type:"int"`
	Name        string `col:"name" type:"varchar" size:"30"`
	Email       string `col:"email" type:"varchar" size:"100"`
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
