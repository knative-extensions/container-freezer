package daemon_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "k8s.io/api/authentication/v1"
	"knative.dev/container-freezer/pkg/daemon"
	ltesting "knative.dev/pkg/logging/testing"
)

func TestHandler(t *testing.T) {
	pauseBodyText := fmt.Sprintf(`{ "action": %q }`, "pause")
	pauseBody := bytes.NewBufferString(pauseBodyText)
	resumeBodyText := fmt.Sprintf(`{ "action": %q }`, "resume")
	resumeBody := bytes.NewBufferString(resumeBodyText)
	tt := []struct {
		name         string
		headers      http.Header
		tokens       map[string]*authv1.TokenReview
		expectStatus int
		expectFreeze string
		expectThaw   string
		body         *bytes.Buffer
	}{{
		name:    "no token",
		body:    pauseBody,
		headers: http.Header{},

		expectStatus: 400,
	}, {
		name: "token not valid",
		body: pauseBody,
		tokens: map[string]*authv1.TokenReview{
			"THE_TOKEN": {
				Status: authv1.TokenReviewStatus{
					Authenticated: false,
				},
			},
		},
		headers: http.Header{
			daemon.TokenHeaderKey: []string{"THE_TOKEN"},
		},

		expectStatus: 403,
	}, {
		name: "token validation fails",
		body: pauseBody,
		headers: http.Header{
			daemon.TokenHeaderKey: []string{"THE_TOKEN"},
		},

		expectStatus: 500,
	}, {
		name: "valid token, freeze",
		body: pauseBody,
		tokens: map[string]*authv1.TokenReview{
			"THE_TOKEN": {
				Status: authv1.TokenReviewStatus{
					Authenticated: true,
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							"authentication.kubernetes.io/pod-uid":  {"the-pod-name"},
							"authentication.kubernetes.io/pod-name": {"the-pod-name"},
						},
					},
				},
			},
		},
		headers: http.Header{
			daemon.TokenHeaderKey: []string{"THE_TOKEN"},
		},

		expectStatus: 200,
		expectFreeze: "the-pod-name",
	}, {
		name: "valid token, thaw",
		body: resumeBody,
		tokens: map[string]*authv1.TokenReview{
			"THE_TOKEN": {
				Status: authv1.TokenReviewStatus{
					Authenticated: true,
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							"authentication.kubernetes.io/pod-uid":  {"the-pod-name"},
							"authentication.kubernetes.io/pod-name": {"the-pod-name"},
						},
					},
				},
			},
		},
		headers: http.Header{
			daemon.TokenHeaderKey: []string{"THE_TOKEN"},
		},

		expectStatus: 200,
		expectThaw:   "the-pod-name",
	}}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			logger := ltesting.TestLogger(t)
			var thawed, froze string
			handler := daemon.Handler{
				Logger: logger,
				Validator: daemon.TokenValidatorFunc(func(ctx context.Context, token string) (*authv1.TokenReview, error) {
					resp, ok := test.tokens[token]
					if !ok {
						return nil, errors.New("some error")
					}

					return resp, nil
				}),
				Freezer: FreezeFunc(func(_ context.Context, podName, containerName string) error {
					froze = podName
					return nil
				}),
				Thawer: ThawFunc(func(_ context.Context, podName, containerName string) error {
					thawed = podName
					return nil
				}),
			}

			resp := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/", test.body)
			req.Header = test.headers
			handler.ServeHTTP(resp, req)

			if got, want := resp.Code, test.expectStatus; got != want {
				t.Errorf("Expected response code %v but was %v", want, got)
			}

			if got, want := froze, test.expectFreeze; got != want {
				t.Errorf("Expected frozen to be %q but was %q", want, got)
			}

			if got, want := thawed, test.expectThaw; got != want {
				t.Errorf("Expected thawed to be %q but was %q", want, got)
			}
		})
	}
}

type FreezeFunc func(ctx context.Context, podName, containerName string) error

func (fn FreezeFunc) Freeze(ctx context.Context, podName, containerName string) error {
	return fn(ctx, podName, containerName)
}

type ThawFunc func(ctx context.Context, podName, containerName string) error

func (fn ThawFunc) Thaw(ctx context.Context, podName, containerName string) error {
	return fn(ctx, podName, containerName)
}
