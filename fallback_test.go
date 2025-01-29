package oso

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/dhuan/mock/pkg/mock"
)

const (
	MOCK_SERVER_HOST = "localhost"
	MOCK_SERVER_PORT = "4000"
)

type TestServerType int

const (
	USE_MOCK_SERVER TestServerType = iota
	USE_REAL_SERVER TestServerType = iota
)

func resetMockServer() {
	url := url.URL{
		Scheme: "http",
		Host:   strings.Join([]string{MOCK_SERVER_HOST, MOCK_SERVER_PORT}, ":"),
		Path:   "__mock__/reset",
	}
	r, err := http.NewRequest("POST", url.String(), nil)
	if err != nil {
		panic(err)
	}

	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
}

func assertCalled(method mock.ConditionValue, path string, count uint, t *testing.T) {
	mockConfig := &mock.MockConfig{Url: strings.Join([]string{MOCK_SERVER_HOST, MOCK_SERVER_PORT}, ":")}
	validationErrors, err := mock.Assert(mockConfig, &mock.AssertOptions{
		Route: path,
		Condition: &mock.Condition{
			Type:  mock.ConditionType_MethodMatch,
			Value: method,
		},
	})
	if err != nil {
		t.Error(err)
	}

	if count == 0 {
		for _, ve := range validationErrors {
			if ve.Code == mock.ValidationErrorCode_NoCall {
				return
			}
		}
		t.Error("expected endpoint not to be called")
	} else {
		if len(validationErrors) > 0 {
			t.Error(mock.ToReadableError(validationErrors))
		}
	}
}

type OsoTestClients struct {
	valid       OsoClientImpl
	unreachable OsoClientImpl
	httpError   OsoClientImpl
	http300     OsoClientImpl
	http400     OsoClientImpl
	http404     OsoClientImpl
}

// Get some test clients.
//
// Supports two cases.
//
// In one case we just want to test that the fallback URL is called, to test
// that all endpoints that we expect to fall back actually do. In this case we
// use the mock test server.
//
// In the other case, we want to use a real functional Oso test server, so
// that we can check that the request actually succeeds when the fallback
// service is running.
func getOsoTestClients(serverType TestServerType) OsoTestClients {
	const realTestServer = "http://localhost:8081"
	const mockTestServer = "http://localhost:4000"
	var testServer string
	if serverType == USE_MOCK_SERVER {
		testServer = mockTestServer
	} else {
		testServer = realTestServer
	}
	const testServer300 = "http://localhost:4000/return-300"
	const testServer404 = "http://localhost:4000/return-404"
	const testServer400 = "http://localhost:4000/return-400"
	const testServer500 = "http://localhost:4000/return-500"
	const testServerNonexistent = "http://localhost:6000"
	const apiKey = "e_0123456789_12345_osotesttoken01xiIn"
	return OsoTestClients{
		valid:       NewClient(testServer, apiKey).(OsoClientImpl),
		unreachable: NewClientWithFallbackUrl(testServerNonexistent, apiKey, testServer).(OsoClientImpl),
		httpError:   NewClientWithFallbackUrl(testServer500, apiKey, testServer).(OsoClientImpl),
		http300:     NewClientWithFallbackUrl(testServer300, apiKey, testServer).(OsoClientImpl),
		http400:     NewClientWithFallbackUrl(testServer400, apiKey, testServer).(OsoClientImpl),
		http404:     NewClientWithFallbackUrl(testServer404, apiKey, testServer).(OsoClientImpl),
	}
}

type FallbackTestCase struct {
	client         OsoClient
	expected_count uint
}

func getFallbackEligibilityTestCases() []FallbackTestCase {
	testClients := getOsoTestClients(USE_MOCK_SERVER)
	const apiKey = "e_0123456789_12345_osotesttoken01xiIn"
	return []FallbackTestCase{
		// NOTE: Fallback will be called after one 400 error because it does not
		// retry.
		{testClients.http404, 0},
		{testClients.http400, 1},
		{testClients.httpError, 1},
		{testClients.unreachable, 1},
		{testClients.http300, 0},
	}
}

func user() Value {
	return Value{Type: "User", ID: "bob"}
}

func repo() Value {
	return Value{Type: "Repo", ID: "acme"}
}

func Test_AuthorizeFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.Authorize(
			user(), "read", repo(),
		)
		assertCalled("post", "api/authorize", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_ListFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.List(user(), "read", "Repo", []Fact{})
		assertCalled("post", "api/list", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_ActionsFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.Actions(
			user(), repo(),
		)
		assertCalled("post", "api/actions", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_BuildQueryFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		var mapping map[string][]string
		repoVar := TypedVar("Repo")
		action := TypedVar("String")
		tc.client.BuildQuery(NewQueryFact(
			"allow", user(), String("read"), repo(),
		),
		).Evaluate(&mapping, map[Variable]Variable{repoVar: action})
		assertCalled("post", "api/evaluate_query", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_AuthorizeLocalFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.AuthorizeLocal(
			user(), "read", repo(),
		)
		assertCalled("post", "api/authorize_query", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_ListLocalFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.ListLocal(user(), "read", "repo()", "id")
		assertCalled("post", "api/list_query", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_ActionsLocalFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.ActionsLocal(
			user(), repo(),
		)
		assertCalled("post", "api/actions_query", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_GetFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.Get(NewFactPattern("has_role", user(), nil, repo()))
		assertCalled("get", "api/facts", tc.expected_count, t)
		resetMockServer()
	}
}

func Test_PolicyMetadataFallback(t *testing.T) {
	testCases := getFallbackEligibilityTestCases()
	for _, tc := range testCases {
		tc.client.GetPolicyMetadata()
		assertCalled("get", "api/policy_metadata", tc.expected_count, t)
		resetMockServer()
	}
}

func getTestFacts() []Fact {
	return []Fact{
		{
			"has_permission",
			[]Value{
				{Type: "User", ID: "bob"},
				String("read"),
				{Type: "Repo", ID: "acme"},
			},
		},
		{
			"has_permission",
			[]Value{
				{Type: "User", ID: "alice"},
				String("read"),
				{Type: "Repo", ID: "acme"},
			},
		},
	}
}

var initializeServer sync.Once

func getRealServerClients(t *testing.T) OsoTestClients {
	testClients := getOsoTestClients(USE_REAL_SERVER)
	initializeServer.Do(func() {
		testClients.valid.clearData()
		err := testClients.valid.Batch(func(tx BatchTransaction) {
			for _, fact := range getTestFacts() {
				tx.Insert(fact)
			}
		})
		if err != nil {
			t.Fatal("Failed to do test server setup: ", err)
		}
	})
	return testClients
}

func Test_BatchFails(t *testing.T) {
	testClients := getRealServerClients(t)
	err := testClients.unreachable.Batch(func(tx BatchTransaction) {
		tx.Insert(Fact{
			"has_permission",
			[]Value{
				{Type: "User", ID: "alice"},
				String("read"),
				{Type: "Repo", ID: "acme"},
			},
		})
	})
	if err == nil {
		t.Fatal("Expected insert to fail")
	}
	assertCalled("post", "api/batch", 0, t)
	resetMockServer()
}

func Test_AuthorizeIfOsoIsUnreachable(t *testing.T) {
	testClients := getRealServerClients(t)
	res, err := testClients.unreachable.Authorize(
		Value{Type: "User", ID: "bob"},
		"read",
		Value{Type: "Repo", ID: "acme"},
	)
	if err != nil {
		t.Fatal("Expected authorize to succeed: ", err)
	}
	if res != true {
		t.Fatal("Expected authorize to be true")
	}
}

func Test_GetIfOsoIsUnreachable(t *testing.T) {
	testClients := getRealServerClients(t)
	perms, err := testClients.unreachable.Get(
		NewFactPattern("has_permission", nil, String("read"), nil),
	)
	if err != nil {
		t.Fatal("Expected get to succeed")
	}
	if len(perms) != len(getTestFacts()) {
		t.Fatal("Expected permissions length to equal length of all our facts")
	}
}

func Test_AuthorizeIfOsoReturnsHttpError(t *testing.T) {
	testClients := getRealServerClients(t)
	res, err := testClients.httpError.Authorize(
		Value{Type: "User", ID: "bob"},
		"read",
		Value{Type: "Repo", ID: "acme"},
	)
	if err != nil {
		t.Fatal("Expected authorize to succeed: ", err)
	}
	if res != true {
		t.Fatal("Expected authorize to be true")
	}
}

func Test_GetIfOsoReturnsHttpError(t *testing.T) {
	testClients := getRealServerClients(t)
	perms, err := testClients.unreachable.Get(
		NewFactPattern("has_permission", nil, String("read"), nil),
	)
	if err != nil {
		t.Fatal("Expected get to succeed")
	}
	if len(perms) != len(getTestFacts()) {
		t.Fatal("Expected permissions length to equal length of all our facts")
	}
}

func Test_AuthorizeIfOsoReturnsHttp400(t *testing.T) {
	testClients := getRealServerClients(t)
	res, err := testClients.http400.Authorize(
		Value{Type: "User", ID: "bob"},
		"read",
		Value{Type: "Repo", ID: "acme"},
	)
	if err != nil {
		t.Fatal("Expected authorize to succeed: ", err)
	}
	if res != true {
		t.Fatal("Expected authorize to be true")
	}
}

func Test_GetIfOsoReturnsHttp400(t *testing.T) {
	testClients := getRealServerClients(t)
	perms, err := testClients.http400.Get(
		NewFactPattern("has_permission", nil, String("read"), nil),
	)
	if err != nil {
		t.Fatal("Expected get to succeed")
	}
	if len(perms) != len(getTestFacts()) {
		t.Fatal("Expected permissions length to equal length of all our facts")
	}
}

func Test_FallbackEndToEnd(t *testing.T) {
	oso := NewClientWithFallbackUrl("http://localhost:6000", "e_0123456789_12345_osotesttoken01xiIn", "http://localhost:8081")

	user := Value{Type: "User", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++
	acme := Value{Type: "Repo", ID: fmt.Sprintf("%v", idCounter)}
	idCounter++

	t.Run("tell", func(t *testing.T) {
		e := oso.Insert(NewFact("has_permission", user, String("read"), acme))
		if e == nil {
			t.Fatalf("Insert should fail because it is not supported by fallback")
		}
	})

	t.Run("authorize", func(t *testing.T) {
		result, e := oso.AuthorizeWithContext(user, "read", acme, []Fact{
			{
				Predicate: "has_permission",
				Args:      []Value{user, String("read"), acme},
			},
		})
		if e != nil || result != true {
			t.Fatalf("Expect authorize to succeed")
		}
	})
}
