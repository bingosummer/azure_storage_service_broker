package autorest

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/azure/go-autorest/autorest/mocks"
)

func ExampleWithErrorUnlessOK() {
	r := mocks.NewResponse()
	r.Request = mocks.NewRequest()

	// Respond and leave the response body open (for a subsequent responder to close)
	err := Respond(r,
		WithErrorUnlessOK(),
		ByClosingIfError())

	if err == nil {
		fmt.Printf("%s of %s returned HTTP 200", r.Request.Method, r.Request.URL)

		// Complete handling the response and close the body
		Respond(r,
			ByClosing())
	}
	// Output: GET of https://microsoft.com/a/b/c/ returned HTTP 200
}

func ExampleByUnmarshallingJSON() {
	c := `
	{
		"name" : "Rob Pike",
		"age"  : 42
	}
	`

	type V struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	v := &V{}

	Respond(mocks.NewResponseWithContent(c),
		ByUnmarshallingJSON(v),
		ByClosing())

	fmt.Printf("%s is %d years old\n", v.Name, v.Age)
	// Output: Rob Pike is 42 years old
}

func TestCreateResponderDoesNotModify(t *testing.T) {
	r1 := mocks.NewResponse()
	r2 := mocks.NewResponse()
	p := CreateResponder()
	err := p.Respond(r1)
	if err != nil {
		t.Errorf("autorest: CreateResponder failed (%v)", err)
	}
	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("autorest: CreateResponder without decorators modified the response")
	}
}

func TestCreateResponderRunsDecoratorsInOrder(t *testing.T) {
	s := ""

	d := func(n int) RespondDecorator {
		return func(r Responder) Responder {
			return ResponderFunc(func(resp *http.Response) error {
				err := r.Respond(resp)
				if err == nil {
					s += fmt.Sprintf("%d", n)
				}
				return err
			})
		}
	}

	p := CreateResponder(d(1), d(2), d(3))
	err := p.Respond(&http.Response{})
	if err != nil {
		t.Errorf("autorest: Respond failed (%v)", err)
	}

	if s != "123" {
		t.Errorf("autorest: CreateResponder invoked decorators in an incorrect order; expected '123', received '%s'", s)
	}
}

func TestByIgnoring(t *testing.T) {
	r := mocks.NewResponse()

	Respond(r,
		(func() RespondDecorator {
			return func(r Responder) Responder {
				return ResponderFunc(func(r2 *http.Response) error {
					r1 := mocks.NewResponse()
					if !reflect.DeepEqual(r1, r2) {
						t.Errorf("autorest: ByIgnoring modified the HTTP Response -- received %v, expected %v", r2, r1)
					}
					return nil
				})
			}
		})(),
		ByIgnoring(),
		ByClosing())
}

func TestByClosing(t *testing.T) {
	r := mocks.NewResponse()
	err := Respond(r, ByClosing())
	if err != nil {
		t.Errorf("autorest: ByClosing failed (%v)", err)
	}
	if r.Body.(*mocks.Body).IsOpen() {
		t.Errorf("autorest: ByClosing did not close the response body")
	}
}

func TestByClosingAcceptsNilResponse(t *testing.T) {
	r := mocks.NewResponse()

	Respond(r,
		(func() RespondDecorator {
			return func(r Responder) Responder {
				return ResponderFunc(func(resp *http.Response) error {
					resp.Body.Close()
					r.Respond(nil)
					return nil
				})
			}
		})(),
		ByClosing())
}

func TestByClosingAcceptsNilBody(t *testing.T) {
	r := mocks.NewResponse()

	Respond(r,
		(func() RespondDecorator {
			return func(r Responder) Responder {
				return ResponderFunc(func(resp *http.Response) error {
					resp.Body.Close()
					resp.Body = nil
					r.Respond(resp)
					return nil
				})
			}
		})(),
		ByClosing())
}

func TestByClosingClosesEvenAfterErrors(t *testing.T) {
	var e error

	r := mocks.NewResponse()
	Respond(r,
		withErrorRespondDecorator(&e),
		ByClosing())

	if r.Body.(*mocks.Body).IsOpen() {
		t.Errorf("autorest: ByClosing did not close the response body after an error occurred")
	}
}

func TestByClosingClosesReturnsNestedErrors(t *testing.T) {
	var e error

	r := mocks.NewResponse()
	err := Respond(r,
		withErrorRespondDecorator(&e),
		ByClosing())

	if err == nil || !reflect.DeepEqual(e, err) {
		t.Errorf("autorest: ByClosing failed to return a nested error")
	}
}

func TestByClosingIfErrorAcceptsNilResponse(t *testing.T) {
	var e error

	r := mocks.NewResponse()

	Respond(r,
		withErrorRespondDecorator(&e),
		(func() RespondDecorator {
			return func(r Responder) Responder {
				return ResponderFunc(func(resp *http.Response) error {
					resp.Body.Close()
					r.Respond(nil)
					return nil
				})
			}
		})(),
		ByClosingIfError())
}

func TestByClosingIfErrorAcceptsNilBody(t *testing.T) {
	var e error

	r := mocks.NewResponse()

	Respond(r,
		withErrorRespondDecorator(&e),
		(func() RespondDecorator {
			return func(r Responder) Responder {
				return ResponderFunc(func(resp *http.Response) error {
					resp.Body.Close()
					resp.Body = nil
					r.Respond(resp)
					return nil
				})
			}
		})(),
		ByClosingIfError())
}

func TestByClosingIfErrorClosesIfAnErrorOccurs(t *testing.T) {
	var e error

	r := mocks.NewResponse()
	Respond(r,
		withErrorRespondDecorator(&e),
		ByClosingIfError())

	if r.Body.(*mocks.Body).IsOpen() {
		t.Errorf("autorest: ByClosingIfError did not close the response body after an error occurred")
	}
}

func TestByClosingIfErrorDoesNotClosesIfNoErrorOccurs(t *testing.T) {
	r := mocks.NewResponse()
	Respond(r,
		ByClosingIfError())

	if !r.Body.(*mocks.Body).IsOpen() {
		t.Errorf("autorest: ByClosingIfError closed the response body even though no error occurred")
	}
}

func TestByUnmarhallingJSON(t *testing.T) {
	v := &mocks.T{}
	r := mocks.NewResponseWithContent(jsonT)
	err := Respond(r,
		ByUnmarshallingJSON(v),
		ByClosing())
	if err != nil {
		t.Errorf("autorest: ByUnmarshallingJSON failed (%v)", err)
	}
	if v.Name != "Rob Pike" || v.Age != 42 {
		t.Errorf("autorest: ByUnmarshallingJSON failed to properly unmarshal")
	}
}

func TestRespondAcceptsNullResponse(t *testing.T) {
	err := Respond(nil)
	if err != nil {
		t.Errorf("autorest: Respond returned an unexpected error when given a null Response (%v)", err)
	}
}

func TestWithErrorUnlessStatusCode(t *testing.T) {
	r := mocks.NewResponse()
	r.Request = mocks.NewRequest()
	r.Status = "400 BadRequest"
	r.StatusCode = http.StatusBadRequest

	err := Respond(r,
		WithErrorUnlessStatusCode(http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError),
		ByClosingIfError())

	if err != nil {
		t.Errorf("autorest: WithErrorUnlessStatusCode returned an error (%v) for an acceptable status code (%s)", err, r.Status)
	}
}

func TestWithErrorUnlessStatusCodeEmitsErrorForUnacceptableStatusCode(t *testing.T) {
	r := mocks.NewResponse()
	r.Request = mocks.NewRequest()
	r.Status = "400 BadRequest"
	r.StatusCode = http.StatusBadRequest

	err := Respond(r,
		WithErrorUnlessStatusCode(http.StatusOK, http.StatusUnauthorized, http.StatusInternalServerError),
		ByClosingIfError())

	if err == nil {
		t.Errorf("autorest: WithErrorUnlessStatusCode failed to return an error for an unacceptable status code (%s)", r.Status)
	}
}

func TestWithErrorUnlessOK(t *testing.T) {
	r := mocks.NewResponse()
	r.Request = mocks.NewRequest()

	err := Respond(r,
		WithErrorUnlessOK(),
		ByClosingIfError())

	if err != nil {
		t.Errorf("autorest: WithErrorUnlessOK returned an error for OK status code (%v)", err)
	}
}

func TestWithErrorUnlessOKEmitsErrorIfNotOK(t *testing.T) {
	r := mocks.NewResponse()
	r.Request = mocks.NewRequest()
	r.Status = "400 BadRequest"
	r.StatusCode = http.StatusBadRequest

	err := Respond(r,
		WithErrorUnlessOK(),
		ByClosingIfError())

	if err == nil {
		t.Errorf("autorest: WithErrorUnlessOK failed to return an error for a non-OK status code (%v)", err)
	}
}
