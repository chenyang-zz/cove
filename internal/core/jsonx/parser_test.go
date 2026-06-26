package jsonx_test

import (
	"encoding/json"
	"testing"

	"github.com/boxify/api-go/internal/core/jsonx"
)

type fakeParser struct {
	called bool
}

func (p *fakeParser) Repair(input string) (string, error) {
	return input, nil
}

func (p *fakeParser) Unmarshal(input string, out any) error {
	p.called = true
	data := []byte(`{"score":0.95}`)
	return json.Unmarshal(data, out)
}

func TestGenericDecodeUsesParserContract(t *testing.T) {
	parser := fakeParser{}

	var _ jsonx.Parser = &parser

	got, err := jsonx.Parse[struct {
		Score json.Number `json:"score"`
	}](&parser, `ignored`)
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if got.Score.String() != "0.95" {
		t.Fatalf("score = %s", got.Score.String())
	}
	if !parser.called {
		t.Fatal("parser was not used")
	}
}
