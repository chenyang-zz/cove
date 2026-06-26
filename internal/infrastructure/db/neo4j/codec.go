package neo4j

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func Decode[T any](value any) (T, error) {
	var out T
	if err := DecodeInto(value, &out); err != nil {
		return out, err
	}
	return out, nil
}

func DecodeInto(value any, out any) error {
	if out == nil {
		return fmt.Errorf("neo4j decode output is nil")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal neo4j value: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal neo4j value: %w", err)
	}
	return nil
}

func DecodeRow[T any](row Row, key string) (T, error) {
	var out T
	value, ok := row[key]
	if !ok {
		return out, fmt.Errorf("neo4j row missing key %q", key)
	}
	return Decode[T](value)
}

func DecodeOne[T any](rows []Row, key string) (T, error) {
	var out T
	if len(rows) == 0 {
		return out, fmt.Errorf("neo4j rows are empty")
	}
	return DecodeRow[T](rows[0], key)
}

func DecodeMany[T any](rows []Row, key string) ([]T, error) {
	out := make([]T, 0, len(rows))
	for i, row := range rows {
		item, err := DecodeRow[T](row, key)
		if err != nil {
			return nil, fmt.Errorf("decode neo4j row %d: %w", i, err)
		}
		out = append(out, item)
	}
	return out, nil
}

func Encode(value any) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("neo4j encode value is nil")
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil, fmt.Errorf("neo4j encode value is nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct && rv.Kind() != reflect.Map && rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("neo4j encode value must be struct, map, slice or array, got %T", value)
	}
	return encodeValue(rv)
}

func EncodeRows(value any) ([]map[string]any, error) {
	if value == nil {
		return nil, fmt.Errorf("neo4j encode rows value is nil")
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, fmt.Errorf("neo4j encode rows value is nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("neo4j encode rows value must be slice or array, got %T", value)
	}
	rows := make([]map[string]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		row, err := encodeMap(rv.Index(i))
		if err != nil {
			return nil, fmt.Errorf("encode neo4j row %d: %w", i, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func Params(value any) (map[string]any, error) {
	if value == nil {
		return nil, fmt.Errorf("neo4j params value is nil")
	}
	return encodeMap(reflect.ValueOf(value))
}

func encodeStruct(rv reflect.Value) (map[string]any, error) {
	rt := rv.Type()
	params := make(map[string]any, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok := paramFieldName(field)
		if !ok {
			continue
		}
		value, err := encodeValue(rv.Field(i))
		if err != nil {
			return nil, fmt.Errorf("encode field %q: %w", name, err)
		}
		params[name] = value
	}
	return params, nil
}

func encodeMap(value reflect.Value) (map[string]any, error) {
	encoded, err := encodeValue(value)
	if err != nil {
		return nil, err
	}
	params, ok := encoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("neo4j encoded value must be map, got %T", encoded)
	}
	return params, nil
}

func encodeValue(value reflect.Value) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	if value.Type() == reflect.TypeOf(time.Time{}) {
		return value.Interface(), nil
	}
	if value.Kind() == reflect.Map {
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string, got %s", value.Type().Key())
		}
		out := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			item, err := encodeValue(iter.Value())
			if err != nil {
				return nil, fmt.Errorf("encode map key %q: %w", iter.Key().String(), err)
			}
			out[iter.Key().String()] = item
		}
		return out, nil
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		out := make([]any, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			item, err := encodeValue(value.Index(i))
			if err != nil {
				return nil, fmt.Errorf("encode index %d: %w", i, err)
			}
			out = append(out, item)
		}
		return out, nil
	}
	if value.Kind() == reflect.Struct {
		return encodeStruct(value)
	}
	switch value.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return value.Interface(), nil
	default:
		return nil, fmt.Errorf("unsupported type %s", value.Type())
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func paramFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	name := strings.Split(tag, ",")[0]
	if name != "" {
		return name, true
	}
	return field.Name, true
}
