package oso

import (
	"fmt"
)

type Instance interface {
	Type() string
	Id() string
}

type StringInstance struct {
	Str string
}

func (s StringInstance) Type() string {
	return "String"
}

func (s StringInstance) Id() string {
	return s.Str
}

func String(s string) StringInstance {
	return StringInstance{Str: s}
}

func (c Client) String(s string) StringInstance {
	return String(s)
}

type Type interface {
	Type() string
}

func (c Client) Authorize(actor Instance, action string, resource Instance, context_facts []Fact) (bool, error) {
	if context_facts == nil {
		context_facts = make([]Fact, 0)
	}
	payload := AuthorizeQuery{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
		ResourceId:   resource.Id(),
		ContextFacts: context_facts,
	}

	resp, err := c.PostAuthorize(payload)
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

func (c Client) AuthorizeResources(actor Instance, action string, resources []Instance, context_facts []Fact) ([]Instance, error) {
	if context_facts == nil {
		context_facts = make([]Fact, 0)
	}
	key := func(e TypedId) string {
		return fmt.Sprintf("%s:%s", e.Type, e.Id)
	}

	if len(resources) == 0 {
		return []Instance{}, nil
	}

	resourcesExtracted := make([]TypedId, len(resources))
	for i := range resources {
		resourcesExtracted[i] = TypedId{Type: resources[i].Type(), Id: resources[i].Id()}
	}

	payload := AuthorizeResourcesQuery{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		Action:       action,
		Resources:    resourcesExtracted,
		ContextFacts: context_facts,
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
		k := key(TypedId{Type: resources[i].Type(), Id: resources[i].Id()})
		_, ok := resultsLookup[k]
		if ok {
			results[n_results] = resources[i]
			n_results++
		}
	}

	return results[0:n_results], nil
}

func (c Client) List(actor Instance, action string, resource Type, context_facts []Fact) ([]string, error) {
	if context_facts == nil {
		context_facts = make([]Fact, 0)
	}
	payload := ListQuery{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
		ContextFacts: context_facts,
	}

	resp, err := c.PostList(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c Client) Actions(actor Instance, resource Instance, context_facts []Fact) ([]string, error) {
	if context_facts == nil {
		context_facts = make([]Fact, 0)
	}
	payload := ActionsQuery{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		ResourceType: resource.Type(),
		ResourceId:   resource.Id(),
		ContextFacts: context_facts,
	}

	resp, err := c.PostActions(payload)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c Client) Tell(predicate string, args ...Instance) error {
	jsonArgs := []TypedId{}
	for _, arg := range args {
		jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
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

type BulkFact struct {
	Predicate string
	Args      []Instance
}

func (c Client) BulkTell(facts []BulkFact) error {
	payload := []Fact{}

	for _, fact := range facts {
		jsonArgs := []TypedId{}
		for _, arg := range fact.Args {
			jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
		}
		payload = append(payload, Fact{
			Predicate: fact.Predicate,
			Args:      jsonArgs,
		})
	}
	_, e := c.PostBulkLoad(payload)
	if e != nil {
		return e
	}
	return nil
}

func (c Client) Delete(predicate string, args ...Instance) error {
	jsonArgs := []TypedId{}
	for _, arg := range args {
		jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
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
	payload := []Fact{}

	for _, fact := range facts {
		jsonArgs := []TypedId{}
		for _, arg := range fact.Args {
			jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
		}
		payload = append(payload, Fact{
			Predicate: fact.Predicate,
			Args:      jsonArgs,
		})
	}
	_, e := c.PostBulkDelete(payload)
	if e != nil {
		return e
	}
	return nil
}

// TODO(gj): Do we need equivalent of Oso::Client::get_roles in Ruby client?
func (c Client) Get(predicate string, args ...Instance) ([]Fact, error) {
	resp, e := c.GetFacts(predicate, args)
	if e != nil {
		return nil, e
	}
	return resp, nil
}

func (c Client) Policy(policy string) error {
	payload := Policy{
		Filename: "",
		Src:      policy,
	}
	_, e := c.PostPolicy(payload)
	if e != nil {
		return e
	}
	return nil
}
