package neo4j

import (
	"reflect"
	"testing"
	"time"
)

type codecExample struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"text"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	Ignored   string    `json:"-"`
}

func TestParamsUsesJSONTagsAndPreservesValues(t *testing.T) {
	createdAt := time.Date(2026, 6, 23, 10, 30, 0, 0, time.UTC)
	params, err := Params(codecExample{
		ID:        "m1",
		UserID:    "u1",
		Text:      "hello",
		Score:     0.8,
		CreatedAt: createdAt,
		Ignored:   "secret",
	})
	if err != nil {
		t.Fatalf("Params error = %v", err)
	}

	want := map[string]any{
		"id":         "m1",
		"user_id":    "u1",
		"text":       "hello",
		"score":      0.8,
		"created_at": createdAt,
	}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("params = %#v, want %#v", params, want)
	}
	if _, ok := params["Ignored"]; ok {
		t.Fatalf("Params included json ignored field: %#v", params)
	}
}

func TestEncodeUsesJSONTagsAndPreservesNeo4jValues(t *testing.T) {
	createdAt := time.Date(2026, 6, 23, 10, 30, 0, 0, time.UTC)
	props, err := Encode(&codecExample{
		ID:        "m1",
		UserID:    "u1",
		Text:      "hello",
		Score:     0.8,
		CreatedAt: createdAt,
		Ignored:   "secret",
	})
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}

	want := map[string]any{
		"id":         "m1",
		"user_id":    "u1",
		"text":       "hello",
		"score":      0.8,
		"created_at": createdAt,
	}
	if !reflect.DeepEqual(props, want) {
		t.Fatalf("props = %#v, want %#v", props, want)
	}
}

func TestEncodeRowsEncodesStructSliceForUnwind(t *testing.T) {
	rows, err := Encode([]codecExample{
		{ID: "m1", UserID: "u1", Text: "one"},
		{ID: "m2", UserID: "u1", Text: "two"},
	})
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}

	want := []any{
		map[string]any{"id": "m1", "user_id": "u1", "text": "one", "score": float64(0), "created_at": time.Time{}},
		map[string]any{"id": "m2", "user_id": "u1", "text": "two", "score": float64(0), "created_at": time.Time{}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestEncodeRejectsUnsupportedScalar(t *testing.T) {
	if _, err := Encode("not-supported"); err == nil {
		t.Fatal("Encode error = nil, want error")
	}
}

func TestParamsClonesMap(t *testing.T) {
	input := map[string]any{"id": "m1"}
	params, err := Params(input)
	if err != nil {
		t.Fatalf("Params error = %v", err)
	}
	params["id"] = "changed"
	if input["id"] != "m1" {
		t.Fatalf("Params mutated input map: %#v", input)
	}
}

func TestDecodeOneFromProjectionMap(t *testing.T) {
	createdAt := time.Date(2026, 6, 23, 10, 30, 0, 0, time.UTC)
	rows := []Row{
		{
			"memory": map[string]any{
				"id":         "m1",
				"user_id":    "u1",
				"text":       "hello",
				"score":      0.8,
				"created_at": createdAt,
			},
		},
	}

	got, err := DecodeOne[codecExample](rows, "memory")
	if err != nil {
		t.Fatalf("DecodeOne error = %v", err)
	}
	want := codecExample{
		ID:        "m1",
		UserID:    "u1",
		Text:      "hello",
		Score:     0.8,
		CreatedAt: createdAt,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestDecodeMany(t *testing.T) {
	rows := []Row{
		{"memory": map[string]any{"id": "m1", "user_id": "u1", "text": "one"}},
		{"memory": map[string]any{"id": "m2", "user_id": "u1", "text": "two"}},
	}

	got, err := DecodeMany[codecExample](rows, "memory")
	if err != nil {
		t.Fatalf("DecodeMany error = %v", err)
	}
	if len(got) != 2 || got[0].ID != "m1" || got[1].ID != "m2" {
		t.Fatalf("got = %#v", got)
	}
}

func TestDecodeRowMissingKeyReturnsError(t *testing.T) {
	if _, err := DecodeRow[codecExample](Row{}, "memory"); err == nil {
		t.Fatal("DecodeRow error = nil, want error")
	}
}

func TestDecodeIntoNilOutputReturnsError(t *testing.T) {
	if err := DecodeInto(map[string]any{"id": "m1"}, nil); err == nil {
		t.Fatal("DecodeInto error = nil, want error")
	}
}
