package duel

import (
	"bytes"
	"errors"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
	}
}

func (h *Handler) DuelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "DuelHandler"), slog.String("request_id", reqId))

	d, err := h.service.CreateRandomDuel(ctx)
	if err != nil {
		if errors.Is(err, ErrNotEnoughFilms) {
			log.ErrorContext(ctx, "Duel could not fetch films for a duel",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Could not fetch films for a duel", http.StatusServiceUnavailable)
			return
		}
		log.ErrorContext(ctx, "failed to create duel", slog.String("error", err.Error()))
		http.Error(w, "Could not fetch the films for a duel.", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["duelPage"].ExecuteTemplate(&buf, "base", d)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
