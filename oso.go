// Oso Client cloud for Golang.
// For more detailed documentation, see https://www.osohq.com/docs/reference/client-apis/go
package oso

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type Instance struct {
	Type string
	ID   string
}

type Fact struct {
	Name string
	Args []Instance
}

func String(s string) Instance {
	return Instance{Type: "String", ID: s}
}

func fromValue(value value) (*Instance, error) {
	var typ, id string
	if value.Type == nil {
		typ = ""
	} else {
		typ = *value.Type
	}
	if value.Id == nil {
		id = ""
	} else {
		id = *value.Id
	}
	return &Instance{Type: typ, ID: id}, nil
}

func toValue(instance Instance) (*value, error) {
	var typ, id *string
	if instance.Type == "" {
		typ = nil
	} else {
		typ = &instance.Type
	}
	if instance.ID == "" {
		id = nil
	} else {
		id = &instance.ID
	}

	return &value{Id: id, Type: typ}, nil
}

func toInternalFact(f Fact) (*fact, error) {
	valueArgs := []value{}
	for _, arg := range f.Args {
		arg, _ := toValue(arg)
		valueArgs = append(valueArgs, *arg)
	}

	return &fact{Predicate: f.Name, Args: valueArgs}, nil
}

func fromInternalFact(f fact) (*Fact, error) {
	instanceArgs := []Instance{}
	for _, arg := range f.Args {
		arg, _ := fromValue(arg)
		instanceArgs = append(instanceArgs, *arg)
	}

	return &Fact{Name: f.Predicate, Args: instanceArgs}, nil
}

func mapToInternalFacts(facts []Fact) []fact {
	payload := []fact{}
	for _, f := range facts {
		internal_fact, _ := toInternalFact(f)
		payload = append(payload, *internal_fact)
	}
	return payload
}

func mapFromInternalFacts(facts []fact) []Fact {
	payload := []Fact{}
	for _, f := range facts {
		external_fact, _ := fromInternalFact(f)
		payload = append(payload, *external_fact)
	}
	return payload
}

type OsoClient interface {
	// List authorized actions:
	// Fetches a list of actions which an actor can perform on a particular resource.
	Actions(actor Instance, resource Instance) ([]string, error)

	// List authorized actions for a list of resources
	// Fetches a list of actions which an actor can perform on a list of resources.
	//
	// Note: this only works for resources of the same type.
	BulkActions(actor Instance, resources []Instance, context_facts []Fact) ([][]string, error)

	// List authorized actions:
	// Fetches a list of actions which an actor can perform on a particular resource, considering the given context facts.
	ActionsWithContext(actor Instance, resource Instance, context_facts []Fact) ([]string, error)

	// Check a permission:
	// Determines whether or not an action is allowed, based on a combination of authorization data and policy logic.
	Authorize(actor Instance, action string, resource Instance) (bool, error)

	// Check authorized resources:
	// Returns a subset of resources on which an actor can perform a particular action.
	// Ordering and duplicates, if any exist, are preserved.
	AuthorizeResources(actor Instance, action string, resources []Instance) ([]Instance, error)

	// Check authorized resources:
	// Returns a subset of resources on which an actor can perform a particular action, considering the given context facts.
	// Ordering and duplicates, if any exist, are preserved.
	AuthorizeResourcesWithContext(actor Instance, action string, resources []Instance, context_facts []Fact) ([]Instance, error)
	// Check a permission:
	// Determines whether or not an action is allowed, based on a combination of authorization data (including the given context facts) and policy logic.
	AuthorizeWithContext(actor Instance, action string, resource Instance, context_facts []Fact) (bool, error)

	// Transactionally delete and add facts:
	// Deletes and adds many facts in one atomic transaction. The deletions are performed before the adds.
	// Does not throw an error when the facts to delete are not found.
	Bulk(delete []Fact, tell []Fact) error

	// Delete many facts:
	// Deletes many facts at once. Does not throw an error when some of the facts are not found.
	BulkDelete(facts []Fact) error

	// Add many facts:
	// Adds many facts at once.
	BulkTell(facts []Fact) error

	// Delete fact:
	// Deletes a fact. Does not throw an error if the fact is not found.
	Delete(predicate string, args ...Instance) error

	// List facts:
	// Lists facts that are stored in Oso Cloud. Can be used to check the existence of a particular fact, or used to fetch all facts that have a particular argument.
	Get(predicate string, args ...Instance) ([]Fact, error)

	// List authorized resources:
	// Fetches a list of resource ids on which an actor can perform a particular action.
	List(actor Instance, action string, resource string, context_facts []Fact) ([]string, error)

	// List authorized resources:
	// Fetches a list of resource ids on which an actor can perform a particular action, considering the given context facts.
	ListWithContext(actor Instance, action string, resource string, context_facts []Fact) ([]string, error)

	// Update the active policy:
	// Updates the policy in Oso Cloud. The string passed into this method should be written in Polar.
	Policy(policy string) error

	// Query Oso Cloud:
	// Query Oso Cloud for any predicate, and any combination of concrete and
	// wildcard arguments.
	Query(predicate string, args ...*Instance) ([]Fact, error)

	// Add fact:
	// Adds a fact named predicate with the provided arguments.
	Tell(predicate string, args ...Instance) error
}

type client struct {
	url                string
	apiKey             string
	httpClient         *http.Client
	userAgent          string
	lastOffset         string
	fallbackUrl        string
	fallbackHttpClient *http.Client
}

// Create a new Oso client with a custom logger
//
// See https://pkg.go.dev/github.com/hashicorp/go-retryablehttp@v0.7.1#LeveledLogger
// for documentation on the logger interfaces supported.
func NewClientWithLogger(url string, apiKey string, logger interface{}) OsoClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
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
	return client{url, apiKey, retryClient.StandardClient(), userAgent, lastOffset, "", nil}

}

// Create a new Oso client with a fallbackURL and custom logger
//
// See https://pkg.go.dev/github.com/hashicorp/go-retryablehttp@v0.7.1#LeveledLogger
// for documentation on the logger interfaces supported.
func NewClientWithFallbackUrlAndLogger(url string, apiKey string, fallbackUrl string, logger interface{}) OsoClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
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

	if fallbackUrl != "" {
		fallbackClient := &http.Client{}
		return client{url, apiKey, retryClient.StandardClient(), userAgent, lastOffset, fallbackUrl, fallbackClient}
	} else {
		return client{url, apiKey, retryClient.StandardClient(), userAgent, lastOffset, fallbackUrl, nil}
	}

}

// Create a new default Oso client
func NewClient(url string, apiKey string) OsoClient {
	return NewClientWithFallbackUrlAndLogger(url, apiKey, "", nil)
}

// Create a new Oso client with a fallback URL configured
func NewClientWithFallbackUrl(url string, apiKey string, fallbackUrl string) OsoClient {
	return NewClientWithFallbackUrlAndLogger(url, apiKey, fallbackUrl, nil)
}

func (c client) AuthorizeWithContext(actor Instance, action string, resource Instance, context_facts []Fact) (bool, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return false, err
	}
	resourceT, err := toValue(resource)
	if err != nil {
		return false, err
	}
	payload := authorizeQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		Action:       action,
		ResourceType: *resourceT.Type,
		ResourceId:   *resourceT.Id,
		ContextFacts: mapToInternalFacts(context_facts),
	}

	resp, err := c.PostAuthorize(payload)
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

func (c client) Authorize(actor Instance, action string, resource Instance) (bool, error) {
	return c.AuthorizeWithContext(actor, action, resource, nil)
}

func (c client) AuthorizeResourcesWithContext(actor Instance, action string, resources []Instance, context_facts []Fact) ([]Instance, error) {
	key := func(e value) string {
		return fmt.Sprintf("%s:%s", *e.Type, *e.Id)
	}

	if len(resources) == 0 {
		return []Instance{}, nil
	}

	resourcesExtracted := make([]value, len(resources))
	for i := range resources {
		extracted, err := toValue(resources[i])
		if err != nil {
			return nil, err
		}
		resourcesExtracted[i] = *extracted
	}

	actorI, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	payload := authorizeResourcesQuery{
		ActorType:    *actorI.Type,
		ActorId:      *actorI.Id,
		Action:       action,
		Resources:    resourcesExtracted,
		ContextFacts: mapToInternalFacts(context_facts),
	}

	resp, err := c.PostAuthorizeResources(payload)
	if err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return []Instance{}, nil
	}

	resultsLookup := make(map[string]bool, len(resp.Results))
	for i := range resp.Results {
		k := key(resp.Results[i])
		_, ok := resultsLookup[k]
		if !ok {
			resultsLookup[k] = true
		}
	}

	results := make([]Instance, len(resources))
	var n_results = 0
	for i := range resources {
		extracted, err := toValue(resources[i])
		if err != nil {
			return nil, err
		}
		k := key(*extracted)
		_, ok := resultsLookup[k]
		if ok {
			results[n_results] = resources[i]
			n_results++
		}
	}

	return results[0:n_results], nil
}

func (c client) AuthorizeResources(actor Instance, action string, resources []Instance) ([]Instance, error) {
	return c.AuthorizeResourcesWithContext(actor, action, resources, nil)
}

func (c client) ListWithContext(actor Instance, action string, resource string, context_facts []Fact) ([]string, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	payload := listQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		Action:       action,
		ResourceType: resource,
		ContextFacts: mapToInternalFacts(context_facts),
	}

	resp, err := c.PostList(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c client) List(actor Instance, action string, resource string, context_facts []Fact) ([]string, error) {
	return c.ListWithContext(actor, action, resource, nil)
}

func (c client) ActionsWithContext(actor Instance, resource Instance, context_facts []Fact) ([]string, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	resourceT, err := toValue(resource)
	if err != nil {
		return nil, err
	}
	payload := actionsQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		ResourceType: *resourceT.Type,
		ResourceId:   *resourceT.Id,
		ContextFacts: mapToInternalFacts(context_facts),
	}

	resp, err := c.PostActions(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c client) BulkActions(actor Instance, resources []Instance, context_facts []Fact) ([][]string, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	resourcesT := []value{}
	var resourceType *string
	for _, resource := range resources {
		resourceT, err := toValue(resource)
		if err != nil {
			return nil, err
		}
		if resourceType == nil {
			resourceType = resourceT.Type
		} else if *resourceType != *resourceT.Type {
			return nil, fmt.Errorf("BulkActions: resources must be of the same type")
		}
		resourcesT = append(resourcesT, *resourceT)
	}

	queries := []actionsQuery{}
	for i, resource := range resourcesT {
		ContextFacts := []fact{}
		// Only map context facts once
		// since we reuse them across the
		// whole query
		if i == 0 {
			ContextFacts = mapToInternalFacts(context_facts)
		}
		queries = append(queries, actionsQuery{
			ActorType:    *actorT.Type,
			ActorId:      *actorT.Id,
			ResourceType: *resourceType,
			ResourceId:   *resource.Id,
			ContextFacts: ContextFacts,
		})
	}

	resp, err := c.PostBulkActions(queries)
	if err != nil {
		return nil, err
	}
	results := [][]string{}
	for _, r := range resp {
		results = append(results, r.Results)
	}
	return results, nil
}

func (c client) Actions(actor Instance, resource Instance) ([]string, error) {
	return c.ActionsWithContext(actor, resource, nil)
}

func (c client) Tell(name string, args ...Instance) error {
	jsonArgs := []value{}
	for _, arg := range args {
		argT, err := toValue(arg)
		if err != nil {
			return err
		}
		jsonArgs = append(jsonArgs, *argT)
	}
	payload := fact{
		Predicate: name,
		Args:      jsonArgs,
	}
	_, err := c.PostFacts(payload)
	if err != nil {
		return err
	}
	return nil
}

func (c client) BulkTell(facts []Fact) error {
	_, e := c.PostBulkLoad(mapToInternalFacts(facts))
	if e != nil {
		return e
	}
	return nil
}

func (c client) Delete(name string, args ...Instance) error {
	jsonArgs := []value{}
	for _, arg := range args {
		argT, err := toValue(arg)
		if err != nil {
			return err
		}
		jsonArgs = append(jsonArgs, *argT)
	}
	payload := fact{
		Predicate: name,
		Args:      jsonArgs,
	}
	_, err := c.DeleteFacts(payload)
	if err != nil {
		return err
	}
	return nil
}

func (c client) BulkDelete(facts []Fact) error {
	_, e := c.PostBulkDelete(mapToInternalFacts(facts))
	if e != nil {
		return e
	}
	return nil
}

func (c client) Bulk(delete []Fact, tell []Fact) error {
	_, e := c.PostBulk(bulk{Delete: mapToInternalFacts(delete), Tell: mapToInternalFacts(tell)})
	return e
}

func (c client) Get(predicate string, args ...Instance) ([]Fact, error) {
	var jsonPredicate *string
	if predicate == "" {
		jsonPredicate = nil
	} else {
		jsonPredicate = &predicate
	}

	jsonArgs := []value{}
	for _, arg := range args {
		argT, err := toValue(arg)
		if err != nil {
			return nil, err
		}
		jsonArgs = append(jsonArgs, *argT)
	}

	resp, e := c.GetFacts(jsonPredicate, jsonArgs)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return make([]Fact, 0), nil
	}
	return mapFromInternalFacts(resp), nil
}

func (c client) Policy(p string) error {
	payload := policy{
		Filename: nil,
		Src:      p,
	}
	_, e := c.PostPolicy(payload)
	if e != nil {
		return e
	}
	return nil
}

func (c client) Query(predicate string, args ...*Instance) ([]Fact, error) {
	vargs := []value{}
	for _, arg := range args {
		var argV *value
		if arg == nil {
			argV = &value{Type: nil, Id: nil}
		} else {
			var err error
			argV, err = toValue(*arg)
			if err != nil {
				return nil, err
			}
		}
		vargs = append(vargs, *argV)
	}
	query := query{
		Fact: fact{
			Predicate: predicate,
			Args:      vargs,
		},
		ContextFacts: make([]fact, 0),
	}
	resp, e := c.PostQuery(query)
	if e != nil {
		return nil, e
	}
	return mapFromInternalFacts(resp.Results), nil
}
