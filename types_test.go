package oso

import (
	"reflect"
	"testing"
)

func TestBuiltinTypes(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy("declare is_weird(Integer, String, Boolean);")
	e := oso.Bulk([]Fact{}, []Fact{
		{
			Name: "is_weird",
			Args: []Instance{Integer(10), String("yes"), Boolean(true)},
		},
	})
	if e != nil {
		t.Fatalf("Bulk failed: %v", e)
	}
	defer oso.Bulk([]Fact{{Name: "is_weird", Args: []Instance{{}, {}, {}}}}, []Fact{})

	actual, e := oso.Get("is_weird", Integer(10), String("yes"), Boolean(true))
	if e != nil {
		t.Fatalf("Get failed: %v", e)
	}
	expected := []Fact{{
		Name: "is_weird",
		Args: []Instance{
			Instance{Type: "Integer", ID: "10"},
			Instance{Type: "String", ID: "yes"},
			Instance{Type: "Boolean", ID: "true"},
		},
	}}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Got:%v, expected:%v", actual, expected)
	}
}
