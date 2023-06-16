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

func (c *client) apiCall(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.url + "/api" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-OsoApiVersion", "0")

	if c.lastOffset != "" {
		req.Header.Set("OsoOffset", c.lastOffset)
	}

	return req, nil
}

func (c *client) doRequest(req *http.Request, output interface{}, isMutation bool) error {
	res, e := c.httpClient.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	resBodyJSON, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode >= 400 {
		var apiErr apiError
		e = json.Unmarshal(resBodyJSON, &apiErr)
		if e != nil {
			return e
		}
		return errors.New(apiErr.Message)
	}
	if isMutation {
		c.lastOffset = res.Header.Get("OsoOffset")
	}
	e = json.Unmarshal(resBodyJSON, output)
	if e != nil {
		return e
	}
	return nil
}

func (c *client) get(path string, query map[string]string, output interface{}) error {
	req, e := c.apiCall("GET", path, nil)
	if e != nil {
		return e
	}
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	return c.doRequest(req, output, false)
}

func (c *client) post(path string, data interface{}, output interface{}, isMutation bool) error {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	req, e := c.apiCall("POST", path, reqBodyBytes)
	if e != nil {
		return e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	return c.doRequest(req, output, isMutation)
}

func (c *client) delete(path string, data interface{}, output interface{}) error {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return e
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)
	req, e := c.apiCall("DELETE", path, reqBodyBytes)
	if e != nil {
		return e
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	return c.doRequest(req, output, true)
}

func (c *client) GetPolicy() (*getPolicyResult, error) {
	var result getPolicyResult
	if e := c.get("/policy", nil, &result); e != nil {
		return nil, e
	}
	return &result, nil
}

func (c *client) PostPolicy(data policy) (*apiResult, error) {
	var resBody apiResult
	if e := c.post("/policy", data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostFacts(data fact) (*fact, error) {
	url := "/facts"
	var resBody fact
	if e := c.post(url, data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) DeleteFacts(data fact) (*apiResult, error) {
	url := "/facts"
	var resBody apiResult
	if e := c.delete(url, data, &resBody); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostBulkLoad(data []fact) (*apiResult, error) {
	url := "/bulk_load"
	var resBody apiResult
	if e := c.post(url, data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostBulkDelete(data []fact) (*apiResult, error) {
	url := "/bulk_delete"
	var resBody apiResult
	if e := c.post(url, data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostBulk(data bulk) (*apiResult, error) {
	url := "/bulk"
	var resBody apiResult
	if e := c.post(url, data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostAuthorize(data authorizeQuery) (*authorizeResult, error) {
	url := "/authorize"
	var resBody authorizeResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostAuthorizeResources(data authorizeResourcesQuery) (*authorizeResourcesResult, error) {
	url := "/authorize_resources"
	var resBody authorizeResourcesResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostList(data listQuery) (*listResult, error) {
	url := "/list"
	var resBody listResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostActions(data actionsQuery) (*actionsResult, error) {
	url := "/actions"
	var resBody actionsResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) PostQuery(data query) (*queryResult, error) {
	url := "/query"
	var resBody queryResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) GetStats() (*statsResult, error) {
	url := "/stats"
	var resBody statsResult
	if e := c.get(url, nil, &resBody); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) ClearData() (*apiResult, error) {
	url := "/clear_data"
	var resBody apiResult
	if e := c.post(url, nil, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *client) GetFacts(predicate string, args []value) ([]fact, error) {
	url := "/facts"
	params := make(map[string]string)
	params["predicate"] = predicate
	for i, arg := range args {
		params[fmt.Sprintf("args.%d.type", i)] = *arg.Type
		params[fmt.Sprintf("args.%d.id", i)] = *arg.Id
	}
	var resBody []fact
	if e := c.get(url, params, &resBody); e != nil {
		return nil, e
	}
	return resBody, nil
}
