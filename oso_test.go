package oso

import (
	"fmt"
	"testing"
)

type User struct {
	id int
}

func (u User) Id() string {
	return fmt.Sprint(u.id)
}

func (u User) Type() string {
	return "User"
}

type Repo struct {
	id int
}

func (r Repo) Id() string {
	return fmt.Sprint(r.id)
}

func (r Repo) Type() string {
	return "Repo"
}

var idCounter = 1

func TestEverything(t *testing.T) {
	o := NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
      		"read" if "member";
		}
	`)

	user := User{id: idCounter}
	idCounter++
	repoChild := Repo{id: idCounter}
	idCounter++
	repoParent := Repo{id: idCounter}
	idCounter++

	t.Run("everything", func(t *testing.T) {
		allowed, e := o.Authorize(user, "read", repoChild)
		if e != nil || allowed != false {
			t.Fatalf("Authorize = %t, %v, want %t", allowed, e, false)
		}

		results, e := o.List(user, "read", Repo{}, nil)
		if e != nil || len(results) != 0 {
			t.Fatalf("List = %v, %v, want %v", results, e, []string{})
		}

		e = o.Tell("has_relation", repoParent, String("parent"), repoChild)
		if e != nil {
			t.Fatalf("Tell failed: %v", e)
		}

		e = o.Tell("has_role", user, String("member"), repoChild)
		if e != nil {
			t.Fatalf("Tell failed: %v", e)
		}

		roles, e := o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}

		allowed_again, e := o.Authorize(user, "read", repoChild)
		if e != nil || allowed_again != true {
			t.Fatalf("Authorize = %t, %v, want %t", allowed_again, e, true)
		}

		actions, e := o.Actions(user, repoChild)
		if e != nil || len(actions) != 1 || actions[0] != "read" {
			t.Fatalf("Actions = %v, %v, want %v", actions, e, []string{"read"})
		}
	})

	// teardown
	e := o.Delete("has_role", user, String("member"), repoChild)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}

	e = o.Delete("has_relation", repoParent, String("parent"), repoChild)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
}

func TestBulkFacts(t *testing.T) {
	o := NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
      		"read" if "member";
		}
	`)

	user := User{id: idCounter}
	idCounter++
	repoChild := Repo{id: idCounter}
	idCounter++
	repoParent := Repo{id: idCounter}
	idCounter++

	facts := []BulkFact{
		{
			Predicate: "has_role",
			Args:      []Instance{user, String("member"), repoChild},
		},
		{
			Predicate: "has_relation",
			Args:      []Instance{repoParent, String("parent"), repoChild},
		},
	}

	t.Run("bulk facts", func(t *testing.T) {
		e := o.BulkTell(facts)
		if e != nil {
			t.Fatalf("Bulk tell failed: %v", e)
		}
		roles, e := o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		relations, e := o.Get("has_relation", repoParent, String("parent"), repoChild)
		if e != nil || len(relations) != 1 || relations[0].Predicate != "has_relation" {
			t.Fatalf("Get relations = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_relation")
		}

		e = o.BulkDelete(facts)
		if e != nil {
			t.Fatalf("Bulk delete failed: %v", e)
		}
		roles, e = o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 0 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", roles, e, 0)
		}
		relations, e = o.Get("has_relation", repoParent, String("parent"), repoChild)
		if e != nil || len(relations) != 0 {
			t.Fatalf("Get relations = %+v, %v, want %d elements", roles, e, 0)
		}
	})

	// teardown
	o.BulkDelete(facts)
}

func TestAuthorizeResources(t *testing.T) {
	oso := NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	oso.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
			"read" if "member";
		}
	`)

	user := User{id: idCounter}
	idCounter++
	repoAcme := Repo{id: idCounter}
	idCounter++
	repoAnvil := Repo{id: idCounter}
	idCounter++
	repoCoyote := Repo{id: idCounter}
	idCounter++

	e := oso.Tell("has_role", user, oso.String("member"), repoAcme)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}
	e = oso.Tell("has_role", user, oso.String("member"), repoAnvil)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}

	t.Run("authorize_resources", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			results, e := oso.AuthorizeResources(user, "read", []Instance{})
			if e != nil || len(results) != 0 {
				t.Fatalf("AuthorizeResources = %v, %v, want %v", results, e, []Instance{})
			}
			results, e = oso.AuthorizeResources(user, "read", nil)
			if e != nil || len(results) != 0 {
				t.Fatalf("AuthorizeResources = %v, %v, want %v", results, e, []Instance{})
			}
		})
		t.Run("match all", func(t *testing.T) {
			results, e := oso.AuthorizeResources(user, "read", []Instance{repoAcme, repoAnvil})
			expected := []Instance{repoAcme, repoAnvil}
			if e != nil || len(results) != len(expected) {
				t.Fatalf("AuthorizeResources = %v, %v, want %v", results, e, expected)
			}
		})
		t.Run("match some", func(t *testing.T) {
			results, e := oso.AuthorizeResources(user, "read", []Instance{repoAcme, repoCoyote})
			expected := []Instance{repoAcme}
			if e != nil || len(results) != len(expected) {
				t.Fatalf("AuthorizeResources = %v, %v, want %v", results, e, expected)
			}
		})
		t.Run("match none", func(t *testing.T) {
			results, e := oso.AuthorizeResources(user, "read", []Instance{repoCoyote})
			if e != nil || len(results) != 0 {
				t.Fatalf("AuthorizeResources = %v, %v, want %v", results, e, []Instance{})
			}
		})
	})

	// teardown
	e = oso.Delete("has_role", user, oso.String("member"), repoAcme)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}

	e = oso.Delete("has_role", user, oso.String("member"), repoAnvil)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
}

func TestContextFacts(t *testing.T) {
	oso := NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	oso.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
			"read" if "member";
		}
	`)

	user := User{id: idCounter}
	idCounter++
	acme := Repo{id: idCounter}
	idCounter++
	anvil := Repo{id: idCounter}

	t.Run("authorize", func(t *testing.T) {
		t.Run("nil context", func(t *testing.T) {
			result, e := oso.AuthorizeWithContext(user, "read", acme, nil)
			if e != nil || result != false {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, false)
			}
		})
		t.Run("with context", func(t *testing.T) {
			result, e := oso.AuthorizeWithContext(user, "read", acme, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), acme},
				},
			})
			if e != nil || result != true {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, true)
			}
		})
	})

	t.Run("authorize resources", func(t *testing.T) {
		t.Run("neither acme nor anvil", func(t *testing.T) {
			result, e := oso.AuthorizeResourcesWithContext(user, "read", []Instance{acme, anvil}, nil)
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []Instance{})
			}
		})
		t.Run("only acme", func(t *testing.T) {
			result, e := oso.AuthorizeResourcesWithContext(user, "read", []Instance{acme, anvil}, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != acme {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []Instance{acme})
			}
		})
		t.Run("only anvil", func(t *testing.T) {
			result, e := oso.AuthorizeResourcesWithContext(user, "read", []Instance{acme, anvil}, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 1 || result[0] != anvil {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []Instance{anvil})
			}
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("neither acme nor anvil", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", Repo{}, nil)
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("only acme", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", Repo{}, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != acme.Id() {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{acme.Id()})
			}
		})
		t.Run("only anvil", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", Repo{}, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 1 || result[0] != anvil.Id() {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{anvil.Id()})
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
			result, e := oso.ActionsWithContext(user, acme, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("context on acme", func(t *testing.T) {
			result, e := oso.ActionsWithContext(user, acme, []BulkFact{
				{
					Predicate: "has_role",
					Args:      []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != "read" {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{"read"})
			}
		})
	})
}
