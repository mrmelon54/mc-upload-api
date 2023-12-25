package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (r RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return r(req), nil }

func NewTestServer(mux *http.ServeMux) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			return rec.Result()
		}),
	}
}

func WriteRequestToLine(req *http.Request) string {
	buf := new(bytes.Buffer)
	if err := req.Write(buf); err != nil {
		panic(err)
	}
	return buf.String()
}
