package test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/wassimbj/migrator/queries"
)

var dbConn *pgxpool.Pool

func init() {

	dbConn, _ = pgxpool.Connect(context.Background(), "postgres://root:1234@localhost:5432/testdb")

	// drop test data, for testing again
	dbConn.QueryRow(
		context.Background(),
		`ALTER TABLE users DROP COLUMN balance`,
	)

	dbConn.QueryRow(
		context.Background(),
		`DROP TABLE IF EXISTS payments`,
	)

}

func TestTableAlreadyExist(t *testing.T) {
	_, err := queries.TableAlreadyExist(dbConn, "users")

	if err != nil {
		t.Fatal(err)
	}
}

func TestAddCol(t *testing.T) {
	col := queries.Field{
		Name:       "balance",
		DataType:   "int",
		IsNullable: true,
		DefaultVal: "0",
	}
	err := queries.AddCol(dbConn, "users", col)

	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTable(t *testing.T) {
	cols := []queries.Field{
		{
			Name:       "balance",
			DataType:   "numeric", // changed
			Size:       "6,2",     // changed
			IsNullable: true,
			DefaultVal: "0",
		},
	}
	err := queries.UpdateTable(dbConn, "users", cols)

	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTable(t *testing.T) {
	cols := []queries.Field{
		{
			Name:     "id",
			DataType: "int",
		},
		{
			Name:     "card",
			DataType: "varchar",
			Size:     "20",
		},
		{
			Name:       "created_at",
			DataType:   "timestamp",
			DefaultVal: "CURRENT_TIMESTAMP",
		},
	}
	err := queries.CreateTable(dbConn, "payments", cols)

	if err != nil {
		t.Fatal(err)
	}
}
