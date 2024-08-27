package oso

import (
	"reflect"
	"testing"
)

func setupClient() OsoClientImpl {
	o := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn").(OsoClientImpl)
	o.Policy(`
		global {
		  permissions = ["create_repository"];
		  roles = ["user"];

		  "create_repository" if "user";
		}

		has_role(_: User, "user");

		actor User {
		}

		resource Org {
		  roles = ["member", "admin"];
		  permissions = ["read", "delete"];

		  "read" if "member";
		  "read" if global "user";
		  "delete" if "admin";
		}

		resource Repo {
		  roles = ["member", "admin"];
		  permissions = ["read", "write"];
		  relations = { parent: Org };

		  role if role on "parent";

		  "read" if "member";
		  "write" if "admin";

		  "member" if "admin";
		}

		resource Field {}

		allow_field(user: User, "read", repo: Repo, Field{"name"}) if
		  has_permission(user, "read", repo);
	`)

	alice := NewValue("User", "alice")
	bob := NewValue("User", "bob")
	acme := NewValue("Org", "acme")
	anvil := NewValue("Repo", "anvil")

	o.Batch(func(tx BatchTransaction) {
		tx.Insert(NewFact("has_role", alice, String("member"), acme))
		tx.Insert(NewFact("has_role", bob, String("admin"), acme))
		tx.Insert(NewFact("has_relation", anvil, String("parent"), acme))
	})
	return o
}

func teardown(o OsoClientImpl) {
	o.Batch(func(tx BatchTransaction) {
		tx.Delete(NewFactPattern("has_role", nil, nil, nil))
		tx.Delete(NewFactPattern("has_relation", nil, nil, nil))
	})
}

func TestFieldLevel(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "alice")
	action := String("read")
	resource := NewValue("Repo", "anvil")
	field := TypedVar("Field")

	qb := o.BuildQuery(NewQueryFact("allow_field", actor, action, resource, field))
	expected := []string{"name"}

	// EvaluateValues
	{
		result, err := qb.EvaluateValues(field)
		if err != nil {
			t.Fatalf("EvaluateValues failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// Evaluate
	{
		var result []string
		err := qb.Evaluate(&result, field)

		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}
}

func TestGlobalPermissions(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "alice")
	action := String("create_repository")

	qb := o.BuildQuery(NewQueryFact("has_permission", actor, action))

	// EvaluateExists
	{
		result, err := qb.EvaluateExists()
		if err != nil {
			t.Fatalf("EvaluateExists failed, %v", err)
		}
		if !result {
			t.Fatalf("result was false (expected true)")
		}
	}

	// Evaluate
	{
		var result bool
		err := qb.Evaluate(&result, nil)
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		if !result {
			t.Fatalf("result was false (expected true)")
		}
	}
}

func TestBulkActions(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	user := NewValue("User", "alice")
	action := TypedVar("String")
	repo := TypedVar("Repo")
	repos := []string{"anvil", "3", "5"}

	var result map[string][]string
	err := o.BuildQuery(NewQueryFact("allow", user, action, repo)).
		In(repo, repos).
		Evaluate(&result, map[Variable]Variable{repo: action})

	if err != nil {
		t.Fatalf("Evaluate failed, %v", err)
	}
	expected := map[string][]string{"anvil": {"read"}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("result did not match (got %v; expected %v)", result, expected)
	}
}

func TestAuthorizedResources(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	user := NewValue("User", "alice")
	action := String("read")
	repo := TypedVar("Repo")
	repos := []string{"anvil", "3", "5"}

	qb := o.BuildQuery(NewQueryFact("allow", user, action, repo)).
		In(repo, repos)
	expected := []string{"anvil"}

	// EvaluateValues
	{
		result, err := qb.EvaluateValues(repo)
		if err != nil {
			t.Fatalf("EvaluateValues failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// Evaluate
	{
		var result []string
		err := qb.Evaluate(&result, repo)
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}
}

func TestAllowWFilters(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "bob")
	action := TypedVar("String")
	repo := TypedVar("Repo")
	org := NewValue("Org", "acme")

	qb := o.BuildQuery(NewQueryFact("allow", actor, action, repo)).
		And(NewQueryFact("has_relation", repo, String("parent"), org))

	// Evaluate map
	{
		var result map[string][]string
		err := qb.Evaluate(&result, map[Variable]Variable{repo: action})
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		expected := map[string][]string{"anvil": {"read", "write"}}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// Evaluate flat
	expected := [][]string{{"read", "anvil"}, {"write", "anvil"}}
	{
		var result [][]string
		err := qb.Evaluate(&result, []Variable{action, repo})
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// EvaluateCombinations
	{
		result, err := qb.EvaluateCombinations([]Variable{action, repo})
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}
}

func TestAllowWFiltersAndContext(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "bob")
	action := TypedVar("String")
	repo := TypedVar("Repo")
	newRepo := NewValue("Repo", "newRepo")
	org := NewValue("Org", "acme")

	var result map[string][]string
	err := o.BuildQuery(NewQueryFact("allow", actor, action, repo)).
		And(NewQueryFact("has_relation", repo, String("parent"), org)).
		WithContextFacts([]Fact{{
			Predicate: "has_relation",
			Args: []Value{
				newRepo,
				String("parent"),
				org,
			},
		}}).Evaluate(&result, map[Variable]Variable{repo: action})
	if err != nil {
		t.Fatalf("Evaluate failed, %v", err)
	}
	expected := map[string][]string{"anvil": {"read", "write"}, "newRepo": {"read", "write"}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("result did not match (got %v; expected %v)", result, expected)
	}
}

func TestReusingBuilders(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "alice")
	action := TypedVar("String")
	repo := NewValue("Repo", "anvil")

	q := o.BuildQuery(NewQueryFact("allow", actor, action, repo))

	// Evaluate with context
	{
		var result []string
		err := q.WithContextFacts([]Fact{{
			Predicate: "has_role",
			Args: []Value{
				actor,
				String("admin"),
				repo,
			},
		}}).Evaluate(&result, action)
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		expected := []string{"read", "write"}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// Evaluate no context
	{
		var result []string
		err := q.Evaluate(&result, action)
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		expected := []string{"read"}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}
}

func TestMultipleWithContextFacts(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "charlie")
	org := NewValue("Org", "osohq")
	repo := NewValue("Repo", "gitcloud")

	qb := o.BuildQuery(NewQueryFact("allow", actor, String("read"), repo)).
		WithContextFacts([]Fact{{
			Predicate: "has_role",
			Args: []Value{
				actor,
				String("member"),
				org,
			},
		}}).WithContextFacts([]Fact{{
		Predicate: "has_relation",
		Args: []Value{
			repo,
			String("parent"),
			org,
		},
	}})

	var result bool
	err := qb.Evaluate(&result, nil)

	if err != nil {
		t.Fatalf("Evaluate failed, %v", err)
	}
	if !result {
		t.Fatalf("result was false (expected true)")
	}
}

func TestWildcardMapResults(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	actor := NewValue("User", "bob")
	actionVar := TypedVar("String")
	orgVar := TypedVar("Org")

	// Evaluate single var
	{
		var result map[string][]string
		err := o.BuildQuery(NewQueryFact("allow", actor, actionVar, orgVar)).
			Evaluate(&result, map[Variable]Variable{orgVar: actionVar})
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		expected := map[string][]string{"acme": {"delete"}, "*": {"read"}}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}

	// Evaluate multi var
	{
		actorVar := TypedVar("User")
		org := NewValue("Org", "acme")

		var result map[string][]string
		err := o.BuildQuery(NewQueryFact("allow", actorVar, actionVar, org)).
			Evaluate(&result, map[Variable]Variable{actorVar: actionVar})
		if err != nil {
			t.Fatalf("Evaluate failed, %v", err)
		}
		expected := map[string][]string{"*": {"read"}, "alice": {"read"}, "bob": {"delete"}}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("result did not match (got %v; expected %v)", result, expected)
		}
	}
}

func TestNestedMaps(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	user := TypedVar("User")
	action := TypedVar("String")
	repo := TypedVar("Repo")
	repos := []string{"anvil", "3", "5"}

	var result map[string]map[string][]string
	err := o.BuildQuery(NewQueryFact("allow", user, action, repo)).
		In(repo, repos).
		Evaluate(&result, map[Variable]map[Variable]Variable{user: {repo: action}})
	if err != nil {
		t.Fatalf("Evaluate failed, %v", err)
	}
	expected := map[string]map[string][]string{"alice": {"anvil": {"read"}}, "bob": {"anvil": {"read", "write"}}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("result did not match (got %v; expected %v)", result, expected)
	}
}

func TestMultiMap(t *testing.T) {
	o := setupClient()
	defer teardown(o)

	user := TypedVar("User")
	action := TypedVar("String")
	repo := TypedVar("Repo")
	repos := []string{"anvil", "3", "5"}

	qb := o.BuildQuery(NewQueryFact("allow", user, action, repo)).In(repo, repos)

	var result map[string][]string
	err := qb.Evaluate(&result, map[Variable]Variable{user: action, repo: action})
	if err == nil {
		t.Fatalf("expected multiple map values to result in error")
	}
}
