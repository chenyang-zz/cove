package jsonrepair_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/core/jsonx"
	infrajsonrepair "github.com/boxify/api-go/internal/infrastructure/jsonrepair"
)

func TestParserImplementsCoreContract(t *testing.T) {
	var _ jsonx.Parser = infrajsonrepair.NewParser()
}

func TestUnmarshalParsesValidJSON(t *testing.T) {
	parser := infrajsonrepair.NewParser()

	var out struct {
		Name string      `json:"name"`
		Age  json.Number `json:"age"`
	}
	if err := parser.Unmarshal(`{"name":"boxify","age":18}`, &out); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if out.Name != "boxify" || out.Age.String() != "18" {
		t.Fatalf("out = %#v", out)
	}
}

func TestUnmarshalRepairsLLMMarkdownAndSingleQuotes(t *testing.T) {
	parser := infrajsonrepair.NewParser()

	var out struct {
		Employees []string `json:"employees"`
	}
	err := parser.Unmarshal("```json {'employees':['John', 'Anna', ```", &out)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if strings.Join(out.Employees, ",") != "John,Anna" {
		t.Fatalf("employees = %#v", out.Employees)
	}
}

func TestUnmarshalRepairsUnclosedObjectAndArray(t *testing.T) {
	parser := infrajsonrepair.NewParser()

	var out struct {
		Items []string `json:"items"`
	}
	if err := parser.Unmarshal(`{"items":["a","b"`, &out); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if strings.Join(out.Items, ",") != "a,b" {
		t.Fatalf("items = %#v", out.Items)
	}
}

func TestUnmarshalReturnsErrorForNilOutput(t *testing.T) {
	parser := infrajsonrepair.NewParser()

	if err := parser.Unmarshal(`{"ok":true}`, nil); err == nil {
		t.Fatal("Unmarshal error = nil, want error")
	}
}

func TestUnmarshalReturnsMappingError(t *testing.T) {
	parser := infrajsonrepair.NewParser()

	var out struct {
		Age int `json:"age"`
	}
	if err := parser.Unmarshal(`{"age":"not-a-number"}`, &out); err == nil {
		t.Fatal("Unmarshal error = nil, want mapping error")
	}
}
