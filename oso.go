// Oso Client cloud for Golang.
// For more detailed documentation, see https://www.osohq.com/docs/app-integration/client-apis/go
package oso

import (
	"errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
)

// A Value is an argument to a [Fact]. Example:
//
//	Value{Type: "User", ID: "alice"}
//	NewValue("User", "alice")
//
// A Value must have a non-empty Type and ID. Empty strings in either field are invalid.
type Value struct {
	Type string
	ID   string
}

// NewValue is a convenience constructor for [Value].
func NewValue(typ string, id string) Value {
	return Value{Type: typ, ID: id}
}

// Marker interface representing the union of [Fact] and [FactPattern].
type IntoFactPattern interface {
	intoFactPattern() (*factPattern, error)
}

// A Fact is the fundamental data model of Oso Cloud.
// See also: https://www.osohq.com/docs/concepts/oso-cloud-data-model
// A Fact must have a non-empty Predicate and each member of Args must be a valid
// [Value] (eg. they must all have non-empty Types and IDs).
//
// Example:
//
//	NewFact("has_role", NewValue("User", "alice"), String("owner"), NewValue("Repo", "acme"))
type Fact struct {
	Predicate string
	Args      []Value
}

// NewFact is a convenience constructor for [Fact].
func NewFact(predicate string, args ...Value) Fact {
	return Fact{Predicate: predicate, Args: args}
}

// A FactPattern lets you match facts based on the given (non-empty) Predicate
// and Args. The members of Args can be a [Value] (matching arguments of that
// exact value at the given position), a [ValueOfType] (matching facts with an
// argument of the given type at the given position), or nil (matching facts
// with any value at the given position).
//
// Example:
//
//	NewFactPattern("has_role", NewValue("User", "alice"), nil, NewValueOfType("Repo"))
//
// will match all "has_role" facts where the first argument is the User
// "alice", the second argument is anything, and the third argument is a
// Repo.
type FactPattern struct {
	Predicate string
	Args      []ValuePattern
}

// NewFactPattern is a convenience constructor for [FactPattern].
func NewFactPattern(predicate string, args ...ValuePattern) FactPattern {
	return FactPattern{Predicate: predicate, Args: args}
}

// A ValueOfType is used in a [FactPattern] to match an argument having the given Type.
// Type must not be an empty string.
type ValueOfType struct {
	Type string
}

// NewValueOfType is a convenience constructor for [ValueOfType].
func NewValueOfType(t string) ValueOfType {
	return ValueOfType{Type: t}
}

// Marker interface for the union type of [Value] and [ValueOfType].
// Used as arguments to [FactPattern].
type ValuePattern interface {
	typ() string
	id() string
}

func (v Value) typ() string {
	return v.Type
}

func (v Value) id() string {
	return v.ID
}

func (v ValueOfType) typ() string {
	return v.Type
}

func (v ValueOfType) id() string {
	return ""
}

type AuthorizeResult authorizeResult

// TODO explain what this ^ is and why it exists- something about local authz?

type AuthorizeOptions struct {
	ContextFacts []Fact
	ParityHandle *ParityHandle
}

// Constructs a [Value] from the given string.
func String(s string) Value {
	return Value{Type: "String", ID: s}
}

// Constructs a [Value] from the given integer.
func Integer(i int64) Value {
	return Value{Type: "Integer", ID: strconv.FormatInt(i, 10)}
}

// Constructs a [Value] from the given boolean.
func Boolean(b bool) Value {
	ID := "false"
	if b {
		ID = "true"
	}
	return Value{Type: "Boolean", ID: ID}
}

func fromValue(value concreteValue) (*Value, error) {
	return &Value{Type: value.Type, ID: value.Id}, nil
}

func toConcreteValue(instance Value) (*concreteValue, error) {
	if instance.Type == "" {
		return nil, errors.New("Value must have a non-empty Type")
	}
	if instance.ID == "" {
		return nil, errors.New("Value must have a non-empty ID")
	}
	return &concreteValue{Id: instance.ID, Type: instance.Type}, nil
}

func toInternalFact(f Fact) (*fact, error) {
	valueArgs := []concreteValue{}
	for _, arg := range f.Args {
		arg, _ := toConcreteValue(arg)
		valueArgs = append(valueArgs, *arg)
	}

	return &fact{Predicate: f.Predicate, Args: valueArgs}, nil
}

func fromInternalFact(f fact) (*Fact, error) {
	instanceArgs := []Value{}
	for _, arg := range f.Args {
		arg, _ := fromValue(arg)
		instanceArgs = append(instanceArgs, *arg)
	}

	return &Fact{Predicate: f.Predicate, Args: instanceArgs}, nil
}

func mapToInternalFacts(facts []Fact) []fact {
	payload := []fact{}
	for _, f := range facts {
		internalFact, _ := toInternalFact(f)
		payload = append(payload, *internalFact)
	}
	return payload
}

func mapFromInternalFacts(facts []fact) []Fact {
	payload := []Fact{}
	for _, f := range facts {
		externalFact, _ := fromInternalFact(f)
		payload = append(payload, *externalFact)
	}
	return payload
}

func (fact Fact) intoFactPattern() (*factPattern, error) {
	args := []variableValue{}
	for _, arg := range fact.Args {
		// Load-bearing variable assignments
		argType := arg.Type
		argID := arg.ID

		patternArg := variableValue{Type: &argType, Id: &argID} // these shouldn't be nil / empty
		args = append(args, patternArg)
	}
	return &factPattern{Predicate: fact.Predicate, Args: args}, nil
}

func (_factPattern FactPattern) intoFactPattern() (*factPattern, error) {
	args := []variableValue{}
	for _, arg := range _factPattern.Args {
		arg, err := toVariableValue(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, *arg)
	}
	return &factPattern{Predicate: _factPattern.Predicate, Args: args}, nil
}

func toVariableValue(v ValuePattern) (*variableValue, error) {
	if v == nil {
		return &variableValue{}, nil
	}

	var typ string
	var id *string
	if value, isValue := v.(Value); isValue {
		if value.Type == "" || value.ID == "" {
			return nil, errors.New("Value must have non-empty Type and ID")
		}
		typ = value.Type
		id = &value.ID
	}

	if valueOfType, isValueOfType := v.(ValueOfType); isValueOfType {
		if valueOfType.Type == "" {
			return nil, errors.New("ValueOfType must have non-empty Type")
		}
		typ = valueOfType.Type
	}

	return &variableValue{Type: &typ, Id: id}, nil
}

// A BatchTransaction lets you group many Fact inserts and deletes into a single HTTP call.
// See [OsoClientImpl.Batch]
type BatchTransaction interface {
	// Insert the given [Fact] into Oso Cloud as part of the transaction.
	Insert(fact Fact) error
	// Delete the given [Fact] or all facts matching the given [FactPattern] from Oso Cloud as part of the transaction.
	Delete(factPattern IntoFactPattern) error
	privateMarker()
}

type batchTransaction struct {
	changesets []factChangeset
}

func (tx *batchTransaction) Insert(data Fact) error {
	f, err := toInternalFact(data)
	if err != nil {
		return err
	}
	var changeset batchInserts
	lastIndex := len(tx.changesets) - 1
	if lastIndex >= 0 && (tx.changesets)[lastIndex].isInsert() {
		changeset = (tx.changesets)[lastIndex].(batchInserts)
		changeset.Inserts = append(changeset.Inserts, *f)
		tx.changesets[lastIndex] = changeset
	} else {
		// either this is the first changeset, or the last one was a delete.
		changeset = batchInserts{Inserts: []fact{*f}}
		tx.changesets = append(tx.changesets, changeset)
	}

	return nil
}

func (tx *batchTransaction) Delete(data IntoFactPattern) error {
	f, err := data.intoFactPattern()
	if err != nil {
		return err
	}

	var changeset batchDeletes
	lastIndex := len(tx.changesets) - 1
	if lastIndex >= 0 && !(tx.changesets)[lastIndex].isInsert() {
		changeset = (tx.changesets)[lastIndex].(batchDeletes)
		changeset.Deletes = append(changeset.Deletes, *f)
		tx.changesets[lastIndex] = changeset
	} else {
		// either this is the first changeset, or the last one was an insert.
		changeset = batchDeletes{Deletes: []factPattern{*f}}
		tx.changesets = append(tx.changesets, changeset)
	}

	return nil
}

func (tx batchTransaction) privateMarker() {}

// An interface to make it possible to swap out Oso Cloud implementations (eg. for unit tests).
// For more information on these functions, see [OsoClientImpl].
type OsoClient interface {
	Insert(fact Fact) error
	Delete(factOrFactPattern IntoFactPattern) error
	Batch(func(tx BatchTransaction)) error
	Get(factOrFactPattern IntoFactPattern) ([]Fact, error)

	Policy(policy string) error
	GetPolicyMetadata() (*PolicyMetadata, error)

	Actions(actor Value, resource Value) ([]string, error)
	ActionsWithContext(actor Value, resource Value, contextFacts []Fact) ([]string, error)
	Authorize(actor Value, action string, resource Value) (bool, error)
	AuthorizeWithContext(actor Value, action string, resource Value, contextFacts []Fact) (bool, error)
	AuthorizeWithOptions(actor Value, action string, resource Value, options *AuthorizeOptions) (bool, error)
	List(actor Value, action string, resource string, contextFacts []Fact) ([]string, error)
	ListWithContext(actor Value, action string, resource string, contextFacts []Fact) ([]string, error)
	BuildQuery(query QueryFact) QueryBuilder
	AuthorizeLocal(actor Value, action string, resource Value) (string, error)
	AuthorizeLocalWithContext(actor Value, action string, resource Value, contextFacts []Fact) (string, error)
	AuthorizeLocalWithOptions(actor Value, action string, resource Value, options *AuthorizeOptions) (string, error)
	ListLocal(actor Value, action string, resource string, column string) (string, error)
	ListLocalWithContext(actor Value, action string, resource string, column string, contextFacts []Fact) (string, error)
	ActionsLocal(actor Value, resource Value) (string, error)
	ActionsLocalWithContext(actor Value, resource Value, contextFacts []Fact) (string, error)
}

// The default implementation of [OsoClient]. Create an instance using the constructor
// functions on [OsoClient].
type OsoClientImpl struct {
	url                string
	apiKey             string
	httpClient         *http.Client
	userAgent          string
	lastOffset         string
	fallbackUrl        string
	fallbackHttpClient *http.Client
	dataBindings       string
	clientId           string
}

// Create a new Oso client with a fallbackURL and custom logger
//
// See https://pkg.go.dev/github.com/hashicorp/go-retryablehttp@v0.7.1#LeveledLogger
// for documentation on the logger interfaces supported.
func NewClientWithFallbackUrlAndLoggerAndDataBindings(url string, apiKey string, fallbackUrl string, logger interface{}, dataBindings string) OsoClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 10 * time.Millisecond
	retryClient.RetryWaitMax = 1 * time.Second
	retryClient.Logger = logger

	var userAgent string
	rv, err := os.ReadFile("VERSION")
	if err != nil {
		userAgent = "Oso Cloud (golang " + runtime.Version() + ")"
	} else {
		userAgent = "Oso Cloud (golang " + runtime.Version() + "; rv:" + strings.TrimSuffix(string(rv), "\n") + ")"
	}

	lastOffset := ""

	var fallbackClient *http.Client
	if fallbackUrl != "" {
		fallbackClient = &http.Client{}
	} else {
		fallbackClient = nil
	}

	if dataBindings != "" {
		dataBindingsContents, err := os.ReadFile(dataBindings)
		if err != nil {
			panic(err)
		} else {
			dataBindings = string(dataBindingsContents)
		}
	}

	clientId := uuid.New().String()

	return OsoClientImpl{url, apiKey, retryClient.StandardClient(), userAgent, lastOffset, fallbackUrl, fallbackClient, dataBindings, clientId}
}

// Create a new default Oso client
func NewClient(url string, apiKey string) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, "", nil, "")
}

// Create a new Oso client with a fallback URL configured
func NewClientWithFallbackUrl(url string, apiKey string, fallbackUrl string) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, fallbackUrl, nil, "")
}

// Create a new Oso client with a custom logger
//
// See https://pkg.go.dev/github.com/hashicorp/go-retryablehttp@v0.7.1#LeveledLogger
// for documentation on the logger interfaces supported.
func NewClientWithLogger(url string, apiKey string, logger interface{}) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, "", logger, "")
}

func NewClientWithFallbackUrlAndLogger(url string, apiKey string, fallbackUrl string, logger interface{}) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, fallbackUrl, logger, "")
}

func NewClientWithDataBindings(url string, apiKey string, dataBindings string) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, "", nil, dataBindings)
}

func NewClientWithFallbackUrlAndDataBindings(url string, apiKey string, fallbackUrl string, dataBindings string) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, fallbackUrl, nil, dataBindings)
}

func NewClientWithLoggerAndDataBindings(url string, apiKey string, logger interface{}, dataBindings string) OsoClient {
	return NewClientWithFallbackUrlAndLoggerAndDataBindings(url, apiKey, "", logger, dataBindings)
}

// Check a permission depending on data both in Oso Cloud and stored in a local database:
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) AuthorizeLocal(actor Value, action string, resource Value) (string, error) {
	return c.AuthorizeLocalWithOptions(actor, action, resource, &AuthorizeOptions{
		ContextFacts: []Fact{},
		ParityHandle: nil,
	})
}

// Check a permission depending on data both in Oso Cloud and stored in a local database:
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) AuthorizeLocalWithContext(actor Value, action string, resource Value, contextFacts []Fact) (string, error) {
	return c.AuthorizeLocalWithOptions(actor, action, resource, &AuthorizeOptions{
		ContextFacts: contextFacts,
		ParityHandle: nil,
	})
}

func (c OsoClientImpl) AuthorizeLocalWithOptions(actor Value, action string, resource Value, options *AuthorizeOptions) (string, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return "", err
	}
	resourceT, err := toConcreteValue(resource)
	if err != nil {
		return "", err
	}
	payload := authorizeQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		Action:       action,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
		ContextFacts: mapToInternalFacts(options.ContextFacts),
	}

	var parityHandle *ParityHandle
	if options.ParityHandle != nil {
		parityHandle = options.ParityHandle
	}

	resp, err := c.postAuthorizeQuery(payload, parityHandle)
	if err != nil {
		return "", err
	}
	return resp.Sql, nil
}

// List authorized resources depending on data both in Oso Cloud and stored in a local database:
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) ListLocal(actor Value, action string, resourceType string, column string) (string, error) {
	return c.ListLocalWithContext(actor, action, resourceType, column, []Fact{})
}

// List authorized resources depending on data both in Oso Cloud and stored in a local database:
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) ListLocalWithContext(actor Value, action string, resourceType string, column string, contextFacts []Fact) (string, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return "", err
	}

	payload := listQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		Action:       action,
		ResourceType: resourceType,
		ContextFacts: mapToInternalFacts(contextFacts),
	}

	resp, err := c.postListQuery(payload, column)
	if err != nil {
		return "", err
	}
	return resp.Sql, nil
}

// Fetches a query that can be run against your database to determine the actions
// an actor can perform on a resource.
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) ActionsLocal(actor Value, resource Value) (string, error) {
	return c.ActionsLocalWithContext(actor, resource, []Fact{})
}

// Fetches a query that can be run against your database to determine the actions
// an actor can perform on a resource.
// Returns a SQL query to run against the local database.
func (c OsoClientImpl) ActionsLocalWithContext(actor Value, resource Value, contextFacts []Fact) (string, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return "", err
	}
	resourceT, err := toConcreteValue(resource)
	if err != nil {
		return "", err
	}
	payload := actionsQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
		ContextFacts: mapToInternalFacts(contextFacts),
	}

	resp, err := c.postActionsQuery(payload)
	if err != nil {
		return "", err
	}
	return resp.Sql, nil
}

// Determines whether or not an action is allowed, based on a combination of
// authorization data and policy logic.
func (c OsoClientImpl) Authorize(actor Value, action string, resource Value) (bool, error) {
	return c.AuthorizeWithOptions(actor, action, resource, &AuthorizeOptions{
		ContextFacts: []Fact{},
		ParityHandle: nil,
	})
}

// Determines whether or not an action is allowed, based on a combination of
// authorization data (including the given context facts) and policy logic.
func (c OsoClientImpl) AuthorizeWithContext(actor Value, action string, resource Value, contextFacts []Fact) (bool, error) {
	return c.AuthorizeWithOptions(actor, action, resource, &AuthorizeOptions{
		ContextFacts: contextFacts,
		ParityHandle: nil,
	})
}

func (c OsoClientImpl) AuthorizeWithOptions(actor Value, action string, resource Value, options *AuthorizeOptions) (bool, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return false, err
	}
	resourceT, err := toConcreteValue(resource)
	if err != nil {
		return false, err
	}
	payload := authorizeQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		Action:       action,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
		ContextFacts: mapToInternalFacts(options.ContextFacts),
	}
	var parityHandle *ParityHandle
	if options.ParityHandle != nil {
		parityHandle = options.ParityHandle
	}

	resp, err := c.postAuthorize(payload, parityHandle)
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

// Fetches a list of resource ids on which an actor can perform a particular action, considering the given context facts.
func (c OsoClientImpl) ListWithContext(actor Value, action string, resourceType string, contextFacts []Fact) ([]string, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return nil, err
	}
	payload := listQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		Action:       action,
		ResourceType: resourceType,
		ContextFacts: mapToInternalFacts(contextFacts),
	}

	resp, err := c.postList(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// Fetches a list of resource ids on which an actor can perform a particular action.
func (c OsoClientImpl) List(actor Value, action string, resourceType string, contextFacts []Fact) ([]string, error) {
	return c.ListWithContext(actor, action, resourceType, nil)
}

// Fetches a list of actions which an actor can perform on a particular
// resource, considering the given context facts.
func (c OsoClientImpl) ActionsWithContext(actor Value, resource Value, contextFacts []Fact) ([]string, error) {
	actorT, err := toConcreteValue(actor)
	if err != nil {
		return nil, err
	}
	resourceT, err := toConcreteValue(resource)
	if err != nil {
		return nil, err
	}
	payload := actionsQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
		ContextFacts: mapToInternalFacts(contextFacts),
	}

	resp, err := c.postActions(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// Fetches a list of actions which an actor can perform on a particular resource.
func (c OsoClientImpl) Actions(actor Value, resource Value) ([]string, error) {
	return c.ActionsWithContext(actor, resource, nil)
}

// Adds the given fact to Oso Cloud.
func (c OsoClientImpl) Insert(fact Fact) error {
	internalFact, err := toInternalFact(fact)
	if err != nil {
		return err
	}
	_, err = c.postFacts(*internalFact)
	if err != nil {
		return err
	}
	return nil
}

// Delete the [Fact] or all facts matching the given [FactPattern].
//
// Does not throw an error if no facts match the given pattern.
//
// The arguments to pattern can be a [Value] (matching arguments of that
// exact value at the given position), a [ValueOfType] (matching facts with an
// argument of the given type at the given position), or nil (matching facts
// with any value at the given position).
//
// Example:
//
//	oso.Delete(NewFactPattern("has_role", NewValue("User", "alice"), nil, NewValueOfType("Repo")))
//
// will delete all "has_role" facts where the first argument is the User
// "alice", the second argument is anything, and the third argument is a
// Repo.
func (c OsoClientImpl) Delete(pattern IntoFactPattern) error {
	payload, err := pattern.intoFactPattern()
	if err != nil {
		return err
	}
	_, err = c.deleteFacts(*payload)
	if err != nil {
		return err
	}
	return nil
}

// Batch together many inserts and deletes into a single HTTP call.
// Example:
//
//	oso.Batch(func(tx BatchTransaction) {
//	  tx.Insert(NewFact("has_role", NewValue("User", "alice"), String("owner"), NewValue("Repo", "acme"))
//	  tx.Insert(NewFact("has_role", NewValue("User", "alice"), String("member"), NewValue("Repo", "anvil"))
//	  tx.Delete(NewFactPattern("has_role", NewValue("User", "bob"), nil, nil))
//	})
func (c OsoClientImpl) Batch(fn func(BatchTransaction)) error {
	tx := batchTransaction{changesets: []factChangeset{}}
	fn(&tx)
	_, err := c.postBatch(tx.changesets)
	if err != nil {
		return err
	}
	return nil
}

// Lists facts that are stored in Oso Cloud that match the given [FactPattern].
func (c OsoClientImpl) Get(pattern IntoFactPattern) ([]Fact, error) {
	payload, err := pattern.intoFactPattern()
	if err != nil {
		return nil, err
	}

	resp, e := c.getFacts(*payload)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return make([]Fact, 0), nil
	}
	return mapFromInternalFacts(resp), nil
}

// Returns metadata about the currently active policy.
func (c OsoClientImpl) GetPolicyMetadata() (*PolicyMetadata, error) {
	metadata, err := c.getPolicyMetadataResult(nil)
	if err != nil {
		return nil, err
	}
	return &metadata.Metadata, nil
}

// Updates the active policy in Oso Cloud.
// The string passed into this function should be written in Polar.
func (c OsoClientImpl) Policy(p string) error {
	payload := policy{
		Filename: nil,
		Src:      p,
	}
	_, e := c.postPolicy(payload)
	if e != nil {
		return e
	}
	return nil
}

// Query for an arbitrary expression:
// Use [TypedVar] to create variables to use in the query,
// and refer to them in the final [QueryBuilder.Evaluate] call to get their values.
func (c OsoClientImpl) BuildQuery(query QueryFact) QueryBuilder {
	return newBuilder(c, query)
}
