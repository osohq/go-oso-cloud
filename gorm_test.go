package oso

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
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

func TestListFiltering(t *testing.T) {
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

	t.Run("list filtering with context facts", func(t *testing.T) {
		charles := NewValue("User", "charles")
		filter, err := oso.ListLocalWithContext(charles, "read", "Environment", "id", []Fact{
			NewFact("is_god", charles),
		})
		if err != nil {
			t.Fatal(err)
		}
		var environments []Environment
		db.Find(&environments, filter)
		if len(environments) != 5 { // all of em
			t.Fatalf("expected 5 environments, got %d", len(environments))
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

	t.Run("authorize with context facts", func(t *testing.T) {
		charles := NewValue("User", "charles")
		query, err := oso.AuthorizeLocalWithContext(charles, "read", environment1, []Fact{
			NewFact("is_god", charles),
		})
		if err != nil {
			t.Fatal(err)
		}
		var authorizeResult AuthorizeResult
		db.Raw(query).Scan(&authorizeResult)
		if !authorizeResult.Allowed {
			t.Fatalf("expected allowed, got %t", authorizeResult.Allowed)
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

	t.Run("actions with context facts", func(t *testing.T) {
		charles := NewValue("User", "charles")
		query, err := oso.ActionsLocalWithContext(charles, environment1, []Fact{
			NewFact("is_god", charles),
		})
		if err != nil {
			t.Fatal(err)
		}
		var actions []string
		db.Raw(query).Pluck("actions", &actions)
		if !reflect.DeepEqual(actions, []string{"read"}) {
			t.Fatalf("expected [read], got %v", actions)
		}
	})

	t.Run("query select", func(t *testing.T) {
		userVar := TypedVar("User")
		env1 := NewValue("Environment", "1")
		query, err := oso.BuildQuery(NewQueryFact("allow", userVar, String("read"), env1)).WithContextFacts(
			[]Fact{
				NewFact("has_permission", NewValue("User", "dartagnan"), String("read"), env1),
			},
		).EvaluateLocalSelect(map[string]Variable{"user_id": userVar})
		if err != nil {
			t.Fatal(err)
		}
		var userIds []string
		err = db.Raw(query).Pluck("user_id", &userIds).Error
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(userIds)
		if !reflect.DeepEqual(userIds, []string{"alice", "bob", "dartagnan"}) {
			t.Fatalf("expected [alice, bob, dartagnan], got %v", userIds)
		}
	})

	t.Run("query select with no projections", func(t *testing.T) {
		userVar := TypedVar("User")
		env1 := NewValue("Environment", "1")
		query, err := oso.BuildQuery(NewQueryFact("allow", userVar, String("read"), env1)).WithContextFacts(
			[]Fact{
				NewFact("has_permission", NewValue("User", "dartagnan"), String("read"), env1),
			},
		).EvaluateLocalSelect(map[string]Variable{})
		if err != nil {
			t.Fatal(err)
		}
		var results []bool
		err = db.Raw(query).Pluck("result", &results).Error
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(results, []bool{true}) {
			t.Fatalf("expected [true], got %v", results)
		}
	})

	t.Run("query select errors with duplicate variables", func(t *testing.T) {
		userVar := TypedVar("User")
		env1 := NewValue("Environment", "1")
		_, err := oso.BuildQuery(
			NewQueryFact("allow", userVar, String("read"), env1),
		).EvaluateLocalSelect(map[string]Variable{"user_id": userVar, "another_user_id": userVar})
		if err == nil || !strings.Contains(err.Error(), "duplicated User variable") {
			t.Fatal("Expected 'duplicated User variable' error, got none")
		}
	})

	t.Run("query filter", func(t *testing.T) {
		userVar := TypedVar("User")
		env1 := NewValue("Environment", "1")
		filter, err := oso.BuildQuery(NewQueryFact("allow", userVar, String("read"), env1)).WithContextFacts(
			[]Fact{
				NewFact("has_permission", NewValue("User", "dartagnan"), String("read"), env1),
			},
		).EvaluateLocalFilter("users.user_id", userVar)
		if err != nil {
			t.Fatal(err)
		}

		var userIds []string
		query := fmt.Sprintf("SELECT user_id FROM (values ('alice'), ('bob'), ('charlie'), ('dartagnan')) as users(user_id) where %s order by user_id asc", filter)
		err = db.Raw(query).Pluck("user_id", &userIds).Error
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(userIds, []string{"alice", "bob", "dartagnan"}) {
			t.Fatalf("expected [alice, bob, dartagnan], got %v", userIds)
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
