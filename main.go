package main

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/wassimbj/migrator/queries"
	"github.com/wassimbj/migrator/utils"
)

type Migrator struct {
	conn *pgxpool.Pool
}

func Init(conn *pgxpool.Pool) *Migrator {
	return &Migrator{
		conn: conn,
	}
}

func (m *Migrator) Migrate(schema interface{}) {
	elem := reflect.ValueOf(schema).Elem()
	tableName := reflect.TypeOf(schema).Elem().Name()
	var fieldNames []queries.Field

	for i := 0; i < elem.NumField(); i++ {
		fieldName := elem.Type().Field(i).Name
		field, _ := reflect.TypeOf(schema).Elem().FieldByName(fieldName)

		colName := field.Tag.Get("col")
		colSize := field.Tag.Get("size")
		colType := field.Tag.Get("type")

		fieldNames = append(fieldNames, queries.Field{
			Name:     colName,
			DataType: colType,
			Size:     colSize,
		})
	}

	tbl := utils.CamelToSnakeCase(tableName, true)

	exists, err := queries.TableAlreadyExist(m.conn, tbl)
	if err != nil {
		panic(err)
	}
	if exists {
		queries.UpdateTable(m.conn, tbl, fieldNames)
	} else {
		queries.CreateTable(m.conn, tbl, fieldNames)
	}

}

type User struct {
	Id          int    `col:"id" type:"int"`
	Name        string `col:"name" type:"varchar" size:"30"`
	Email       string `col:"email" type:"varchar" size:"100"`
	Password    string `col:"password" type:"varchar" size:"20"`
	ConfirmCode string `col:"confirm_code" type:"varchar" size:"6"`
	CreatedAt   string `col:"created_at" type:"timestamp"`
}

func main() {

	conn, err := pgxpool.Connect(context.Background(), "postgres://root:1234@localhost:5432/testdb")
	if err != nil {
		fmt.Println("DB_CONN_ERROR ", err)
	}

	m := Init(conn)

	m.Migrate(&User{})

}
