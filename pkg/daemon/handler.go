package daemon

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

const TokenHeaderKey = "Token"

type Handler struct {
	Logger *zap.SugaredLogger
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var m messageBody
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&m)
	if err != nil {
		http.Error(w, "error decoding", http.StatusBadRequest)
	}

	switch m.Action {
	case "pause":
		h.Logger.Info("pause request")
	case "resume":
		h.Logger.Info("resume request")
	default:
		h.Logger.Info("bad request: ", m.Action)
		w.WriteHeader(http.StatusNotFound)
	}
}

type messageBody struct {
	Action string `json:"action"`
}
