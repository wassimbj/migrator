package main

import (
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Migrator struct {
	conn *pgxpool.Conn
}

func (mig *Migrator) Init(conn *pgxpool.Conn) *Migrator {
	return &Migrator{
		conn: conn,
	}
}

type Schema struct{}

func (mig *Migrator) AutoMigrate(schema interface{}) {
	val := reflect.ValueOf(schema).Elem()
	for i := 0; i < val.NumField(); i++ {
		fmt.Println()
		fieldType := val.Type().Field(i).Type
		fieldName := val.Type().Field(i).Name
		field, _ := reflect.TypeOf(schema).Elem().FieldByName(fieldName)
		fmt.Println(fieldName, fieldType, field.Tag.Get("db"))
	}
}

type User struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	Password string `db:"password"`
}

func main() {

	mig := Migrator{}

	x := &User{}
	mig.AutoMigrate(x)
	// field, ok := reflect.TypeOf(&User{}).Elem().FieldByName("Name")
	// fmt.Println(ok)
	// fmt.Println(field)
}
