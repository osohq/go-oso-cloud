package oso

import (
	"errors"
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

func fromTypedId(typedId TypedId) (*Instance, error) {
	return &Instance{Type: typedId.Type, Id: typedId.Id}, nil
}

func toTypedId(instance Instance) (*TypedId, error) {
	if instance.Id == "" || instance.Type == "" {
		return nil, errors.New(`Oso: No type and id present on instance passed to Oso`)
	}
	return &TypedId{Id: instance.Id, Type: instance.Type}, nil
}

func fromVariable(variable Variable) (*Instance, error) {
	if variable.Kind == "FreeVariable" {
		return &Instance{Type: "", Id: ""}, nil
	} else if variable.Kind == "TypedVariable" {
		typ := variable.Value.(*TypedVar).Type
		return &Instance{Type: typ, Id: ""}, nil
	} else if variable.Kind == "TypedId" {
		typ := variable.Value.(*TypedId).Type
		id := variable.Value.(*TypedId).Id
		return &Instance{Type: typ, Id: id}, nil
	}
	return nil, errors.New(`Oso: Invalid Variable Kind`)
}

func toVariable(instance *Instance) (*Variable, error) {
	if instance == nil {
		return &Variable{Kind: "FreeVariable", Value: nil}, nil
	}
	if instance.Id == "" {
		if instance.Type == "" {
			return &Variable{Kind: "FreeVariable", Value: nil}, nil
		} else {
			return &Variable{Kind: "TypedVariable", Value: &TypedVar{Type: instance.Type}}, nil
		}
	} else {
		if instance.Type == "" {
			return nil, errors.New(`Oso: Invalid instance can't have type without id.`)
		}
		return &Variable{Kind: "TypedId", Value: &TypedId{Type: instance.Type, Id: instance.Id}}, nil
	}
}

func bulkFactsToFacts(facts []BulkFact) []Fact {
	payload := []Fact{}
	for _, fact := range facts {
		jsonArgs := []TypedId{}
		for _, arg := range fact.Args {
			arg, _ := toTypedId(arg)
			jsonArgs = append(jsonArgs, TypedId{Type: arg.Type, Id: arg.Id})
		}
		payload = append(payload, Fact{
			Predicate: fact.Predicate,
			Args:      jsonArgs,
		})
	}
	return payload
}

func (c Client) AuthorizeWithContext(actor Instance, action string, resource Instance, context_facts []BulkFact) (bool, error) {
	actorT, err := toTypedId(actor)
	if err != nil {
		return false, err
	}
	resourceT, err := toTypedId(resource)
	if err != nil {
		return false, err
	}
	payload := AuthorizeQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		Action:       action,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
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
	key := func(e TypedId) string {
		return fmt.Sprintf("%s:%s", e.Type, e.Id)
	}

	if len(resources) == 0 {
		return []Instance{}, nil
	}

	resourcesExtracted := make([]TypedId, len(resources))
	for i := range resources {
		extracted, err := toTypedId(resources[i])
		if err != nil {
			return nil, err
		}
		resourcesExtracted[i] = *extracted
	}

	actorI, err := toTypedId(actor)
	if err != nil {
		return nil, err
	}
	payload := AuthorizeResourcesQuery{
		ActorType:    actorI.Type,
		ActorId:      actorI.Id,
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
		extracted, err := toTypedId(resources[i])
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
	actorT, err := toTypedId(actor)
	if err != nil {
		return nil, err
	}
	payload := ListQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
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
	actorT, err := toTypedId(actor)
	if err != nil {
		return nil, err
	}
	resourceT, err := toTypedId(resource)
	if err != nil {
		return nil, err
	}
	payload := ActionsQuery{
		ActorType:    actorT.Type,
		ActorId:      actorT.Id,
		ResourceType: resourceT.Type,
		ResourceId:   resourceT.Id,
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
	jsonArgs := []TypedId{}
	for _, arg := range args {
		argT, err := toTypedId(arg)
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
	jsonArgs := []TypedId{}
	for _, arg := range args {
		argT, err := toTypedId(arg)
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

func (c Client) Query(predicate string, args ...*Instance) ([]VariableFact, error) {
	vargs := []Variable{}
	for _, arg := range args {
		argV, err := toVariable(arg)
		if err != nil {
			return nil, err
		}
		vargs = append(vargs, *argV)
	}
	query := Query{
		Fact: VariableFact{
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
