package oso

import (
	"errors"
	"fmt"
)

type expectedResult struct {
	RequestID string `json:"request_id"`
	Expected  bool   `json:"expected"`
}

// ParityHandle is a testing utility in Oso Migrate for comparing expected authorization
// decisions with actual Oso results.
type ParityHandle struct {
	api       *OsoClientImpl
	requestID *string
	expected  *bool
}

func NewParityHandle() *ParityHandle {
	return &ParityHandle{
		api:       nil,
		requestID: nil,
		expected:  nil,
	}
}

// set is an internal method called by the API class after authorize.
//
// Args:
//
//	requestID: The ID of the authorization request.
//	api: Reference to the API instance.
func (p *ParityHandle) set(requestID string, api *OsoClientImpl) error {
	if p.requestID != nil {
		return fmt.Errorf(
			"attempted to set request_id twice. Only one request is allowed per ParityHandle instance. (Original request ID: %v)",
			*p.requestID,
		)
	}

	p.requestID = &requestID
	p.api = api

	if p.expected != nil {
		return p.send()
	}

	return nil
}

// Expect is a public method for users to indicate the expected result of an authorization query.
//
// Args:
//
//	expected: Boolean indicating the expected authorization result.
//
// Returns:
//
//	error: If expected result is set twice.
func (p *ParityHandle) Expect(expected bool) error {
	if p.expected != nil {
		return errors.New("attempted to set expected result twice")
	}

	p.expected = &expected

	if p.requestID != nil {
		return p.send()
	}

	return nil
}

// send sends the expected result to the API
func (p *ParityHandle) send() error {
	if p.api == nil || p.requestID == nil || p.expected == nil {
		return errors.New("ParityHandle not properly initialized")
	}

	expectedResult := expectedResult{
		RequestID: *p.requestID,
		Expected:  *p.expected,
	}

	_, err := p.api.postExpectedResult(expectedResult)
	return err
}
