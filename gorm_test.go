package oso

import (
	"os"
	"reflect"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Environment struct {
	ID     uint
	Tenant uint
	Name   string
}

func (Environment) TableName() string {
	return "environment"
}

func TestLocalDataFiltering(t *testing.T) {
	oso := NewClientWithDataBindings("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn", "../../feature/src/tests/data_bindings/oso_control.yaml")

	postgres := postgres.Open("host=localhost user=oso password=oso dbname=oso_control")
	db, err := gorm.Open(postgres)
	if err != nil {
		t.Fatal(err)
	}

	alice := Value{Type: "User", ID: "alice"}
	bob := Value{Type: "User", ID: "bob"}
	tenant := Value{Type: "Tenant", ID: "1"}
	environment1 := Value{Type: "Environment", ID: "1"}
	environment2 := Value{Type: "Environment", ID: "2"}
	environmentAny := Value{Type: "Environment", ID: "12345"}

	// setup
	policy, err := os.ReadFile("../../feature/src/tests/policies/oso_control.polar")
	if err != nil {
		t.Fatal(err)
	}
	err = oso.Policy(string(policy))
	if err != nil {
		t.Fatal(err)
	}
	err = oso.Insert(NewFact("has_role", alice, String("member"), tenant))
	if err != nil {
		t.Fatal(err)
	}
	err = oso.Insert(NewFact("is_god", bob))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("basic list filtering, all rows", func(t *testing.T) {
		filter, err := oso.ListLocal(alice, "read", "Environment", "id")
		if err != nil {
			t.Fatal(err)
		}
		var environments []Environment
		db.Find(&environments, filter)
		if len(environments) != 5 {
			t.Fatalf("expected 5 environments, got %d", len(environments))
		}
	})

	t.Run("basic list filtering, some rows", func(t *testing.T) {
		filter, err := oso.ListLocal(alice, "create_fact", "Environment", "id")
		if err != nil {
			t.Fatal(err)
		}
		var environments []Environment
		db.Find(&environments, filter)
		if len(environments) != 3 {
			t.Fatalf("expected 3 environments, got %d", len(environments))
		}
	})

	t.Run("list filtering, wildcard", func(t *testing.T) {
		filter, err := oso.ListLocal(bob, "read", "Environment", "id")
		if err != nil {
			t.Fatal(err)
		}
		var environments []Environment
		db.Find(&environments, filter)
		if len(environments) != 5 {
			t.Fatalf("expected 5 environments, got %d", len(environments))
		}
	})

	t.Run("list filtering, no rows", func(t *testing.T) {
		filter, err := oso.ListLocal(alice, "frob", "Environment", "id")
		if err != nil {
			t.Fatal(err)
		}
		var environments []Environment
		db.Find(&environments, filter)
		if len(environments) != 0 {
			t.Fatalf("expected 0 environments, got %d", len(environments))
		}
	})

	t.Run("basic authorize, allowed", func(t *testing.T) {
		query, err := oso.AuthorizeLocal(alice, "create_fact", environment1)
		if err != nil {
			t.Fatal(err)
		}
		var authorizeResult AuthorizeResult
		db.Raw(query).Scan(&authorizeResult)
		if !authorizeResult.Allowed {
			t.Fatalf("expected allowed, got %t", authorizeResult.Allowed)
		}
	})

	t.Run("basic authorize, denied", func(t *testing.T) {
		query, err := oso.AuthorizeLocal(alice, "create_fact", environment2)
		if err != nil {
			t.Fatal(err)
		}
		var authorizeResult AuthorizeResult
		db.Raw(query).Scan(&authorizeResult)
		if authorizeResult.Allowed {
			t.Fatalf("expected denied, got %t", authorizeResult.Allowed)
		}
	})

	t.Run("authorize wildcard allowed", func(t *testing.T) {
		query, err := oso.AuthorizeLocal(bob, "read", environmentAny)
		if err != nil {
			t.Fatal(err)
		}
		var authorizeResult AuthorizeResult
		db.Raw(query).Scan(&authorizeResult)
		if !authorizeResult.Allowed {
			t.Fatalf("expected allowed, got %t", authorizeResult.Allowed)
		}
	})

	t.Run("authorize always denied", func(t *testing.T) {
		query, err := oso.AuthorizeLocal(bob, "frob", environmentAny)
		if err != nil {
			t.Fatal(err)
		}
		var authorizeResult AuthorizeResult
		db.Raw(query).Scan(&authorizeResult)
		if authorizeResult.Allowed {
			t.Fatalf("expected denied, got %t", authorizeResult.Allowed)
		}
	})

	t.Run("actions", func(t *testing.T) {
		query, err := oso.ActionsLocal(bob, environmentAny)
		if err != nil {
			t.Fatal(err)
		}
		var actions []string
		db.Raw(query).Pluck("actions", &actions)
		if !reflect.DeepEqual(actions, []string{"read"}) {
			t.Fatalf("expected [read], got %v", actions)
		}
	})

	// teardown
	err = oso.Delete(NewFactPattern("has_role", alice, String("member"), tenant))
	if err != nil {
		t.Fatal(err)
	}
	err = oso.Delete(NewFactPattern("is_god", bob))
	if err != nil {
		t.Fatal(err)
	}
}
