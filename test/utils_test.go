package test

import (
	"testing"

	"github.com/wassimbj/migrator/utils"
)

func TestCamelToSnakeCase(t *testing.T) {
	result := utils.CamelToSnakeCase("UserPayment", true)
	if result != "user_payments" {
		t.Fail()
	}
}

func TestGetDataTypeWithSize(t *testing.T) {
	result := utils.GetDataTypeWithSize("varchar", "100")
	if result != "varchar(100)" {
		t.Fail()
	}
}

func TestGetSchemaOption(t *testing.T) {
	testData := []string{
		"col:title",
		"type:varchar",
		"size:100",
	}
	result := utils.GetSchemaOption(testData, "col")
	if result != "title" {
		t.Fail()
	}
}
