package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/SolaTyolo/herald/pkg/contract"
)

// deliverDirect handles POST /v1/delivery — generic sync channel send.
// Does not create workflow notifications or jobs; forces remote async=false.
// Business adaptors (GoTrue OTP, etc.) live on the remote Delivery Worker, not here.
func (a *API) deliverDirect(w http.ResponseWriter, r *http.Request) {
	if a.delivery == nil {
		writeError(w, http.StatusServiceUnavailable, "delivery service unavailable")
		return
	}
	var req contract.DeliveryRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, contract.Fail("bad_request", err.Error(), false))
		return
	}
	if strings.TrimSpace(req.ProviderName) == "" {
		writeJSON(w, http.StatusBadRequest, contract.Fail("bad_request", "providerName required", false))
		return
	}
	if req.TaskID == "" {
		if id := chimw.GetReqID(r.Context()); id != "" {
			req.TaskID = id
		} else if id := strings.TrimSpace(r.Header.Get("X-Request-Id")); id != "" {
			req.TaskID = id
		}
	}

	de := a.delivery.DeliverDirect(r.Context(), req)
	status := http.StatusOK
	if !de.Success && (de.Code == "bad_request" || de.Code == "unknown_provider" || de.Code == "unsupported_provider") {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, de)
}
