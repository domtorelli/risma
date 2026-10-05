package product

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/matteotorelli/risma/backend/internal/httpx"
	"github.com/matteotorelli/risma/backend/internal/identity"
	"github.com/matteotorelli/risma/backend/internal/validation"
)

// maxSearchRunes limita il testo di ricerca nell'elenco prodotti.
const maxSearchRunes = 100

// Response è la rappresentazione JSON di un prodotto. Il prezzo è una
// stringa decimale (es. "12.50") per non perdere precisione in un float.
type Response struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description"`
	TypeID            string    `json:"product_type_id"`
	TypeName          string    `json:"product_type_name"`
	Price             string    `json:"price"`
	StockQuantity     int       `json:"stock_quantity"`
	MinStockThreshold int       `json:"min_stock_threshold"`
	BelowThreshold    bool      `json:"below_threshold"`
	CreatorID         string    `json:"creator_id"`
	CreatorUsername   string    `json:"creator_username"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewResponse(p Product) Response {
	return Response{
		ID:                p.ID,
		Name:              p.Name,
		Description:       p.Description,
		TypeID:            p.TypeID,
		TypeName:          p.TypeName,
		Price:             p.Price,
		StockQuantity:     p.StockQuantity,
		MinStockThreshold: p.MinStockThreshold,
		BelowThreshold:    p.BelowThreshold(),
		CreatorID:         p.CreatorID,
		CreatorUsername:   p.CreatorUsername,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

type typeResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type pageResponse struct {
	Products []Response `json:"products"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// detailsRequest sono i campi comuni a creazione e modifica.
type detailsRequest struct {
	TypeID            string `json:"product_type_id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Price             string `json:"price"`
	MinStockThreshold int    `json:"min_stock_threshold"`
}

type createRequest struct {
	detailsRequest
	InitialQuantity int `json:"initial_quantity"`
}

type movementRequest struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
	Note     string `json:"note"`
}

type Handler struct {
	svc    *Service
	logger *slog.Logger
}

func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes collega le rotte dell'inventario al mux; requireAuth è
// passato da main per non far dipendere product dal pacchetto auth.
// La gestione dell'inventario è permessa a Admin e Operatore.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/product-types", requireAuth(http.HandlerFunc(h.listTypes)))
	mux.Handle("GET /api/products", requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/products", requireAuth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/products/{id}", requireAuth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/products/{id}", requireAuth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/products/{id}", requireAuth(http.HandlerFunc(h.delete)))
	mux.Handle("POST /api/products/{id}/movements", requireAuth(http.HandlerFunc(h.recordMovement)))
}

func (h *Handler) listTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.svc.ListTypes(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp := make([]typeResponse, 0, len(types))
	for _, t := range types {
		resp = append(resp, typeResponse{ID: t.ID, Name: t.Name, Description: t.Description})
	}

	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	search := query.Get("q")
	if len([]rune(search)) > maxSearchRunes {
		httpx.Error(w, http.StatusBadRequest, "testo di ricerca troppo lungo")
		return
	}

	typeID := query.Get("type_id")
	if typeID != "" && !httpx.IsUUID(typeID) {
		httpx.FieldErrors(w, map[string]string{"type_id": "tipo di prodotto inesistente"})
		return
	}

	// Valori di pagina mancanti o non numerici valgono 0: il service li
	// riconduce a quelli predefiniti.
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	result, err := h.svc.List(r.Context(), ListInput{
		Search:   search,
		TypeID:   typeID,
		Sort:     query.Get("sort"),
		Order:    query.Get("order"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	products := make([]Response, 0, len(result.Products))
	for _, p := range result.Products {
		products = append(products, NewResponse(p))
	}

	httpx.JSON(w, http.StatusOK, pageResponse{
		Products: products,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	if req.TypeID != "" && !httpx.IsUUID(req.TypeID) {
		h.writeError(w, r, ErrInvalidType)
		return
	}

	actor, _ := identity.FromContext(r.Context())

	created, err := h.svc.Create(r.Context(), actor.UserID, CreateInput{
		TypeID:            req.TypeID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		InitialQuantity:   req.InitialQuantity,
		MinStockThreshold: req.MinStockThreshold,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, NewResponse(created))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(p))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req detailsRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	if req.TypeID != "" && !httpx.IsUUID(req.TypeID) {
		h.writeError(w, r, ErrInvalidType)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, UpdateInput{
		TypeID:            req.TypeID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		MinStockThreshold: req.MinStockThreshold,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(updated))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) recordMovement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req movementRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	actor, _ := identity.FromContext(r.Context())

	updated, err := h.svc.RecordMovement(r.Context(), actor.UserID, id, MovementInput{
		Type:     req.Type,
		Quantity: req.Quantity,
		Note:     req.Note,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(updated))
}

// pathID legge l'id dal percorso; un id malformato è un prodotto inesistente.
func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !httpx.IsUUID(id) {
		httpx.Error(w, http.StatusNotFound, "prodotto non trovato")
		return "", false
	}

	return id, true
}

// writeError traduce gli errori del service nella risposta HTTP.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error

	switch {
	case errors.As(err, &validationErr):
		httpx.FieldErrors(w, validationErr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "prodotto non trovato")
	case errors.Is(err, ErrAlreadyExists):
		httpx.FieldErrors(w, map[string]string{"name": "esiste già un prodotto con questo nome"})
	case errors.Is(err, ErrInvalidType):
		httpx.FieldErrors(w, map[string]string{"product_type_id": "tipo di prodotto inesistente"})
	case errors.Is(err, ErrInsufficientStock):
		httpx.FieldErrors(w, map[string]string{"quantity": "quantità disponibile insufficiente per lo scarico"})
	case errors.Is(err, ErrStockNotEmpty):
		httpx.Error(w, http.StatusConflict, "puoi eliminare solo un prodotto con quantità disponibile pari a zero: registra prima uno scarico")
	case errors.Is(err, ErrQuantityTooLarge):
		httpx.FieldErrors(w, map[string]string{"quantity": "la giacenza risultante è troppo grande"})
	default:
		httpx.InternalError(w, r, h.logger, err)
	}
}
