package oso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// A struct to store information about a request
//
// We need this because we can't just build a request object and reuse it to
// call the fallback service for two reasons:
//
//  1. After the request has been built, we can't distinguish between the part
//     of the path that is from the base URL, and the part of the path that is
//     from the specific API endpoint. For example, if we have
//     `{host}/base/api/authorize`, we only want to append the `/api/authorize`
//     part onto the fallback URL.
//  2. After the request has been made, the body has been consumed, so we need
//     to build a new io.Reader after we know we need to call fallback.
//
// Given that, we pass this object around and build the request as needed.
type RequestData struct {
	method string
	path   string
	data   interface{}
	query  map[string]string
}

func (r RequestData) Body() (io.Reader, error) {
	if r.data == nil {
		return nil, nil
	}
	var reqBodyBytes io.Reader
	reqBodyJSON, e := json.Marshal(r.data)
	if e != nil {
		return nil, e
	}
	if len(reqBodyJSON) > maxBodySize {
		return nil, fmt.Errorf("request payload too large (body size bytes: %d, max body size: %d)", len(reqBodyJSON), maxBodySize)
	}
	reqBodyBytes = bytes.NewBuffer(reqBodyJSON)

	return reqBodyBytes, nil
}

func (r RequestData) Method() string {
	return r.method
}

func (r RequestData) Query() map[string]string {
	return r.query
}

func (r RequestData) Path() string {
	return r.path
}

const maxBodySize = 10 * 1024 * 1024

func (c *OsoClientImpl) buildRequest(baseUrl string, requestData RequestData) (*http.Request, error) {
	url := baseUrl + "/api" + requestData.path
	body, err := requestData.Body()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(requestData.Method(), url, body)
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

	q := req.URL.Query()
	queryArgs := requestData.Query()
	for k, v := range queryArgs {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func (c *OsoClientImpl) apiCall(requestData RequestData) (*http.Request, error) {
	return c.buildRequest(c.url, requestData)
}

func (c *OsoClientImpl) fallbackApiCall(requestData RequestData) (*http.Request, error) {
	return c.buildRequest(c.fallbackUrl, requestData)
}

func (c *OsoClientImpl) fallbackEligible(path string, method string) bool {
	type endpoint struct {
		path   string
		method string
	}

	contains := func(haystack []endpoint, needle endpoint) bool {
		for _, v := range haystack {
			if strings.HasSuffix(needle.path, v.path) && strings.EqualFold(needle.method, v.method) {
				return true
			}
		}

		return false
	}

	eligiblePaths := []endpoint{
		{"/api/authorize", "post"},
		{"/api/authorize_resources", "post"},
		{"/api/list", "post"},
		{"/api/actions", "post"},
		{"/api/query", "post"},
		{"/api/evaluate_query", "post"},
		{"/api/authorize_query", "post"},
		{"/api/list_query", "post"},
		{"/api/actions_query", "post"},
		{"/api/facts", "get"},
		{"/api/policy_metadata", "get"},
	}
	return c.fallbackHttpClient != nil && contains(eligiblePaths,
		endpoint{path, method},
	)
}

// Actually send the request. Takes request data that can be used to build the
// relevant request with different base urls. This is needed to handle falling
// back to a host at a different base url.
func (c *OsoClientImpl) doRequest(requestData RequestData, output interface{}, isMutation bool) error {
	req, err := c.apiCall(requestData)
	if err != nil {
		return err
	}

	// make requests with retryclient
	res, e := c.httpClient.Do(req)
	// NOTE: We have had cases where internal policy evaluation errors have
	// resulted in 400s, so defensively allow 400s to retry on the fallback
	// service.
	if e != nil || res.StatusCode == 400 || res.StatusCode >= 500 {
		// attempt to make a final request to fallbackURL if configured
		if c.fallbackEligible(req.URL.EscapedPath(), req.Method) {
			// Build a new request object for the fallback request
			// NOTE: We can't reuse the original request object because the data in
			// the body is already consumed at this point.
			req, e := c.fallbackApiCall(requestData)
			if e != nil {
				return e
			}
			res, e = c.fallbackHttpClient.Do(req)
			if e != nil {
				return e
			}
		} else {
			// If status code is >= 400 and we don't have fallback configured, we
			// can get into this branch without an error set. In that case we want
			// to continue and return the response object to the caller.
			if e != nil {
				return e
			}
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
	requestData := RequestData{
		method: "GET",
		path:   path,
		data:   nil,
		query:  query,
	}

	return c.doRequest(requestData, output, false)
}

func (c *OsoClientImpl) post(path string, data interface{}, output interface{}, isMutation bool) error {
	requestData := RequestData{
		method: "POST",
		path:   path,
		data:   data,
		query:  nil,
	}
	return c.doRequest(requestData, output, isMutation)
}

func (c *OsoClientImpl) delete(path string, data interface{}, output interface{}) error {
	requestData := RequestData{
		method: "DELETE",
		path:   path,
		data:   data,
		query:  nil,
	}
	return c.doRequest(requestData, output, true)
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
	params["predicate"] = data.Predicate
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
