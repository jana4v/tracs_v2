package gateway

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// simHTTPClient is a shared HTTP client used for all sim proxy calls.
// A 5-second timeout prevents gateway goroutines from blocking indefinitely
// when the ingest sim API is unreachable, which would otherwise cause the
// browser to report "Failed to fetch" for concurrent parallel requests.
var simHTTPClient = &http.Client{Timeout: 5 * time.Second}

// SimProxyHandler proxies simulation API calls from the gateway to the ingest
// service's sim API (default :8082).  This allows the GUI and other REST clients
// to use the gateway as the single entry point rather than calling ingest directly.
type SimProxyHandler struct {
	ingestSimURL string // e.g. "http://localhost:8082"
	logger       *slog.Logger
}

// PutStreamValues handles PUT /sim/streams/{streamID}/values
// and proxies to PUT {ingestSimURL}/streams/{streamID}/values.
func (h *SimProxyHandler) PutStreamValues(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(chi.URLParam(r, "streamID"))
	h.proxy(w, r, http.MethodPut, h.ingestSimURL+"/streams/"+streamID+"/values")
}

// GetStreams handles GET /sim/streams and proxies to GET {ingestSimURL}/streams.
func (h *SimProxyHandler) GetStreams(w http.ResponseWriter, r *http.Request) {
	h.proxy(w, r, http.MethodGet, h.ingestSimURL+"/streams")
}

// GetStream handles GET /sim/streams/{streamID} and proxies to
// GET {ingestSimURL}/streams/{streamID}.
func (h *SimProxyHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(chi.URLParam(r, "streamID"))
	h.proxy(w, r, http.MethodGet, h.ingestSimURL+"/streams/"+streamID)
}

// proxy forwards the request to the ingest sim API and streams the response back.
func (h *SimProxyHandler) proxy(w http.ResponseWriter, r *http.Request, method, targetURL string) {
	req, err := http.NewRequestWithContext(r.Context(), method, targetURL, r.Body)
	if err != nil {
		h.logger.Error("sim proxy: failed to build upstream request", "error", err)
		writeError(w, http.StatusInternalServerError, "internal proxy error")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := simHTTPClient.Do(req)
	if err != nil {
		h.logger.Warn("sim proxy: ingest sim API unreachable",
			"url", targetURL, "error", err)
		writeError(w, http.StatusBadGateway, "ingest sim API unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Warn("sim proxy: error copying upstream response body", "url", targetURL, "error", err)
	}
}
