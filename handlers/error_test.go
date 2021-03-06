package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	test "github.com/cbsinteractive/bakery/tests"
	"github.com/google/go-cmp/cmp"
)

func TestHandler_ErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		auth         string
		mockResp     func(req *http.Request) (*http.Response, error)
		expectErr    ErrorResponse
		expBody      string
		expectStatus int
	}{
		{
			name: "when manifest returns 4xx, expect 500  w/ err msg reflecting origin status code",
			url:  "origin/some/path/to/master.m3u8",
			auth: "authenticate-me",
			mockResp: func(*http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 400,
					Body:       ioutil.NopCloser(bytes.NewBufferString("")),
					Header:     http.Header{},
				}

				lastModified := time.Now().UTC().Format(http.TimeFormat)
				resp.Header.Add("Last-Modified", lastModified)

				return resp, nil
			},
			expectStatus: 400,
			expectErr: ErrorResponse{
				Message: "manifest origin error",
				Errors: map[string][]string{
					"fetching manifest": []string{"returning http status of 400"},
				},
			},
		},
		{
			name:         "when request is made with bad filters, expect error from parser",
			url:          "/b(10000,10)/origin/some/path/to/master.mpd",
			auth:         "authenticate-me",
			mockResp:     default200Response("OK"),
			expectStatus: 400,
			expectErr: ErrorResponse{
				Message: "failed parsing filters",
				Errors: map[string][]string{
					"Bitrate": []string{"invalid range for provided values", "( 10000, 10 )"},
				},
			},
		},
		{
			name:         "when propeller channel is passed with bad path, expect 500 status code w/ err msg reflecting origin configuration",
			url:          "propeller/master.m3u8",
			auth:         "authenticate-me",
			mockResp:     default200Response("OK"),
			expectStatus: 500,
			expectErr: ErrorResponse{
				Message: "failed configuring origin",
				Errors: map[string][]string{
					"propeller origin": []string{"invalid url format /propeller/master.m3u8"},
				},
			},
		},
		{
			name:         "when request is made without protocol, proper error response is thrown",
			url:          "/some/random/request",
			auth:         "authenticate-me",
			mockResp:     default200Response("OK"),
			expectStatus: 400,
			expectErr: ErrorResponse{
				Message: "failed parsing filters",
				Errors: map[string][]string{
					"Protocol": []string{"unsupported protocol"},
				},
			},
		},
		{
			name:         "when request is made and bad HLS manifest is returned, expect error",
			url:          "origin/some/path/to/master.m3u8",
			auth:         "authenticate-me",
			mockResp:     default200Response("OK"),
			expectStatus: 500,
			expectErr: ErrorResponse{
				Message: "failed to filter manifest",
				Errors: map[string][]string{
					"#EXTM3U absent": []string{},
				},
			},
		},
		{
			name:         "when request is made and bad MPD manifest is returned, expect error",
			url:          "origin/some/path/to/master.mpd",
			auth:         "authenticate-me",
			mockResp:     default200Response("OK"),
			expectStatus: 500,
			expectErr: ErrorResponse{
				Message: "failed to filter manifest",
				Errors: map[string][]string{
					"EOF": []string{},
				},
			},
		},
		{
			name:         "when request is made with bad auth headers, expect authentication error",
			url:          "origin/some/path/to/master.mpd",
			auth:         "bad-token",
			mockResp:     default200Response("OK"),
			expectStatus: 403,
			expectErr:    ErrorResponse{},
			expBody:      "you must pass a valid api token as \"x-bakery-origin-token\"\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := testConfig(test.MockClient(tc.mockResp))
			handler := c.SetupMiddleware().Then(LoadHandler(c))
			// set req + response recorder and serve it
			req := getRequest(tc.url, t)
			req.Header.Set("x-bakery-origin-token", tc.auth)
			rec := getResponseRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.expectStatus {
				t.Errorf("expected status %v; got %v", tc.expectStatus, res.StatusCode)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			var got ErrorResponse
			json.Unmarshal(body, &got)
			if !cmp.Equal(got, tc.expectErr) {
				t.Errorf("Wrong error returned\ngot %v\nexpected: %v\ndiff: %v",
					got, tc.expectErr, cmp.Diff(got, tc.expectErr))
			}

			//authentication executes in a seperate handler
			if tc.expBody != "" {
				if !cmp.Equal(string(body), tc.expBody) {
					t.Errorf("Wrong body returned\ngot %v\nexpected: %v\ndiff: %v",
						string(body), tc.expBody, cmp.Diff(string(body), tc.expBody))
				}
			}
		})
	}

}
