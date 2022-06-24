// This file is generated from the openapi spec
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

type Foo struct {
	Hello string `json:"hello"`
}

type ApiResult struct {
	Message string `json:"message"`
}

type ApiError struct {
	Message string `json:"message"`
}

type Policy struct {
	Filename string `json:"filename"`
	Src      string `json:"src"`
}

type GetPolicyResult struct {
	Policy Policy `json:"policy"`
}

type Fact struct {
	Predicate string    `json:"predicate"`
	Args      []TypedId `json:"args"`
}

type TypedId struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type ArgQuery struct {
	Tag string `json:"tag"`
	Id  string `json:"id"`
}

type Role struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Role         string `json:"role"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
}

type Relation struct {
	FromType string `json:"from_type"`
	FromId   string `json:"from_id"`
	Relation string `json:"relation"`
	ToType   string `json:"to_type"`
	ToId     string `json:"to_id"`
}

type AuthorizeResult struct {
	Allowed bool `json:"allowed"`
}

type AuthorizeQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
	ContextFacts []Fact `json:"context_facts"`
}

type AuthorizeResourcesResult struct {
	Results []TypedId `json:"results"`
}

type AuthorizeResourcesQuery struct {
	ActorType    string    `json:"actor_type"`
	ActorId      string    `json:"actor_id"`
	Action       string    `json:"action"`
	Resources    []TypedId `json:"resources"`
	ContextFacts []Fact    `json:"context_facts"`
}

type ListResult struct {
	Results []string `json:"results"`
}

type ListQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ContextFacts []Fact `json:"context_facts"`
}

type ActionsResult struct {
	Results []string `json:"results"`
}

type ActionsQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
	ContextFacts []Fact `json:"context_facts"`
}

type StatsResult struct {
	NumRoles             int `json:"num_roles"`
	NumRelations         int `json:"num_relations"`
	NumFacts             int `json:"num_facts"`
	RecentAuthorizations int `json:"recent_authorizations"`
}

type Backup struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Filepath string `json:"filepath"`
}

type Client struct {
	url    string
	apiKey string
}

func NewClient(url string, apiKey string) Client {
	return Client{url, apiKey}
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

func (c Client) Hello() (*Foo, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody Foo
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) GetPolicy() (*GetPolicyResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/policy"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody GetPolicyResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostPolicy(data Policy) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/policy"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

// NOTE: This method does not codegen property
func (c Client) GetFacts(predicate string, args []Instance) ([]Fact, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/facts"
	req, e := c.apiCall("GET", url, reqBodyBytes)
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

func (c Client) PostFacts(data Fact) (*Fact, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/facts"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody Fact
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) DeleteFacts(data Fact) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/facts"
	req, e := c.apiCall("DELETE", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) GetInspect(tag string, id string) (*[]Fact, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/inspect"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	q.Set("tag", tag)
	q.Set("id", id)
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
	return &resBody, nil
}

func (c Client) GetRoles(actor_type string, actor_id string, role string, resource_type string, resource_id string) (*[]Role, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/roles"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	q.Set("actor_type", actor_type)
	q.Set("actor_id", actor_id)
	q.Set("role", role)
	q.Set("resource_type", resource_type)
	q.Set("resource_id", resource_id)
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
	var resBody []Role
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostRoles(data Role) (*Role, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/roles"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody Role
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) DeleteRoles(data Role) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/roles"
	req, e := c.apiCall("DELETE", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) GetRelations(object_type string, object_id string, relation string) (*[]Relation, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/relations"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	q.Set("object_type", object_type)
	q.Set("object_id", object_id)
	q.Set("relation", relation)
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
	var resBody []Relation
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostRelations(data Relation) (*Relation, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/relations"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody Relation
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) DeleteRelations(data Relation) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/relations"
	req, e := c.apiCall("DELETE", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostBulkLoad(data []Fact) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/bulk_load"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostBulkDelete(data []Fact) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/bulk_delete"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostAuthorize(data AuthorizeQuery) (*AuthorizeResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/authorize"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody AuthorizeResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostAuthorizeResources(data AuthorizeResourcesQuery) (*AuthorizeResourcesResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/authorize_resources"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody AuthorizeResourcesResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostList(data ListQuery) (*ListResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/list"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ListResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) PostActions(data ActionsQuery) (*ActionsResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJson, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJson)
	url := "/actions"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ActionsResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) GetStats() (*StatsResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/stats"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody StatsResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) ClearData() (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/clear_data"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) ListBackups() (*[]Backup, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/backups"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody []Backup
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) CreateBackup() (*Backup, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/backups"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody Backup
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) DeleteBackup(backup_key string) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := fmt.Sprintf("/backups/%v", backup_key)
	req, e := c.apiCall("DELETE", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c Client) RestoreFromBackup(backup_key string) (*ApiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := fmt.Sprintf("/backups/%v/restore", backup_key)
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
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
	var resBody ApiResult
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}
