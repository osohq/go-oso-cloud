package oso

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
)

type queryId struct{ id string }

func (q queryId) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.id)
}

type queryConstraint struct {
	Type string   `json:"type"`
	IDs  []string `json:"ids"`
}

func (this queryConstraint) clone() queryConstraint {
	if this.IDs == nil {
		return queryConstraint{
			Type: this.Type,
			IDs:  nil,
		}
	}
	return queryConstraint{
		Type: this.Type,
		IDs:  append([]string{}, this.IDs...),
	}
}

type QueryFact struct {
	Predicate string
	Args      []queryArg
}

type IntoQueryArg interface {
	intoQueryArg() queryArg
}

func NewQueryFact(predicate string, args ...IntoQueryArg) QueryFact {
	queryArgs := make([]queryArg, 0, len(args))
	for _, arg := range args {
		queryArgs = append(queryArgs, arg.intoQueryArg())
	}

	return QueryFact{
		Predicate: predicate,
		Args:      queryArgs,
	}
}

type queryArg struct {
	typ   string
	varId queryId
	value string
}

type apiQueryCall struct {
	predicate string
	args      []queryId
}

func (call apiQueryCall) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{call.predicate, call.args})
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyz"

func randId(n int) queryId {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return queryId{id: fmt.Sprintf("var_%s", b)}
}

// A Variable which can be referred to in a query.
// Should be constructed with the [TypedVar] helper.
//
// A Variable must have a non-empty typ and id. Empty strings in either field are invalid.
type Variable struct {
	typ string
	id  queryId
}

func (vari Variable) intoQueryArg() queryArg {
	return queryArg{
		typ:   vari.typ,
		varId: vari.id,
		value: "", // nil
	}
}

func (val Value) intoQueryArg() queryArg {
	return queryArg{
		typ:   val.Type,
		varId: queryId{}, // nil
		value: val.ID,
	}
}

// Construct a new query variable of a specific type.
//
// Note that you must use a concrete type here: the abstract types "Actor" and "Resource"
// are not allowed.
func TypedVar(Type string) Variable {
	return Variable{
		typ: Type,
		id:  randId(7),
	}
}

// Helper class to support building a custom Oso query.
//
// Initialize this with [OsoClientImpl.BuildQuery] and chain calls to [QueryBuilder.And] and [QueryBuilder.In] to add additional constraints.
//
// After building your query, run it and get the results by calling one of the Evaluate* methods.
type QueryBuilder struct {
	oso          OsoClientImpl
	predicate    apiQueryCall
	calls        []apiQueryCall
	constraints  map[queryId]queryConstraint
	contextFacts []Fact
	Error        error
}

func newBuilder(oso OsoClientImpl, fact QueryFact) QueryBuilder {
	this := QueryBuilder{oso: oso, calls: []apiQueryCall{}, constraints: map[queryId]queryConstraint{}}
	args := make([]queryId, 0, len(fact.Args))
	for _, arg := range fact.Args {
		id := this.pushArg(arg)
		args = append(args, id)
	}
	this.predicate = apiQueryCall{predicate: fact.Predicate, args: args}
	return this
}

func (this QueryBuilder) clone() QueryBuilder {
	constraints := map[queryId]queryConstraint{}
	for k, v := range this.constraints {
		constraints[k] = v.clone()
	}

	return QueryBuilder{
		oso:          this.oso,
		predicate:    this.predicate,
		calls:        append([]apiQueryCall{}, this.calls...),
		constraints:  constraints,
		contextFacts: append([]Fact{}, this.contextFacts...),
		Error:        this.Error,
	}
}

func (this *QueryBuilder) pushArg(arg queryArg) queryId {
	if arg.value == "" { // variable
		argId := arg.varId
		if _, exists := this.constraints[argId]; !exists {
			this.constraints[argId] = queryConstraint{Type: arg.typ, IDs: nil}
		}
		return argId
	} else { // value
		value := arg.value
		type_ := arg.typ
		newVar := TypedVar(type_)
		newId := newVar.id
		this.constraints[newId] = queryConstraint{Type: type_, IDs: []string{value}}
		return newId
	}
}

// Constrain a query variable to be one of a set of values.
// For example:
//
//	repos := []string {"acme", "anvil"}
//	repo := TypedVar("Repo")
//	action := TypedVar("String")
//	// Get all the actions the actor can perform on the repos that are in the given set
//	authorizedActions, err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, repo)).
//		In(repo, repos).
//		EvaluateValues(action)
func (this QueryBuilder) In(v Variable, values []string) QueryBuilder {
	if this.Error != nil {
		return this
	}
	clone := this.clone()
	bind, exists := clone.constraints[v.id]
	if !exists {
		clone.Error = errors.New("can only constrain variables that are used in the query")
		return clone
	}
	if bind.IDs != nil {
		clone.Error = errors.New("can only set values on each variable once")
		return clone
	}
	bind.IDs = values
	return clone
}

// Add another condition that must be true of the query results.
// For example:
//
//	// Query for all the repos on which the given actor can perform the given action,
//	// and require the repos to belong to the given folder
//	repo := TypedVar("Repo")
//	authorizedReposInFolder, err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, repo)).
//		And(NewQueryFact("has_relation", repo, String("folder"), folder)).
//		EvaluateValues(repo)
func (this QueryBuilder) And(fact QueryFact) QueryBuilder {
	if this.Error != nil {
		return this
	}
	clone := this.clone()
	args := make([]queryId, 0, len(fact.Args))
	for _, arg := range fact.Args {
		id := clone.pushArg(arg)
		args = append(args, id)
	}
	clone.calls = append(clone.calls, apiQueryCall{predicate: fact.Predicate, args: args})
	return clone
}

// Add context facts to the query.
func (this QueryBuilder) WithContextFacts(facts []Fact) QueryBuilder {
	if this.Error != nil {
		return this
	}
	out := this.clone()
	out.contextFacts = append(out.contextFacts, facts...)
	return out
}

func (this QueryBuilder) asQuery() (query, error) {
	constraints := make(map[string]queryConstraint)
	for k, v := range this.constraints {
		constraints[k.id] = v
	}

	contextFacts := make([]fact, 0, len(this.contextFacts))
	for _, fact := range this.contextFacts {
		ifact, err := toInternalFact(fact)
		if err != nil {
			return query{}, err
		}
		contextFacts = append(contextFacts, *ifact)
	}

	return query{
		Predicate:    this.predicate,
		Calls:        this.calls,
		Constraints:  constraints,
		ContextFacts: contextFacts,
	}, nil
}

// Evaluate the query and return a boolean representing if the action is authorized or not.
//
//	// true if the given actor can perform the given action on the given resource
//	allowed, err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, resource)).
//		EvaluateExists()
func (this QueryBuilder) EvaluateExists() (bool, error) {
	if this.Error != nil {
		return false, this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return false, err
	}
	results, err := this.oso.postQuery(query)
	if err != nil {
		return false, err
	}
	can := len(results.Results) != 0
	return can, nil
}

// Evaluate the query and return a slice of values for the given variable. For example:
//
//	action := TypedVar("String")
//	// all the actions the actor can perform on the given resource- eg. ["read", "write"]
//	actions, err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, resource)).
//		EvaluateValues(action)
func (this QueryBuilder) EvaluateValues(t Variable) ([]string, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return nil, err
	}
	results, err := this.oso.postQuery(query)
	if err != nil {
		return nil, err
	}
	// Use a map to track unique values
	seen := make(map[string]struct{})
	out := make([]string, 0) // Can't predict capacity
	for _, row := range results.Results {
		val := handleWildcard(row[t.id.id])
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out, nil
}

// Evaluate the query and return a slice of tuples of values for the given variables. For example:
//
//	action := TypedVar("String")
//	repo := TypedVar("Repo")
//	// a slice of pairs of allowed actions and repo IDs- eg. [["read", "acme"], ["read", "anvil"], ["write", "anvil"]]
//	pairs, err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, repo)).
//		EvaluateCombinations([]Variable {action, repo})
func (this QueryBuilder) EvaluateCombinations(ts []Variable) ([][]string, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return nil, err
	}
	results, err := this.oso.postQuery(query)
	if err != nil {
		return nil, err
	}
	out := make([][]string, 0, len(results.Results))
	for _, row := range results.Results {
		outRow := make([]string, 0, len(ts))
		for _, t := range ts {
			outRow = append(outRow, handleWildcard(row[t.id.id]))
		}
		out = append(out, outRow)
	}
	return out, nil
}

// Evaluate the query, and write the result into the `out` parameter.
// The shape of the return value is determined by what you pass in:
//
// - If you pass no arguments, returns a boolean. For example:
//
//	// true if the given actor can perform the given action on the given resource
//	var allowed bool
//	err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, resource)).
//		Evaluate(&allowed, nil)
//
// - If you pass a variable, returns a slice of values for that variable. For example:
//
//	action := TypedVar("String")
//	// all the actions the actor can perform on the given resource- eg. ["read", "write"]
//	var actions []string
//	err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, resource)).
//		Evaluate(&actions, action)
//
// - If you pass a slice of variables, returns a slice of tuples of values for those variables.
// For example:
//
//	action := TypedVar("String")
//	repo := TypedVar("Repo")
//	// a slice of pairs of allowed actions and repo IDs- eg. [["read", "acme"], ["read", "anvil"], ["write", "anvil"]]
//	var pairs [][]string
//	err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, repo)).
//		Evaluate(&pairs, []Variable {action, repo})
//
// - If you pass a map mapping one input variable (call it K) to another
// (call it V), returns a map of unique values of K to the unique values of
// V for each value of K. For example:
//
//	action := TypedVar("String")
//	repo := TypedVar("Repo")
//	// a map of repo IDs to allowed actions-  eg. { "acme": ["read"], "anvil": ["read", "write"]}
//	var mapping map[string][]string
//	err := oso.
//		BuildQuery(NewQueryFact("allow", actor, action, repo)).
//		Evaluate(&mapping, map[Variable]Variable{repo: action})
func (this QueryBuilder) Evaluate(out interface{}, arg interface{}) error {
	if this.Error != nil {
		return this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return err
	}
	results, err := this.oso.postQuery(query)
	if err != nil {
		return err
	}
	return evaluateResults(reflect.ValueOf(out), arg, results.Results)
}

func evaluateResults(out reflect.Value, arg interface{}, results []map[string]string) error {
	if out.Kind() != reflect.Pointer {
		return errors.New("`out` must be pointer")
	}
	if arg == nil {
		out.Elem().SetBool(len(results) != 0)
		return nil
	}
	ref := reflect.ValueOf(arg)
	switch ref.Kind() {
	case reflect.Struct:
		vari, ok := arg.(Variable)
		if !ok {
			return fmt.Errorf("non-Variable struct `%s`", ref.Type().String())
		}
		outElem := out.Elem().Type()

		// Use a map to track unique values
		seen := make(map[string]struct{})
		list := reflect.MakeSlice(outElem, 0, 0) // Can't predict capacity
		for _, r := range results {
			val := handleWildcard(r[vari.id.id])
			if _, exists := seen[val]; !exists {
				seen[val] = struct{}{}
				list = reflect.Append(list, reflect.ValueOf(val))
			}
		}
		out.Elem().Set(list)
		return nil
	case reflect.Slice:
		outElem := out.Elem().Type()
		list := reflect.MakeSlice(outElem, 0, len(results))
		for _, r := range results {
			elem := reflect.New(outElem.Elem())
			err := evaluateResultItem(elem, arg, r)
			if err != nil {
				return err
			}
			list = reflect.Append(list, elem.Elem())
		}
		out.Elem().Set(list)
		return nil
	case reflect.Map:
		outElem := out.Elem().Type()
		structuredGrouping := reflect.MakeMap(outElem)
		if ref.Len() > 1 {
			return errors.New("`Evaluate` cannot accept maps with >1 elements")
		}
		for _, v := range ref.MapKeys() {
			subarg := ref.MapIndex(v)
			vari, ok := v.Interface().(Variable)
			if !ok {
				return fmt.Errorf("non-Variable key `%s`", v.Type().String())
			}
			grouping := map[string][]map[string]string{}
			for _, result := range results {
				key, exists := result[vari.id.id]
				if !exists {
					return errors.New("API result missing variable. This shouldn't happen--please reach out to Oso.")
				}
				key = handleWildcard(key)
				if list, exists := grouping[key]; exists {
					grouping[key] = append(list, result)
				} else {
					grouping[key] = append([]map[string]string{}, result)
				}
			}
			for key, value := range grouping {
				subout := reflect.New(outElem.Elem())
				err := evaluateResults(subout, subarg.Interface(), value)
				if err != nil {
					return err
				}
				structuredGrouping.SetMapIndex(reflect.ValueOf(key), subout.Elem())
			}
		}
		out.Elem().Set(structuredGrouping)
		return nil
	}
	return errors.New("bad type match in evaluateResults")
}

func evaluateResultItem(out reflect.Value, arg interface{}, result map[string]string) error {
	if out.Kind() != reflect.Pointer {
		return errors.New("`out` must be pointer")
	}
	ref := reflect.ValueOf(arg)
	switch ref.Kind() {
	case reflect.Struct:
		t, ok := arg.(Variable)
		if !ok {
			return fmt.Errorf("non-Variable struct `%s`", ref.Type().String())
		}
		out.Elem().Set(reflect.ValueOf(handleWildcard(result[t.id.id])))
		return nil
	case reflect.Slice:
		outElem := out.Elem().Type()
		list := reflect.MakeSlice(outElem, 0, ref.Len())
		for i := 0; i < ref.Len(); i++ {
			subarg := ref.Index(i)
			elem := reflect.New(outElem.Elem())
			err := evaluateResultItem(elem, subarg.Interface(), result)
			if err != nil {
				return err
			}
			list = reflect.Append(list, elem.Elem())
		}
		out.Elem().Set(list)
		return nil
	}
	return errors.New("bad type match in evaluateResultItem")
}

func handleWildcard(v string) string {
	if v == "" {
		return "*"
	}
	return v
}

// Fetches a complete SQL query that can be run against your database,
// selecting a row for each authorized combination of the query variables in
// `columnNamesToQueryVars` (ie. combinations of variables that satisfy the
// Oso query).
// If you pass an empty map, the returned SQL query will select a single row
// with a boolean column called `result`.
func (this QueryBuilder) EvaluateLocalSelect(columnNamesToQueryVars map[string]Variable) (string, error) {
	if this.Error != nil {
		return "", this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return "", err
	}
	queryVarsToColumnNames := make(map[string]string)
	for columnName, queryVar := range columnNamesToQueryVars {
		id := queryVar.id.id
		if _, containsKey := queryVarsToColumnNames[id]; containsKey {
			return "", fmt.Errorf("Found a duplicated %s variable- you may not select a query variable more than once.", queryVar.typ)
		}
		queryVarsToColumnNames[id] = columnName
	}

	result, err := this.oso.postQueryLocal(query, localQuerySelect(queryVarsToColumnNames))
	if err != nil {
		return "", err
	}

	return result.Sql, nil
}

// Fetches a SQL fragment, which you can embed into the `WHERE` clause of a SQL
// query against your database to filter out unauthorized rows (ie. rows that
// don't satisfy the Oso query).
func (this QueryBuilder) EvaluateLocalFilter(columnName string, queryVar Variable) (string, error) {
	if this.Error != nil {
		return "", this.Error
	}
	query, err := this.asQuery()
	if err != nil {
		return "", err
	}

	result, err := this.oso.postQueryLocal(query, localQueryFilter(columnName, queryVar.id.id))
	if err != nil {
		return "", err
	}

	return result.Sql, nil
}
