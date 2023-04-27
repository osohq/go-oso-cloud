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

type apiResult struct {
	Message string `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
}

type policy struct {
	Filename *string `json:"filename"`
	Src      string  `json:"src"`
}

type getPolicyResult struct {
	Policy *policy `json:"policy"`
}

type fact struct {
	Predicate string  `json:"predicate"`
	Args      []value `json:"args"`
}

type value struct {
	Type *string `json:"type"`
	Id   *string `json:"id"`
}

type bulk struct {
	Delete []fact `json:"delete"`
	Tell   []fact `json:"tell"`
}

type authorizeResult struct {
	Allowed bool `json:"allowed"`
}

type authorizeQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
	ContextFacts []fact `json:"context_facts"`
}

type authorizeResourcesResult struct {
	Results []value `json:"results"`
}

type authorizeResourcesQuery struct {
	ActorType    string  `json:"actor_type"`
	ActorId      string  `json:"actor_id"`
	Action       string  `json:"action"`
	Resources    []value `json:"resources"`
	ContextFacts []fact  `json:"context_facts"`
}

type listResult struct {
	Results []string `json:"results"`
}

type listQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ContextFacts []fact `json:"context_facts"`
}

type actionsResult struct {
	Results []string `json:"results"`
}

type actionsQuery struct {
	ActorType    string `json:"actor_type"`
	ActorId      string `json:"actor_id"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
	ContextFacts []fact `json:"context_facts"`
}

type queryResult struct {
	Results []fact `json:"results"`
}

type query struct {
	Fact         fact   `json:"fact"`
	ContextFacts []fact `json:"context_facts"`
}

type statsResult struct {
	NumRoles     int `json:"num_roles"`
	NumRelations int `json:"num_relations"`
	NumFacts     int `json:"num_facts"`
}

func (c client) apiCall(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.url + "/api" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-OsoApiVersion", "0")

	return req, nil
}

func (c client) post(path string, body io.Reader) (*http.Response, error) {
	req, err := c.apiCall("POST", path, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func (c client) delete(path string, body io.Reader) (*http.Response, error) {
	req, err := c.apiCall("DELETE", path, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func (c client) GetPolicy() (*getPolicyResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/policy"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody getPolicyResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostPolicy(data policy) (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/policy"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostFacts(data fact) (*fact, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/facts"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody fact
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) DeleteFacts(data fact) (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/facts"
	req, e := c.apiCall("DELETE", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostBulkLoad(data []fact) (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/bulk_load"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostBulkDelete(data []fact) (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/bulk_delete"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostBulk(data bulk) (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/bulk"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostAuthorize(data authorizeQuery) (*authorizeResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/authorize"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody authorizeResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostAuthorizeResources(data authorizeResourcesQuery) (*authorizeResourcesResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/authorize_resources"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody authorizeResourcesResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostList(data listQuery) (*listResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/list"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody listResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostActions(data actionsQuery) (*actionsResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/actions"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody actionsResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) PostQuery(data query) (*queryResult, error) {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return nil, e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	url := "/query"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody queryResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) GetStats() (*statsResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/stats"
	req, e := c.apiCall("GET", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody statsResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c client) ClearData() (*apiResult, error) {
	var reqBodyBytes io.Reader
	reqBodyBytes = nil
	url := "/clear_data"
	req, e := c.apiCall("POST", url, reqBodyBytes)
	if e != nil {
		return nil, e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody apiResult
	e = json.Unmarshal(resBodyJSON, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}

// NOTE: This method does not codegen property
func (c client) GetFacts(predicate string, args []value) (*[]fact, error) {
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
		q.Set(fmt.Sprintf("args.%d.type", i), *arg.Type)
		q.Set(fmt.Sprintf("args.%d.id", i), *arg.Id)
	}
	req.URL.RawQuery = q.Encode()
	res, e := c.httpClient.Do(req)
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
	var resBody []fact
	e = json.Unmarshal(resBodyJson, &resBody)
	if e != nil {
		return nil, e
	}
	return &resBody, nil
}
