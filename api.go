package oso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
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
	Predicate string          `json:"predicate"`
	Args      []concreteValue `json:"args"`
}

type factPattern struct {
	Predicate string          `json:"predicate"`
	Args      []variableValue `json:"args"`
}

type concreteValue struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type variableValue struct {
	Type *string `json:"type"`
	Id   *string `json:"id"`
}

type factChangeset interface {
	isInsert() bool
}

type batchInserts struct {
	Inserts []fact `json:"inserts"`
}

func (b batchInserts) isInsert() bool {
	return true
}

type batchDeletes struct {
	Deletes []factPattern `json:"deletes"`
}

func (b batchDeletes) isInsert() bool {
	return false
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

type query struct {
	Predicate    apiQueryCall               `json:"predicate"`
	Calls        []apiQueryCall             `json:"calls"`
	Constraints  map[string]queryConstraint `json:"constraints"`
	ContextFacts []fact                     `json:"context_facts"`
}

type queryResult struct {
	Results []map[string]string `json:"results"`
}

type statsResult struct {
	NumRoles     int `json:"num_roles"`
	NumRelations int `json:"num_relations"`
	NumFacts     int `json:"num_facts"`
}

type getPolicyMetadataResult struct {
	Metadata PolicyMetadata `json:"metadata"`
}

// Maps each resource type declared in the policy to the permissions, roles, and relations
// that are valid for that resource.
type PolicyMetadata struct {
	Resources map[string]ResourceMetadata `json:"resources"`
}

// The permissions, roles, and relations that are valid for a particular resource type.
type ResourceMetadata struct {
	Permissions []string          `json:"permissions"`
	Roles       []string          `json:"roles"`
	Relations   map[string]string `json:"relations"`
}

type localAuthQuery struct {
	Query        authorizeQuery `json:"query"`
	DataBindings string         `json:"data_bindings"`
}

type localListQuery struct {
	Query        listQuery `json:"query"`
	Column       string    `json:"column"`
	DataBindings string    `json:"data_bindings"`
}

type localActionsQuery struct {
	Query        actionsQuery `json:"query"`
	DataBindings string       `json:"data_bindings"`
}

type localQueryResult struct {
	Sql string `json:"sql"`
}

const maxBodySize = 10 * 1024 * 1024

func (c *OsoClientImpl) apiCall(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.url + "/api" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-OsoApiVersion", "0")
	req.Header.Set("X-Request-ID", uuid.New().String())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Oso-Instance-Id", c.clientId)

	if c.lastOffset != "" {
		req.Header.Set("OsoOffset", c.lastOffset)
	}

	return req, nil
}

func (c *OsoClientImpl) doRequest(req *http.Request, output interface{}, isMutation bool) error {
	fallbackEligible := func(url *url.URL) bool {
		contains := func(haystack []string, needle string) bool {
			for _, v := range haystack {
				if v == needle {
					return true
				}
			}

			return false
		}

		eligiblePaths := []string{"/api/authorize", "/api/authorize_resources", "/api/list", "/api/actions", "/api/query"}
		return c.fallbackHttpClient != nil && contains(eligiblePaths, url.EscapedPath())
	}
	// make requests with retryclient
	res, e := c.httpClient.Do(req)
	if e != nil {
		// attempt to make a final request to fallbackURL if configured
		if fallbackEligible(req.URL) {
			// override the URL for the request to point to fallback
			fb := c.fallbackUrl + req.URL.Path
			fbUrl, err := url.Parse(fb)
			if err != nil {
				return err
			}
			req.URL = fbUrl
			res, e = c.fallbackHttpClient.Do(req)
			if e != nil {
				return e
			}
		} else {
			return e
		}
	}
	defer res.Body.Close()
	resBodyJSON, e := io.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		var apiErr apiError
		e = json.Unmarshal(resBodyJSON, &apiErr)
		if e != nil {
			return e
		}
		requestID := res.Header.Get("X-Request-ID")
		return errors.New("Oso Cloud error: " + apiErr.Message + " (Request ID: " + requestID + ")")
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

func (c *OsoClientImpl) get(path string, query map[string]string, output interface{}) error {
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

func (c *OsoClientImpl) post(path string, data interface{}, output interface{}, isMutation bool) error {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return e
	}
	if len(reqBodyJSON) > maxBodySize {
		return fmt.Errorf("Request payload too large (bodySizeBytes: %d, maxBodySize: %d)", len(reqBodyJSON), maxBodySize)
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

func (c *OsoClientImpl) delete(path string, data interface{}, output interface{}) error {
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(data)
	if e != nil {
		return e
	}
	if len(reqBodyJSON) > maxBodySize {
		return fmt.Errorf("Request payload too large (bodySizeBytes: %d, maxBodySize: %d)", len(reqBodyJSON), maxBodySize)
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

func (c *OsoClientImpl) getPolicy() (*getPolicyResult, error) {
	var result getPolicyResult
	if e := c.get("/policy", nil, &result); e != nil {
		return nil, e
	}
	return &result, nil
}

func (c *OsoClientImpl) getPolicyMetadataResult(version *string) (*getPolicyMetadataResult, error) {
	var result getPolicyMetadataResult
	params := make(map[string]string)
	if version != nil {
		params["version"] = *version
	}
	if e := c.get("/policy_metadata", params, &result); e != nil {
		return nil, e
	}
	return &result, nil
}

func (c *OsoClientImpl) postPolicy(data policy) (*apiResult, error) {
	var resBody apiResult
	if e := c.post("/policy", data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postFacts(data fact) (*apiResult, error) {
	url := "/batch"
	changesets := []factChangeset{batchInserts{Inserts: []fact{data}}}

	var resBody apiResult
	if e := c.post(url, changesets, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) deleteFacts(data factPattern) (*apiResult, error) {
	url := "/batch"

	changesets := []factChangeset{batchDeletes{Deletes: []factPattern{data}}}
	var resBody apiResult
	if e := c.post(url, changesets, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postBatch(data []factChangeset) (*apiResult, error) {
	url := "/batch"
	var resBody apiResult
	if e := c.post(url, data, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postAuthorize(data authorizeQuery) (*authorizeResult, error) {
	url := "/authorize"
	var resBody authorizeResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postList(data listQuery) (*listResult, error) {
	url := "/list"
	var resBody listResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postActions(data actionsQuery) (*actionsResult, error) {
	url := "/actions"
	var resBody actionsResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postQuery(data query) (*queryResult, error) {
	url := "/evaluate_query"
	var resBody queryResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) getStats() (*statsResult, error) {
	url := "/stats"
	var resBody statsResult
	if e := c.get(url, nil, &resBody); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) clearData() (*apiResult, error) {
	url := "/clear_data"
	var resBody apiResult
	if e := c.post(url, nil, &resBody, true); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) getFacts(data factPattern) ([]fact, error) {
	url := "/facts"
	params := make(map[string]string)
	// TODO document that we don't support nil predicates anymore (why did we ever??)
	for i, arg := range data.Args {
		if arg.Type != nil {
			params[fmt.Sprintf("args.%d.type", i)] = *arg.Type
		}
		if arg.Id != nil {
			params[fmt.Sprintf("args.%d.id", i)] = *arg.Id
		}
	}
	var resBody []fact
	if e := c.get(url, params, &resBody); e != nil {
		return nil, e
	}
	return resBody, nil
}

func (c *OsoClientImpl) postAuthorizeQuery(query authorizeQuery) (*localQueryResult, error) {
	url := "/authorize_query"
	data := localAuthQuery{
		Query:        query,
		DataBindings: c.dataBindings,
	}
	var resBody localQueryResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postListQuery(query listQuery, column string) (*localQueryResult, error) {
	url := "/list_query"
	data := localListQuery{
		Query:        query,
		Column:       column,
		DataBindings: c.dataBindings,
	}
	var resBody localQueryResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}

func (c *OsoClientImpl) postActionsQuery(query actionsQuery) (*localQueryResult, error) {
	url := "/actions_query"
	data := localActionsQuery{
		Query:        query,
		DataBindings: c.dataBindings,
	}
	var resBody localQueryResult
	if e := c.post(url, data, &resBody, false); e != nil {
		return nil, e
	}
	return &resBody, nil
}
