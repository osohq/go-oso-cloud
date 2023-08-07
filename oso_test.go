package oso

import (
	"fmt"
	"testing"
)

type User struct {
	id int
}

func (u User) Instance() Instance {
	return Instance{Type: "User", ID: fmt.Sprint(u.id)}
}

type Repo struct {
	id int
}

func (r Repo) Instance() Instance {
	return Instance{Type: "Repo", ID: fmt.Sprint(r.id)}
}

type Computer struct {
	id int
}

func (c Computer) Instance() Instance {
	return Instance{Type: "Computer", ID: fmt.Sprint(c.id)}
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

	user := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoChild := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoParent := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
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

		e = o.Tell("has_relation", repoParent, String("parent"), repoChild)
		if e != nil {
			t.Fatalf("Tell failed: %v", e)
		}

		e = o.Tell("has_role", user, String("member"), repoChild)
		if e != nil {
			t.Fatalf("Tell failed: %v", e)
		}

		e = o.Tell("has_role", user, String("member"), repoParent)
		if e != nil {
			t.Fatalf("Tell failed: %v", e)
		}
		roles, e := o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 1 || roles[0].Name != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		roles, e = o.Get("has_role", user, Instance{}, repoChild)
		if e != nil || len(roles) != 1 || roles[0].Name != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		roles, e = o.Get("has_role", user, Instance{}, Instance{})
		if e != nil || len(roles) != 2 || roles[0].Name != "has_role" {
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

	user := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoChild := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoParent := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++

	facts := []Fact{
		{
			Name: "has_role",
			Args: []Instance{user, String("member"), repoChild},
		},
		{
			Name: "has_relation",
			Args: []Instance{repoParent, String("parent"), repoChild},
		},
	}

	t.Run("bulk facts", func(t *testing.T) {
		e := o.BulkTell(facts)
		if e != nil {
			t.Fatalf("Bulk tell failed: %v", e)
		}
		roles, e := o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 1 || roles[0].Name != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		relations, e := o.Get("has_relation", repoParent, String("parent"), repoChild)
		if e != nil || len(relations) != 1 || relations[0].Name != "has_relation" {
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

		e = o.Bulk([]Fact{}, facts)
		if e != nil {
			t.Fatalf("Bulk failed: %v", e)
		}
		roles, e = o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 1 || roles[0].Name != "has_role" {
			t.Fatalf("Get roles = %+v, %v, want %d elements with %q predicate", roles, e, 1, "has_role")
		}
		e = o.Bulk(facts, []Fact{})
		if e != nil {
			t.Fatalf("Bulk failed: %v", e)
		}
		roles, e = o.Get("has_role", user, String("member"), repoChild)
		if e != nil || len(roles) != 0 {
			t.Fatalf("Get roles = %+v, %v, want %d elements", roles, e, 0)
		}
	})

	// teardown
	o.BulkDelete(facts)
}

func TestAuthorizeResources(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
			"read" if "member";
			"read" if "read" on "parent";
		}
	`)

	user := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoAcme := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoAnvil := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	repoCoyote := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++

	e := oso.Tell("has_role", user, String("member"), repoAcme)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}
	e = oso.Tell("has_role", user, String("member"), repoAnvil)
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
	e = oso.Delete("has_role", user, String("member"), repoAcme)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}

	e = oso.Delete("has_role", user, String("member"), repoAnvil)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
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

	user := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	acme := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	anvil := Instance{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}

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
					Name: "has_role",
					Args: []Instance{user, String("member"), acme},
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
			result, e := oso.AuthorizeResourcesWithContext(user, "read", []Instance{acme, anvil}, []Fact{
				{
					Name: "has_role",
					Args: []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != acme {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []Instance{acme})
			}
		})
		t.Run("only anvil", func(t *testing.T) {
			result, e := oso.AuthorizeResourcesWithContext(user, "read", []Instance{acme, anvil}, []Fact{
				{
					Name: "has_role",
					Args: []Instance{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 1 || result[0] != anvil {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []Instance{anvil})
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
					Name: "has_role",
					Args: []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != acme.ID {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{acme.ID})
			}
		})
		t.Run("only anvil", func(t *testing.T) {
			result, e := oso.ListWithContext(user, "read", "Repo", []Fact{
				{
					Name: "has_role",
					Args: []Instance{user, String("member"), anvil},
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
					Name: "has_role",
					Args: []Instance{user, String("member"), anvil},
				},
			})
			if e != nil || len(result) != 0 {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{})
			}
		})
		t.Run("context on acme", func(t *testing.T) {
			result, e := oso.ActionsWithContext(user, acme, []Fact{
				{
					Name: "has_role",
					Args: []Instance{user, String("member"), acme},
				},
			})
			if e != nil || len(result) != 1 || result[0] != "read" {
				t.Fatalf("AuthorizeWithContext nil = %v, %v want %v", result, e, []string{"read"})
			}
		})
	})
}

func TestQuery(t *testing.T) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy(`
		actor User {}
		resource Computer {}

		hello(friend) if
			is_friendly(friend);

		something_else(friend, other_friend, _anybody) if
			is_friendly(friend) and is_friendly(other_friend);
	`)
	sam := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	gabe := Instance{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	steve := Instance{Type: "Computer", ID: fmt.Sprintf("%v", idCounter)}

	e := oso.Tell("is_friendly", sam)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}
	e = oso.Tell("is_friendly", gabe)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}
	e = oso.Tell("is_friendly", steve)
	if e != nil {
		t.Fatalf("Tell failed: %v", e)
	}

	t.Run("query", func(t *testing.T) {
		result, e := oso.Query("hello", nil)
		if e != nil || len(result) != 3 || result[0].Name != "hello" {
			t.Fatalf("Query failed, %v", result)
		}
		result, e = oso.Query("hello", &Instance{Type: "User"})
		if e != nil || len(result) != 2 || result[0].Name != "hello" {
			t.Fatalf("Query failed, %v", result)
		}
	})

	// teardown
	e = oso.Delete("is_friendly", sam)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
	e = oso.Delete("is_friendly", gabe)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
	e = oso.Delete("is_friendly", steve)
	if e != nil {
		t.Fatalf("Delete failed: %v", e)
	}
}
