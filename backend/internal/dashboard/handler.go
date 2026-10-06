package dashboard

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/domtorelli/risma/backend/internal/httpx"
)

type movementResponse struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Quantity       int       `json:"quantity"`
	Note           *string   `json:"note"`
	CreatedAt      time.Time `json:"created_at"`
	ProductID      string    `json:"product_id"`
	ProductName    string    `json:"product_name"`
	ProductDeleted bool      `json:"product_deleted"`
	Username       string    `json:"username"`
}

type typeCountResponse struct {
	TypeID   string `json:"product_type_id"`
	TypeName string `json:"product_type_name"`
	Count    int    `json:"count"`
}

// summaryResponse è la rappresentazione JSON della dashboard. Il valore
// dell'inventario è una stringa decimale, come i prezzi.
type summaryResponse struct {
	TotalUsers      int                 `json:"total_users"`
	TotalProducts   int                 `json:"total_products"`
	InventoryValue  string              `json:"inventory_value"`
	RecentMovements []movementResponse  `json:"recent_movements"`
	ProductsByType  []typeCountResponse `json:"products_by_type"`
}

type Handler struct {
	svc    *Service
	logger *slog.Logger
}

func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes collega la rotta della dashboard al mux; la dashboard è
// visibile a tutti gli utenti autenticati.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/dashboard", requireAuth(http.HandlerFunc(h.summary)))
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Summary(r.Context())
	if err != nil {
		httpx.InternalError(w, r, h.logger, err)
		return
	}

	movements := make([]movementResponse, 0, len(s.RecentMovements))
	for _, m := range s.RecentMovements {
		movements = append(movements, movementResponse(m))
	}

	byType := make([]typeCountResponse, 0, len(s.ProductsByType))
	for _, c := range s.ProductsByType {
		byType = append(byType, typeCountResponse(c))
	}

	httpx.JSON(w, http.StatusOK, summaryResponse{
		TotalUsers:      s.TotalUsers,
		TotalProducts:   s.TotalProducts,
		InventoryValue:  s.InventoryValue,
		RecentMovements: movements,
		ProductsByType:  byType,
	})
}
