package oso

import (
	"fmt"
)

type Instance struct {
	Type string
	Id   string
}

type BulkFact struct {
	Predicate string
	Args      []Instance
}

func String(s string) Instance {
	return Instance{Type: "String", Id: s}
}

func fromValue(value Value) (*Instance, error) {
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
	return &Instance{Type: typ, Id: id}, nil
}

func toValue(instance Instance) (*Value, error) {
	var typ, id *string
	if instance.Type == "" {
		typ = nil
	} else {
		typ = &instance.Type
	}
	if instance.Id == "" {
		id = nil
	} else {
		id = &instance.Id
	}

	return &Value{Id: id, Type: typ}, nil
}

func bulkFactsToFacts(facts []BulkFact) []Fact {
	payload := []Fact{}
	for _, fact := range facts {
		jsonArgs := []Value{}
		for _, arg := range fact.Args {
			arg, _ := toValue(arg)
			jsonArgs = append(jsonArgs, *arg)
		}
		payload = append(payload, Fact{
			Predicate: fact.Predicate,
			Args:      jsonArgs,
		})
	}
	return payload
}

func (c Client) AuthorizeWithContext(actor Instance, action string, resource Instance, context_facts []BulkFact) (bool, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return false, err
	}
	resourceT, err := toValue(resource)
	if err != nil {
		return false, err
	}
	payload := AuthorizeQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		Action:       action,
		ResourceType: *resourceT.Type,
		ResourceId:   *resourceT.Id,
		ContextFacts: bulkFactsToFacts(context_facts),
	}

	resp, err := c.PostAuthorize(payload)
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

func (c Client) Authorize(actor Instance, action string, resource Instance) (bool, error) {
	return c.AuthorizeWithContext(actor, action, resource, nil)
}

func (c Client) AuthorizeResourcesWithContext(actor Instance, action string, resources []Instance, context_facts []BulkFact) ([]Instance, error) {
	key := func(e Value) string {
		return fmt.Sprintf("%s:%s", *e.Type, *e.Id)
	}

	if len(resources) == 0 {
		return []Instance{}, nil
	}

	resourcesExtracted := make([]Value, len(resources))
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
	payload := AuthorizeResourcesQuery{
		ActorType:    *actorI.Type,
		ActorId:      *actorI.Id,
		Action:       action,
		Resources:    resourcesExtracted,
		ContextFacts: bulkFactsToFacts(context_facts),
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

func (c Client) AuthorizeResources(actor Instance, action string, resources []Instance) ([]Instance, error) {
	return c.AuthorizeResourcesWithContext(actor, action, resources, nil)
}

func (c Client) ListWithContext(actor Instance, action string, resource string, context_facts []BulkFact) ([]string, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	payload := ListQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		Action:       action,
		ResourceType: resource,
		ContextFacts: bulkFactsToFacts(context_facts),
	}

	resp, err := c.PostList(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c Client) List(actor Instance, action string, resource string, context_facts []Fact) ([]string, error) {
	return c.ListWithContext(actor, action, resource, nil)
}

func (c Client) ActionsWithContext(actor Instance, resource Instance, context_facts []BulkFact) ([]string, error) {
	actorT, err := toValue(actor)
	if err != nil {
		return nil, err
	}
	resourceT, err := toValue(resource)
	if err != nil {
		return nil, err
	}
	payload := ActionsQuery{
		ActorType:    *actorT.Type,
		ActorId:      *actorT.Id,
		ResourceType: *resourceT.Type,
		ResourceId:   *resourceT.Id,
		ContextFacts: bulkFactsToFacts(context_facts),
	}

	resp, err := c.PostActions(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c Client) Actions(actor Instance, resource Instance) ([]string, error) {
	return c.ActionsWithContext(actor, resource, nil)
}

func (c Client) Tell(predicate string, args ...Instance) error {
	jsonArgs := []Value{}
	for _, arg := range args {
		argT, err := toValue(arg)
		if err != nil {
			return err
		}
		jsonArgs = append(jsonArgs, *argT)
	}
	payload := Fact{
		Predicate: predicate,
		Args:      jsonArgs,
	}
	_, err := c.PostFacts(payload)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) BulkTell(facts []BulkFact) error {
	_, e := c.PostBulkLoad(bulkFactsToFacts(facts))
	if e != nil {
		return e
	}
	return nil
}

func (c Client) Delete(predicate string, args ...Instance) error {
	jsonArgs := []Value{}
	for _, arg := range args {
		argT, err := toValue(arg)
		if err != nil {
			return err
		}
		jsonArgs = append(jsonArgs, *argT)
	}
	payload := Fact{
		Predicate: predicate,
		Args:      jsonArgs,
	}
	_, err := c.DeleteFacts(payload)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) BulkDelete(facts []BulkFact) error {
	_, e := c.PostBulkDelete(bulkFactsToFacts(facts))
	if e != nil {
		return e
	}
	return nil
}

func (c Client) Get(predicate string, args ...Instance) ([]Fact, error) {
	resp, e := c.GetFacts(predicate, args)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return make([]Fact, 0), nil

	}
	return *resp, nil
}

func (c Client) Policy(policy string) error {
	payload := Policy{
		Filename: nil,
		Src:      policy,
	}
	_, e := c.PostPolicy(payload)
	if e != nil {
		return e
	}
	return nil
}

func (c Client) Query(predicate string, args ...*Instance) ([]Fact, error) {
	vargs := []Value{}
	for _, arg := range args {
		var argV *Value
		if arg == nil {
			argV = &Value{Type: nil, Id: nil}
		} else {
			var err error
			argV, err = toValue(*arg)
			if err != nil {
				return nil, err
			}
		}
		vargs = append(vargs, *argV)
	}
	query := Query{
		Fact: Fact{
			Predicate: predicate,
			Args:      vargs,
		},
		ContextFacts: make([]Fact, 0),
	}
	resp, e := c.PostQuery(query)
	if e != nil {
		return nil, e
	}
	return resp.Results, nil
}
