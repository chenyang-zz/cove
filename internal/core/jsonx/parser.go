package jsonx

import "github.com/boxify/api-go/internal/xerr"

type Parser interface {
	Repair(input string) (string, error)
	Unmarshal(input string, out any) error
}

func Parse[T any](parser Parser, input string) (T, error) {
	var out T
	if err := parser.Unmarshal(input, &out); err != nil {
		return out, xerr.Wrapf(err, "parse json failed: %v", err)
	}
	return out, nil
}
