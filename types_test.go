package oso

import (
	"reflect"
	"testing"
)

func TestBuiltinTypes(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy("declare is_weird(Integer, String, Boolean);")
	e := oso.Insert(Fact{
		Predicate: "is_weird",
		Args:      []Value{Integer(10), String("yes"), Boolean(true)},
	})
	if e != nil {
		t.Fatalf("Bulk failed: %v", e)
	}
	defer oso.Delete(NewFactPattern("is_weird", nil, nil, nil))

	actual, e := oso.Get(NewFactPattern("is_weird", Integer(10), String("yes"), Boolean(true)))
	if e != nil {
		t.Fatalf("Get failed: %v", e)
	}
	expected := []Fact{
		NewFact("is_weird", NewValue("Integer", "10"), NewValue("String", "yes"), NewValue("Boolean", "true")),
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Got:%v, expected:%v", actual, expected)
	}
}
