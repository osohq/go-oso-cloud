package oso

import (
	"os"
	"testing"
)

func setupParityTest() (OsoClient, Value, Value, Value) {
	oso := NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	oso.Policy(`
		actor User {}
		resource Document {
			permissions = ["read", "write"];
			roles = ["viewer", "editor"];
			"viewer" if "editor";
			"write" if "editor";
			"read" if "viewer";
		}
	`)

	alice := Value{Type: "User", ID: "alice"}
	bob := Value{Type: "User", ID: "bob"}
	doc := Value{Type: "Document", ID: "doc1"}

	oso.Insert(NewFact("has_role", alice, String("editor"), doc))
	oso.Insert(NewFact("has_role", bob, String("viewer"), doc))

	return oso, alice, bob, doc
}

func teardownParityTest(oso OsoClient) {
	oso.Delete(NewFactPattern("has_role", nil, nil, nil))
}

func TestParityHandleSuccess(t *testing.T) {
	oso, alice, _, doc := setupParityTest()
	defer teardownParityTest(oso)

	parityHandle := NewParityHandle()
	err := parityHandle.Expect(true)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}

	_, err = oso.AuthorizeWithOptions(alice, "read", doc, &AuthorizeOptions{
		ParityHandle: parityHandle,
	})
	if err != nil {
		t.Fatalf("AuthorizeWithOptions failed: %v", err)
	}

	osoImpl := oso.(OsoClientImpl)
	expectedResult := expectedResult{
		RequestID: *parityHandle.requestID,
		Expected:  true,
	}

	_, err = osoImpl.postExpectedResult(expectedResult)
	if err != nil {
		t.Fatalf("postExpectedResult failed: %v", err)
	}

}

func TestExpectAfterAuthorize(t *testing.T) {
	oso, alice, _, doc := setupParityTest()
	defer teardownParityTest(oso)

	parityHandle := NewParityHandle()

	_, err := oso.AuthorizeWithOptions(alice, "read", doc, &AuthorizeOptions{
		ParityHandle: parityHandle,
	})
	if err != nil {
		t.Fatalf("AuthorizeWithOptions failed: %v", err)
	}

	err = parityHandle.Expect(true)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}

}

func TestDoubleExpectRaisesError(t *testing.T) {
	parityHandle := NewParityHandle()

	err := parityHandle.Expect(true)
	if err != nil {
		t.Fatalf("First Expect failed: %v", err)
	}

	err = parityHandle.Expect(false)
	if err == nil {
		t.Fatalf("Expected error for double expect, got nil")
	}
}

func TestOneRequestPerHandle(t *testing.T) {
	oso, alice, bob, doc := setupParityTest()
	defer teardownParityTest(oso)

	parityHandle := NewParityHandle()

	_, err := oso.AuthorizeWithOptions(alice, "read", doc, &AuthorizeOptions{
		ParityHandle: parityHandle,
	})
	if err != nil {
		t.Fatalf("First AuthorizeWithOptions failed: %v", err)
	}

	_, err = oso.AuthorizeWithOptions(bob, "write", doc, &AuthorizeOptions{
		ParityHandle: parityHandle,
	})
	if err == nil {
		t.Fatalf("Expected error for second request, got: %v", err)
	}
}

func TestParityHandlesAreSeparate(t *testing.T) {
	parityHandle1 := NewParityHandle()
	parityHandle2 := NewParityHandle()

	err := parityHandle1.Expect(true)
	if err != nil {
		t.Fatalf("parityHandle1.Expect(true) failed: %v", err)
	}

	err = parityHandle2.Expect(false)
	if err != nil {
		t.Fatalf("parityHandle2.Expect(false) failed: %v", err)
	}
}

func setupAuthLocalParityTest() (OsoClient, Value, Value, Value) {
	oso := NewClientWithDataBindings("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn", "../../feature/src/tests/data_bindings/oso_control.yaml")

	policy, err := os.ReadFile("../../feature/src/tests/policies/oso_control.polar")
	if err != nil {
		panic(err)
	}
	err = oso.Policy(string(policy))
	if err != nil {
		panic(err)
	}

	alice := Value{Type: "User", ID: "alice"}
	bob := Value{Type: "User", ID: "bob"}
	environment := Value{Type: "Environment", ID: "1"}
	tenant := Value{Type: "Tenant", ID: "1"}

	oso.Insert(NewFact("has_role", alice, String("member"), tenant))
	oso.Insert(NewFact("is_god", bob))

	return oso, alice, bob, environment
}

func teardownAuthLocalParityTest(oso OsoClient) {
	alice := Value{Type: "User", ID: "alice"}
	bob := Value{Type: "User", ID: "bob"}
	tenant := Value{Type: "Tenant", ID: "1"}

	oso.Delete(NewFactPattern("has_role", alice, String("member"), tenant))
	oso.Delete(NewFactPattern("is_god", bob))
}

func TestAuthLocalWithParityHandle(t *testing.T) {
	oso, _, bob, environment := setupAuthLocalParityTest()
	defer teardownAuthLocalParityTest(oso)

	parityHandle := NewParityHandle()
	sql, err := oso.AuthorizeLocalWithOptions(bob, "read", environment, &AuthorizeOptions{
		ParityHandle: parityHandle,
	})
	parityHandle.Expect(true)

	

	if err != nil {
		t.Fatalf("AuthorizeLocalWithOptions failed: %v", err)
	}

	if sql == "" {
		t.Fatalf("Expected non-empty SQL query")
	}
}
