package main

import (
	"reflect"
	"strings"

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

/*
TODO: Add Auto Increment field
- for new cols
CREATE SEQUENCE tablename_colname_seq;
CREATE TABLE tablename (
    colname integer NOT NULL DEFAULT nextval('tablename_colname_seq')
);
ALTER SEQUENCE tablename_colname_seq OWNED BY tablename.colname;

- for existing columns
CREATE SEQUENCE my_serial AS integer START 1 OWNED BY address.new_id;
ALTER TABLE address ALTER COLUMN new_id SET DEFAULT nextval('my_serial');


TODO: Add indexes
-- CREATE INDEX "users_email" ON "users" ("email");

TODO: Add Primary keys

TODO: Add Unique

TODO: Add Foreign  Keys
-- ALTER TABLE "posts"
	ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id")
*/
func (m *Migrator) Migrate(schema interface{}) {
	elem := reflect.ValueOf(schema).Elem()
	tableName := reflect.TypeOf(schema).Elem().Name()
	var fieldNames []queries.Field

	for i := 0; i < elem.NumField(); i++ {
		fieldName := elem.Type().Field(i).Name
		field, _ := reflect.TypeOf(schema).Elem().FieldByName(fieldName)
		dbTags := strings.Split(field.Tag.Get("db"), ";")

		colName := utils.GetSchemaOption(dbTags, "col")
		colSize := utils.GetSchemaOption(dbTags, "size")
		colType := utils.GetSchemaOption(dbTags, "type")
		autoIncr := utils.GetSchemaOption(dbTags, "autoIncr")
		isNull := utils.GetSchemaOption(dbTags, "null")
		idx := utils.GetSchemaOption(dbTags, "index")
		if idx == "true" {
			idx = "idx_" + colName
		}
		pk := utils.GetSchemaOption(dbTags, "primaryKey")
		ref := utils.GetSchemaOption(dbTags, "ref")

		// ref:table_name,col_name
		fieldNames = append(fieldNames, queries.Field{
			Name:       colName,
			DataType:   colType,
			Size:       colSize,
			IsNullable: isNull == "true",
			IsAutoIncr: autoIncr == "true",
			Index:      idx,
			IsPk:       pk == "true",
			Ref:        ref,
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
