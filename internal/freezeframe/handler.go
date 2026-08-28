// Package freezeframe serves the frame-guessing game: a still lifted from a
// film, and a handful of titles to pick between.
package freezeframe

import (
	"bytes"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
)

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) FreezeFrameHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "FreezeFrameHandler"),
		slog.String("request_id", reqId),
	)

	buf := bytes.Buffer{}
	err := view.TemplateCache["freezeFramePage"].ExecuteTemplate(&buf, "base", nil)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
