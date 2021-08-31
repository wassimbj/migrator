package queries

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/wassimbj/migrator/utils"
)

type Field struct {
	Name       string
	DataType   string
	Size       string
	DefaultVal string
	IsNullable bool
	IsAutoIncr bool
	Index      string
	IsPk       bool
	IsUnique   bool
	Ref        string // foreign key references
}

// check if table exist
func TableAlreadyExist(conn *pgxpool.Pool, name string) (bool, error) {
	var count int
	err := conn.QueryRow(
		context.Background(),
		"SELECT count(*) FROM information_schema.columns WHERE table_name=$1",
		name,
	).Scan(&count)

	return count > 0, err
}

type ChangedField struct {
	f         Field
	changed   []string
	isChanged bool // if anything changed on this field
	isNew     bool
}

// check and return new and old fields
func GetChangedFields(fields []Field, schema []Field) []ChangedField {
	var changedFields []ChangedField
	var colExist bool = false
	var isChanged bool = false

	// check if fields exist on the database tables
	for _, f := range fields {
		for _, s := range schema {
			if f.Name == s.Name {
				colExist = true
				// check if anything changed to update only the changed fields
				if f.DataType != s.DataType || f.DefaultVal != s.DefaultVal || f.Size != s.Size || f.IsNullable != s.IsNullable {
					isChanged = true
				}
				break
			}
		}
		changedFields = append(changedFields, ChangedField{
			f:         f,
			isChanged: isChanged,
			isNew:     !colExist,
		})
		colExist = false
	}

	return changedFields
}

// add new column
func AddCol(conn *pgxpool.Pool, tbl string, f Field) error {
	sqlStr := fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN %s",
		tbl,
		fmt.Sprintf("%s %s NOT NULL", f.Name, utils.GetDataTypeWithSize(f.DataType, f.Size)),
	)
	err := conn.QueryRow(
		context.Background(),
		sqlStr,
	).Scan()

	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	log.Println(sqlStr)

	return nil
}

// edit column
// https://www.postgresql.org/docs/9.1/sql-altertable.html
func EditCol(conn *pgxpool.Pool, tbl string, f Field) error {
	// cast old values to the new type
	// USING col_name::new_type

	sqlStr := fmt.Sprintf("ALTER TABLE %s \n", tbl)

	// set data type
	sqlStr += fmt.Sprintf(
		"ALTER COLUMN %s TYPE %s, \n",
		f.Name, utils.GetDataTypeWithSize(f.DataType, f.Size),
	)

	// set default value
	if f.DefaultVal != "" {
		sqlStr += fmt.Sprintf(
			"ALTER COLUMN %s SET DEFAULT %s, \n",
			f.Name, f.DefaultVal,
		)
	}

	// set nullable
	if f.IsNullable {
		sqlStr += fmt.Sprintf(
			"ALTER COLUMN %s DROP NOT NULL \n",
			f.Name,
		)
	} else {
		sqlStr += fmt.Sprintf(
			"ALTER COLUMN %s SET NOT NULL \n",
			f.Name,
		)
	}

	err := conn.QueryRow(
		context.Background(),
		sqlStr,
	).Scan()

	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	log.Println(sqlStr)
	return nil
}

// add columns, modify size, type...
// name: table name, fields: field that will be updated
func UpdateTable(conn *pgxpool.Pool, name string, fields []Field) error {

	// get table columns with type and other details
	schema, _ := conn.Query(
		context.Background(), `
		SELECT column_name, udt_name, character_maximum_length, column_default, is_nullable
		FROM information_schema.columns WHERE table_name = $1`, name,
	)
	var schemaInfo struct {
		colName    string
		dataType   string
		size       string
		defaultVal string
		isNull     bool
	}

	var schemaFields []Field
	for schema.Next() {
		val, _ := schema.Values()
		schemaInfo.colName = val[0].(string)
		schemaInfo.dataType = val[1].(string)
		schemaInfo.size = ""
		schemaInfo.defaultVal = ""
		schemaInfo.isNull = val[4] == "YES"

		if val[2] != nil {
			schemaInfo.size = strconv.Itoa(int(val[2].(int32)))
		} else if val[3] != nil {
			schemaInfo.defaultVal = val[3].(string)
		}

		schemaFields = append(schemaFields, Field{
			Name:       schemaInfo.colName,
			DataType:   utils.PgTypeToAlias(schemaInfo.dataType),
			Size:       schemaInfo.size,
			DefaultVal: schemaInfo.defaultVal,
			IsNullable: schemaInfo.isNull,
		})
	}

	for _, cf := range GetChangedFields(fields, schemaFields) {
		if cf.isNew {
			err := AddCol(conn, name, cf.f)
			if err != nil {
				return err
			}
		} else {
			if cf.isChanged {
				err := EditCol(conn, name, cf.f)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// https://www.postgresql.org/docs/9.1/sql-createtable.html
func CreateTable(conn *pgxpool.Pool, name string, fields []Field) error {

	// generate create table sql
	sqlStr := fmt.Sprintf("CREATE TABLE %s ( \n", name)
	for i, field := range fields {
		sqlStr += fmt.Sprintf("%s %s NOT NULL", field.Name, utils.GetDataTypeWithSize(field.DataType, field.Size))

		if i < len(fields)-1 {
			sqlStr += ",\n"
		}
	}
	sqlStr += "\n)"

	err := conn.QueryRow(context.Background(), sqlStr).Scan()

	if err != pgx.ErrNoRows {
		return err
	}

	log.Println(sqlStr)

	return nil

}
