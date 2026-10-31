package tstypes

import (
	"reflect"
	"slices"
	"testing"
)

const sample = `
export type Base = {
	id: string;
};

/** A thing. */
export type Thing = Base & {
	name: string;
	/** Said once; "nested: string" in a comment is not a field. */
	label?: string | null;
	// nor is this: string;
	nested: {
		inner: number;
	};
	map: Record<string, string[]>;
	kind:
		| 'a'
		| 'b';
};

export type Picked = Pick<Thing, 'name'>;

export type Inline = { secret: string; uri: string };
`

func TestFieldsOfATypeScriptType(t *testing.T) {
	got, err := Fields(sample, "Thing")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"id", "name", "label", "nested", "map", "kind"}
	if !slices.Equal(got, want) {
		t.Errorf("fields = %v, want %v", got, want)
	}

	if got, _ := Fields(sample, "Inline"); !slices.Equal(got, []string{"secret", "uri"}) {
		t.Errorf("a type on one line has fields %v, want [secret uri]", got)
	}

	if _, err := Fields(sample, "Missing"); err == nil {
		t.Error("a type that is not declared was found")
	}
}

func TestJSONFieldsOfAGoStruct(t *testing.T) {
	type embedded struct {
		ID string `json:"id"`
	}
	type answer struct {
		embedded
		Name    string `json:"name"`
		Label   string `json:"label,omitempty"`
		Secret  string `json:"-"`
		private string
	}

	got := JSONFields(reflect.TypeOf(answer{private: ""}))
	if want := []string{"id", "name", "label"}; !slices.Equal(got, want) {
		t.Errorf("fields = %v, want %v", got, want)
	}
}
