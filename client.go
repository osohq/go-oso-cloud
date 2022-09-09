package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	url    string
	apiKey string
	// TODO(gj): configurable logging?
}

func NewClient(url string, apiKey string) Client {
	return Client{url, apiKey}
}

type Instance interface {
	Id() string
	Type() string
}

type Type interface {
	Type() string
}

type Role struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Role         string `json:"role"`
	ActorID      string `json:"actor_id"`
	ActorType    string `json:"actor_type"`
}

func (c Client) apiCall(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.url + "/api" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.apiKey)

	return req, nil
}

func (c Client) get(path string) (*http.Response, error) {
	req, err := c.apiCall("GET", path, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func (c Client) post(path string, body io.Reader) (*http.Response, error) {
	req, err := c.apiCall("POST", path, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func (c Client) delete(path string, body io.Reader) (*http.Response, error) {
	req, err := c.apiCall("DELETE", path, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func (c Client) Authorize(actor Instance, action string, resource Instance) (bool, error) {
	payload := struct {
		ActorType    string `json:"actor_type"`
		ActorID      string `json:"actor_id"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}{
		ActorType:    actor.Type(),
		ActorID:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
		ResourceID:   resource.Id(),
	}

	reqBodyJSON, e := json.Marshal(payload)
	if e != nil {
		return false, e
	}

	reqBody := bytes.NewBuffer(reqBodyJSON)
	res, e := c.post("/authorize", reqBody)
	if e != nil {
		return false, e
	}
	defer res.Body.Close()

	resBodyJSON, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return false, e
	}
	if res.StatusCode != 200 {
		return false, errors.New(string(resBodyJSON))
	}

	var resBody struct {
		Allowed bool `json:"allowed"`
	}
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return false, e
	}
	return resBody.Allowed, nil
}

func (c Client) List(actor Instance, action string, resource Type) ([]int, error) {
	payload := struct {
		ActorType    string `json:"actor_type"`
		ActorID      string `json:"actor_id"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
	}{
		ActorType:    actor.Type(),
		ActorID:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
	}

	reqBodyJSON, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}

	reqBody := bytes.NewBuffer(reqBodyJSON)
	res, e := c.post("/list", reqBody)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()

	resBodyJSON, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return nil, e
	}
	if res.StatusCode != 200 {
		return nil, errors.New(string(resBodyJSON))
	}

	type ListRes struct {
		Results []int `json:"results"`
	}
	var resBody ListRes
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return resBody.Results, nil
}

type relationReq struct {
	FromID   string `json:"from_id"`
	FromType string `json:"from_type"`
	Relation string `json:"relation"`
	ToID     string `json:"to_id"`
	ToType   string `json:"to_type"`
}

func (c Client) AddRelation(from Instance, name string, to Instance) error {
	reqBody := relationReq{
		FromID:   from.Id(),
		FromType: from.Type(),
		Relation: name,
		ToID:     to.Id(),
		ToType:   to.Type(),
	}
	reqBodyJSON, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJSON)
	res, e := c.post("/relations", reqBodyBytes)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	resBody, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode != 200 {
		return errors.New(string(resBody))
	}
	return nil
}

func (c Client) DeleteRelation(from Instance, name string, to Instance) error {
	reqBody := relationReq{
		FromID:   from.Id(),
		FromType: from.Type(),
		Relation: name,
		ToID:     to.Id(),
		ToType:   to.Type(),
	}
	reqBodyJSON, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJSON)
	res, e := c.delete("/relations", reqBodyBytes)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	resBody, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode != 200 {
		return errors.New(string(resBody))
	}
	return nil
}

func (c Client) AddRole(actor Instance, name string, resource Instance) error {
	reqBody := Role{
		ActorID:      actor.Id(),
		ActorType:    actor.Type(),
		Role:         name,
		ResourceID:   resource.Id(),
		ResourceType: resource.Type(),
	}
	reqBodyJSON, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJSON)
	res, e := c.post("/roles", reqBodyBytes)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	resBody, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode != 200 {
		return errors.New(string(resBody))
	}
	return nil
}

func (c Client) DeleteRole(actor Instance, name string, resource Instance) error {
	reqBody := Role{
		ActorID:      actor.Id(),
		ActorType:    actor.Type(),
		Role:         name,
		ResourceID:   resource.Id(),
		ResourceType: resource.Type(),
	}
	reqBodyJSON, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJSON)
	res, e := c.post("/roles", reqBodyBytes)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	resBody, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode != 200 {
		return errors.New(string(resBody))
	}
	return nil
}

// TODO(gj): Do we need equivalent of Oso::Client::get_roles in Ruby client?
func (c Client) GetResourceRoleForActor(resource Instance, role string, actor Instance) ([]Role, error) {
	req, e := c.apiCall("GET", "/roles", nil)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	q.Set("actor_type", actor.Type())
	q.Set("actor_id", actor.Id())
	q.Set("role", role)
	q.Set("resource_type", resource.Type())
	q.Set("resource_id", resource.Id())
	req.URL.RawQuery = q.Encode()
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	resBodyJSON, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return nil, e
	}
	if res.StatusCode != 200 {
		return nil, errors.New(string(resBodyJSON))
	}
	var resBody []Role
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return resBody, nil
}
