package role

import (
	"log/slog"
	"net/http"

	"github.com/matteotorelli/risma/backend/internal/httpx"
)

// Response è la rappresentazione JSON di un ruolo.
type Response struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type Handler struct {
	svc    *Service
	logger *slog.Logger
}

func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes collega le rotte dei ruoli al mux. I ruoli sono fissi: le
// API li espongono solo in lettura, per i form di creazione e cambio ruolo.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/roles", requireAuth(http.HandlerFunc(h.list)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		httpx.InternalError(w, r, h.logger, err)
		return
	}

	resp := make([]Response, 0, len(roles))
	for _, ro := range roles {
		resp = append(resp, Response{ID: ro.ID, Name: ro.Name, Description: ro.Description})
	}

	httpx.JSON(w, http.StatusOK, resp)
}
