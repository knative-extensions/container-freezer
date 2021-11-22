package daemon

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
)

const TokenHeaderKey = "Token"

type TokenValidator interface {
	Validate(ctx context.Context, token string) (*authv1.TokenReview, error)
}

type Freezer interface {
	Freeze(ctx context.Context, podName, containerName string) error
}

type Thawer interface {
	Thaw(ctx context.Context, podName, containerName string) error
}

type FreezeThawer interface {
	Freezer
	Thawer
}

type Handler struct {
	Validator TokenValidator
	Freezer   Freezer
	Thawer    Thawer
	Logger    *zap.SugaredLogger
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if token == "" {
		h.Logger.Error("No token in header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := h.Validator.Validate(r.Context(), token)
	if err != nil {
		h.Logger.Error("Validating token failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !resp.Status.Authenticated {
		h.Logger.Error("Authenticating via token failed")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var m messageBody
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		h.Logger.Error("Unable to decode message body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
	}

	podUid := resp.Status.User.Extra["authentication.kubernetes.io/pod-uid"][0]
	switch m.Action {
	case "pause":
		h.Logger.Info("pause request received, freezing pod: %s", podUid)
		h.Freezer.Freeze(r.Context(), podUid, "user-container")
	case "resume":
		h.Logger.Info("resume request received, thawing pod: %s", podUid)
		h.Thawer.Thaw(r.Context(), podUid, "user-container")
	default:
		h.Logger.Info("invalid action specified: ", m.Action)
		w.WriteHeader(http.StatusNotFound)
	}
}

type messageBody struct {
	Action string `json:"action"`
}

type TokenValidatorFunc func(ctx context.Context, token string) (*authv1.TokenReview, error)

func (fn TokenValidatorFunc) Validate(ctx context.Context, token string) (*authv1.TokenReview, error) {
	return fn(ctx, token)
}
