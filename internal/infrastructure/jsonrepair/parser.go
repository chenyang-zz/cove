package jsonrepair

import (
	"bytes"
	"encoding/json"
	"errors"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
	corejsonx "github.com/boxify/api-go/internal/core/jsonx"
	"github.com/boxify/api-go/internal/xerr"
)

var _ corejsonx.Parser = &Parser{}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Repair(input string) (string, error) {
	repaired, err := jsonrepair.RepairJSON(input)
	if err != nil {
		return "", xerr.Wrap(err, "repair json failed")
	}
	return repaired, nil
}

func (p *Parser) Unmarshal(input string, out any) error {
	if out == nil {
		return errors.New("json output target is nil")
	}
	repaired, err := p.Repair(input)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(repaired)))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return xerr.Wrap(err, "decode repaired json failed")
	}
	return nil
}
