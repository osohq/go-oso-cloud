package oso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

type TypedId struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type Type interface {
	Type() string
}

type AuthorizeReq struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
}

type AuthorizeRes struct {
	allowed bool
}

func (c Client) apiCall(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.url + "/api" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.apiKey)
	req.Header.Set("User-Agent", "Oso Cloud (golang)")

	return req, nil
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
	payload := AuthorizeReq{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
		ResourceId:   resource.Id(),
	}

	reqBodyJson, e := json.Marshal(payload)
	if e != nil {
		return false, e
	}

	reqBody := bytes.NewBuffer(reqBodyJson)
	res, e := c.post("/authorize", reqBody)
	if e != nil {
		return false, e
	}
	defer res.Body.Close()

	resBodyJson, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return false, e
	}
	if res.StatusCode != 200 {
		return false, errors.New(string(resBodyJson))
	}

	var resBody AuthorizeRes
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return false, e
	}
	return resBody.allowed, nil
}

type ListReq struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
}

type ListRes struct {
	results []int // TODO
}

func (c Client) List(actor Instance, action string, resource Type) ([]int, error) {
	payload := ListReq{
		ActorType:    actor.Type(),
		ActorId:      actor.Id(),
		Action:       action,
		ResourceType: resource.Type(),
	}

	reqBodyJson, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}

	reqBody := bytes.NewBuffer(reqBodyJson)
	res, e := c.post("/list", reqBody)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()

	resBodyJson, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return nil, e
	}
	if res.StatusCode != 200 {
		return nil, errors.New(string(resBodyJson))
	}

	var resBody ListRes
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return resBody.results, nil
}

type Fact struct {
	Predicate string    `json:"predicate"`
	Args      []TypedId `json:"args"`
}

func (c Client) Tell(predicate string, args ...Instance) error {
	jsonArgs := []TypedId{}
	for _, arg := range args {
		jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
	}
	reqBody := Fact{
		Predicate: predicate,
		Args:      jsonArgs,
	}
	reqBodyJson, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJson)
	res, e := c.post("/facts", reqBodyBytes)
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

func (c Client) Delete(predicate string, args ...Instance) error {
	jsonArgs := []TypedId{}
	for _, arg := range args {
		jsonArgs = append(jsonArgs, TypedId{Type: arg.Type(), Id: arg.Id()})
	}
	reqBody := Fact{
		Predicate: predicate,
		Args:      jsonArgs,
	}
	reqBodyJson, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJson)
	res, e := c.delete("/facts", reqBodyBytes)
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
func (c Client) Get(predicate string, args ...Instance) ([]Fact, error) {
	req, e := c.apiCall("GET", "/facts", nil)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	q.Set("predicate", predicate)
	for i, arg := range args {
		q.Set(fmt.Sprintf("args.%d.type", i), arg.Type())
		q.Set(fmt.Sprintf("args.%d.id", i), arg.Id())
	}
	req.URL.RawQuery = q.Encode()
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	resBodyJson, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return nil, e
	}
	if res.StatusCode != 200 {
		return nil, errors.New(string(resBodyJson))
	}
	var resBody []Fact
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return resBody, nil
}

type PolicyReq struct {
	Src string `json:"src"`
}

func (c Client) Policy(policy string) error {
	reqBody := PolicyReq{
		Src: policy,
	}
	reqBodyJson, e := json.Marshal(reqBody)
	if e != nil {
		return e
	}
	reqBodyBytes := bytes.NewBuffer(reqBodyJson)
	res, e := c.post("/policy", reqBodyBytes)
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
