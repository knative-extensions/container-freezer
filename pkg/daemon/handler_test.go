package daemon_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "k8s.io/api/authentication/v1"
	"knative.dev/container-freezer/pkg/daemon"
)

func TestHandler(t *testing.T) {
	tt := []struct {
		name         string
		path         string
		headers      http.Header
		tokens       map[string]*authv1.TokenReview
		expectStatus int
		expectFreeze string
		expectThaw   string
	}{{
		name:    "no token",
		path:    "/freeze",
		headers: http.Header{},

		expectStatus: 400,
	}, {
		name: "token not valid",
		path: "/freeze",
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
		path: "/freeze",
		headers: http.Header{
			daemon.TokenHeaderKey: []string{"THE_TOKEN"},
		},

		expectStatus: 500,
	}, {
		name: "valid token, freeze",
		path: "/freeze",
		tokens: map[string]*authv1.TokenReview{
			"THE_TOKEN": {
				Status: authv1.TokenReviewStatus{
					Authenticated: true,
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							"authentication.kubernetes.io/pod-uid": {"the-pod-name"},
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
		path: "/thaw",
		tokens: map[string]*authv1.TokenReview{
			"THE_TOKEN": {
				Status: authv1.TokenReviewStatus{
					Authenticated: true,
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							"authentication.kubernetes.io/pod-uid": {"the-pod-name"},
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
			var thawed, froze string
			handler := daemon.Handler{
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
			req := httptest.NewRequest("POST", test.path, nil)
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
