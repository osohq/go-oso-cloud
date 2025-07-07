package oso

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type User struct {
	id int
}

func (u User) Value() Value {
	return Value{Type: "User", ID: fmt.Sprint(u.id)}
}

type Repo struct {
	id int
}

func (r Repo) Value() Value {
	return Value{Type: "Repo", ID: fmt.Sprint(r.id)}
}

type Computer struct {
	id int
}

func (c Computer) Value() Value {
	return Value{Type: "Computer", ID: fmt.Sprint(c.id)}
}

var idCounter = 1

func TestEverything(t *testing.T) {
	o := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
      		"read" if "member";
			"read" if "read" on "parent";
		}
	`)

	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoChild := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoParent := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++

	t.Run("everything", func(t *testing.T) {
		allowed, e := o.Authorize(user, "read", repoChild)
		if e != nil || allowed != false {
			t.Fatalf("Authorize = %t, %v, want %t", allowed, e, false)
		}

		results, e := o.List(user, "read", "Repo", nil)
		if e != nil || len(results) != 0 {
			t.Fatalf("List = %v, %v, want %v", results, e, []string{})
		}

		e = o.Insert(NewFact("has_relation", repoParent, String("parent"), repoChild))
		if e != nil {
			t.Fatalf("Insert failed: %v", e)
		}

		// bad facts
		e = o.Insert(NewFact("has_role", user, String("member"), repoChild))
		if e != nil {
			t.Fatalf("Insert failed: %v", e)
		}
		e = o.Insert(NewFact("has_role", user, String("member"), repoParent))
		if e != nil {
			t.Fatalf("Insert failed: %v", e)
		}

		e = o.Insert(NewFact("has_role", NewValue("", "bad"), String("member"), repoParent))
		if e == nil {
			t.Fatalf("Expected failure for bad type, got: %v", e)
		}
		e = o.Insert(NewFact("has_role", NewValue("User", ""), String("member"), repoParent))
		if e == nil {
			t.Fatalf("Expected failure for bad id, got: %v", e)
		}

		roles, e := o.Get(NewFactPattern("has_role", user, String("member"), repoChild))
		if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		roles, e = o.Get(NewFactPattern("has_role", user, nil, repoChild))
		if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		roles, e = o.Get(NewFactPattern("has_role", user, nil, nil))
		if e != nil || len(roles) != 2 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}

		facts, e := o.Get(NewFactPattern("has_role", nil, nil, nil))
		if e != nil || len(facts) != 2 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", facts, e, 3)
		}

		allowedAgain, e := o.Authorize(user, "read", repoChild)
		if e != nil || allowedAgain != true {
			t.Fatalf("Authorize = %t, %v, want %t", allowedAgain, e, true)
		}

		actions, e := o.Actions(user, repoChild)
		if e != nil || len(actions) != 1 || actions[0] != "read" {
			t.Fatalf("Actions = %v, %v, want %v", actions, e, []string{"read"})
		}
	})

	// teardown
	e := o.Delete(NewFact("has_role", user, String("member"), repoChild))
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}

	e = o.Delete(NewFact("has_role", user, String("member"), repoParent))
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}

	e = o.Delete(NewFact("has_relation", repoParent, String("parent"), repoChild))
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
}

func TestAPIError(t *testing.T) {
	o := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	e := o.Insert(NewFact("does_not_exist", user, String("taco")))
	if e == nil || !strings.HasPrefix(e.Error(), "Oso Cloud error: ") {
		t.Fatalf("Invalid API request had unexpected result: %v", e)
	}
}

func TestRequestBodyTooBig(t *testing.T) {
	o := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	e := o.Insert(NewFact("has_role", user, String(strings.Repeat("a", 10*1024*1024))))
	if e == nil || !strings.HasPrefix(e.Error(), "request payload too large") {
		t.Fatalf("Invalid API request had unexpected result: %v", e)
	}
}

func TestBatch(t *testing.T) {
	o := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member", "owner"];
			permissions = ["read"];
			relations = { parent: Repo };
			"read" if "member";
			"read" if "read" on "parent";
			"read" if "owner";
		}

		resource Issue {
			roles = ["member", "owner"];
			permissions = ["read"];
			"read" if "member";
			"read" if "owner";
		}
	`)

	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoChild := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoParent := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++

	facts := []Fact{
		NewFact("has_role", user, String("member"), repoChild),
		NewFact("has_role", user, String("owner"), repoChild),
		NewFact("has_relation", repoParent, String("parent"), repoChild),
	}

	t.Run("batch", func(t *testing.T) {
		e := o.Batch(func(tx BatchTransaction) {
			for _, fact := range facts {
				tx.Insert(fact)
			}
		})
		if e != nil {
			t.Fatalf("Batch insert failed: %v", e)
		}

		roles, e := o.Get(NewFactPattern("has_role", user, String("member"), repoChild))
		if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		relations, e := o.Get(NewFactPattern("has_relation", repoParent, String("parent"), repoChild))
		if e != nil || len(relations) != 1 || relations[0].Predicate != "has_relation" {
			t.Fatalf("Get relations = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_relation")
		}

		e = o.Batch(func(tx BatchTransaction) {
			tx.Delete(NewFact("has_role", user, String("owner"), repoChild))
			tx.Delete(NewFact("has_role", user, String("member"), repoChild))
			tx.Delete(NewFactPattern("has_relation", nil, nil, nil))
		})
		if e != nil {
			t.Fatalf("Batch delete failed: %v", e)
		}
		roles, e = o.Get(NewFactPattern("has_role", user, String("member"), repoChild))
		if e != nil || len(roles) != 0 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", roles, e, 0)
		}
		roles, e = o.Get(NewFactPattern("has_role", user, String("owner"), repoChild))
		if e != nil || len(roles) != 0 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", roles, e, 0)
		}
		relations, e = o.Get(NewFactPattern("has_relation", repoParent, String("parent"), repoChild))
		if e != nil || len(relations) != 0 {
			t.Fatalf("Get relations = %+v, %v, want %d elements", roles, e, 0)
		}

		e = o.Batch(func(tx BatchTransaction) {
			tx.Insert(NewFact("has_role", NewValue("User", "1"), String("member"), NewValue("Repo", "1")))
			tx.Insert(NewFact("has_role", NewValue("User", "2"), String("owner"), NewValue("Repo", "2")))
			tx.Insert(NewFact("has_role", NewValue("User", "1"), String("member"), NewValue("Issue", "1")))
			tx.Insert(NewFact("has_role", NewValue("User", "2"), String("owner"), NewValue("Issue", "2")))
			tx.Delete(NewFactPattern("has_role", nil, String("member"), nil))
			tx.Delete(NewFactPattern("has_role", nil, nil, NewValueOfType("Issue")))
		})
		roles, e = o.Get(NewFactPattern("has_role", nil, nil, nil))
		if e != nil || len(roles) != 1 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", roles, e, 0)
		}
		if roles[0].Args[0] != NewValue("User", "2") {
			t.Fatalf("Expected has_role(User{1}, \"owner\", Repo{2}), got %v", roles[0])
		}

		e = o.Batch(func(tx BatchTransaction) {
			tx.Insert(
				NewFact(
					"has_role",
					NewValue("", "1"), 		// Empty Type
					String("member"),
					NewValue("Repo", "1"),
				),
			)
		})
		if e == nil {
			t.Fatalf("Expected bad id error got %e\n", e)
		}

		e = o.Batch(func(tx BatchTransaction) {
			tx.Insert(
				NewFact(
					"has_role",
					NewValue("User", ""), 		// Empty ID
					String("member"),
					NewValue("Repo", "1"),
				),
			)
		})
		if e == nil {
			t.Fatalf("Expected bad id error got %e\n", e)
		}
	})

	// teardown
	o.Delete(NewFactPattern("has_relation", nil, nil, nil))
	o.Delete(NewFactPattern("has_role", nil, nil, nil))
}

func TestContextFacts(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
			"read" if "member";
		}
	`)

	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	acme := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	anvil := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}

	t.Run("authorize", func(t *testing.T) {
		t.Run("nil context", func(t *testing.T) {
			result, e := oso.AuthorizeWithContext(user, "read", acme, nil)
			if e != nil || result != false {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, false)
			}
		})
		t.Run("with context", func(t *testing.T) {
			result, e := oso.AuthorizeWithContext(user, "read", acme, []Fact{
				{
					Predicate: "has_role",
					Args:      []Value{user, String("member"), acme},
				},
			})
			if e != nil || result != true {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, true)
			}
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("neither acme nor anvil", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", "Repo", nil)
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("only acme", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", "Repo", []Fact{
				{
					Predicate: "has_role",
					Args:      []Value{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != acme.ID {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{acme.ID})
			}
		})
		t.Run("only anvil", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", "Repo", []Fact{
				{
					Predicate: "has_role",
					Args:      []Value{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 1 || result[0] != anvil.ID {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{anvil.ID})
			}
		})
	})

	t.Run("actions", func(t *testing.T) {
		t.Run("no context", func(t *testing.T) {
			result, e := oso.ActionsWithContext(user, acme, nil)
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("context on wrong object", func(t *testing.T) {
			result, e := oso.ActionsWithContext(user, acme, []Fact{
				{
					Predicate: "has_role",
					Args:      []Value{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("context on acme", func(t *testing.T) {
			result, e := oso.ActionsWithContext(user, acme, []Fact{
				{
					Predicate: "has_role",
					Args:      []Value{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != "read" {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{"read"})
			}
		})
	})
}

func TestPolicyMetadata(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy(`
	actor User { }

	resource Organization {
		roles = ["admin", "member"];
		permissions = [
			"read", "add_member", "repository.create",
		];

		# role hierarchy:
		# admins inherit all member permissions
		"member" if "admin";

		# org-level permissions
		"read" if "member";
		"add_member" if "admin";
		# permission to create a repository
		# in the organization
		"repository.create" if "admin";
	}

	resource Repository {
		permissions = ["read", "delete"];
		roles = ["member", "admin"];
		relations = {
			organization: Organization,
		};

		# inherit all roles from the organization
		role if role on "organization";

		# admins inherit all member permissions
		"member" if "admin";

		"read" if "member";
		"delete" if "admin";
	}
	`)

	t.Run("GetPolicyMetadata", func(t *testing.T) {
		result, e := oso.GetPolicyMetadata()
		if e != nil {
			t.Fatalf("Query failed, %s", e)
		}
		expected := PolicyMetadata{
			Resources: map[string]ResourceMetadata{
				"Organization": {
					Permissions: []string{
						"add_member",
						"read",
						"repository.create",
					},
					Roles:     []string{"admin", "member"},
					Relations: map[string]string{},
				},
				"Repository": {
					Permissions: []string{
						"delete",
						"read",
					},
					Roles: []string{
						"admin",
						"member",
					},
					Relations: map[string]string{
						"organization": "Organization",
					},
				},
				"User": {
					Permissions: []string{},
					Roles:       []string{},
					Relations:   map[string]string{},
				},
				"global": {
					Permissions: []string{},
					Roles:       []string{},
					Relations:   map[string]string{},
				},
			},
		}

		if !reflect.DeepEqual(*result, expected) {
			t.Fatalf("GetPolicyMetadata failed,\ngot:\n%v\nexpected:\n%v", *result, expected)
		}
	})
}
