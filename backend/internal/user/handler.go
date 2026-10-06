package user

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

// maxSearchRunes limita il testo di ricerca nell'elenco utenti.
const maxSearchRunes = 100

// Response è la rappresentazione JSON di un utente. I campi sono elencati
// esplicitamente: PasswordHash e i dati interni non escono mai dall'API.
type Response struct {
	ID          string     `json:"id"`
	RoleID      string     `json:"role_id"`
	RoleName    string     `json:"role_name"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	DateOfBirth string     `json:"date_of_birth"` // AAAA-MM-GG
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// DeactivatedAt è valorizzato solo per gli utenti disattivati.
	DeactivatedAt *time.Time `json:"deactivated_at"`
}

func NewResponse(u User) Response {
	return Response{
		ID:            u.ID,
		RoleID:        u.RoleID,
		RoleName:      u.RoleName,
		Username:      u.Username,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		DateOfBirth:   u.DateOfBirth.Format(time.DateOnly),
		Status:        u.Status,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		DeactivatedAt: u.DeletedAt,
	}
}

// pageResponse è una pagina dell'elenco utenti.
type pageResponse struct {
	Users    []Response `json:"users"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// profileRequest sono i campi comuni a creazione e modifica. La data di
// nascita arriva come stringa AAAA-MM-GG.
type profileRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
}

type createRequest struct {
	RoleID string `json:"role_id"` // facoltativo, solo per l'Admin
	profileRequest
}

type changeRoleRequest struct {
	RoleID string `json:"role_id"`
}

type Handler struct {
	svc    *Service
	logger *slog.Logger
}

func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes collega le rotte degli utenti al mux. requireAuth lascia
// passare gli utenti autenticati, requireAdmin solo gli Admin: sono passati
// da main per non far dipendere user dal pacchetto auth.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	mux.Handle("GET /api/users", requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/users", requireAuth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/users/{id}", requireAuth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/users/{id}", requireAuth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/users/{id}", requireAdmin(http.HandlerFunc(h.deactivate)))
	mux.Handle("POST /api/users/{id}/reactivate", requireAuth(http.HandlerFunc(h.reactivate)))
	mux.Handle("PUT /api/users/{id}/role", requireAdmin(http.HandlerFunc(h.changeRole)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	search := query.Get("q")
	if len([]rune(search)) > maxSearchRunes {
		httpx.Error(w, http.StatusBadRequest, "testo di ricerca troppo lungo")
		return
	}

	// Valori di pagina mancanti o non numerici valgono 0: il service li
	// riconduce a quelli predefiniti.
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	result, err := h.svc.List(r.Context(), ListInput{
		Search:      search,
		Deactivated: query.Get("deactivated") == "true",
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	users := make([]Response, 0, len(result.Users))
	for _, u := range result.Users {
		users = append(users, NewResponse(u))
	}

	httpx.JSON(w, http.StatusOK, pageResponse{
		Users:    users,
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

	if req.RoleID != "" && !httpx.IsUUID(req.RoleID) {
		h.writeError(w, r, validation.Field("role_id", "ruolo inesistente"))
		return
	}

	actor, _ := identity.FromContext(r.Context())

	created, err := h.svc.Create(r.Context(), actor.Role, CreateInput{
		RoleID:      req.RoleID,
		Username:    req.Username,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DateOfBirth: parseDate(req.DateOfBirth),
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

	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(u))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req profileRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	updated, err := h.svc.Update(r.Context(), id, UpdateInput{
		Username:    req.Username,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DateOfBirth: parseDate(req.DateOfBirth),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(updated))
}

func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	actor, _ := identity.FromContext(r.Context())

	if err := h.svc.Deactivate(r.Context(), actor.UserID, actor.Role, id); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	u, err := h.svc.Reactivate(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(u))
}

func (h *Handler) changeRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req changeRoleRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	if !httpx.IsUUID(req.RoleID) {
		h.writeError(w, r, validation.Field("role_id", "ruolo inesistente"))
		return
	}

	actor, _ := identity.FromContext(r.Context())

	updated, err := h.svc.ChangeRole(r.Context(), actor.Role, id, req.RoleID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, NewResponse(updated))
}

// pathID legge l'id dal percorso; un id malformato è un utente inesistente.
func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !httpx.IsUUID(id) {
		httpx.Error(w, http.StatusNotFound, "utente non trovato")
		return "", false
	}

	return id, true
}

// parseDate interpreta una data AAAA-MM-GG. Una data mancante o malformata
// diventa la data zero, che la validazione del service segnala insieme agli
// altri campi non validi.
func parseDate(value string) time.Time {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}
	}

	return date
}

// writeError traduce gli errori del service nella risposta HTTP.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error

	switch {
	case errors.As(err, &validationErr):
		httpx.FieldErrors(w, validationErr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "utente non trovato")
	case errors.Is(err, ErrAlreadyExists):
		httpx.Error(w, http.StatusConflict, "username o email già usati da un altro utente")
	case errors.Is(err, ErrInvalidRole):
		httpx.FieldErrors(w, map[string]string{"role_id": "ruolo inesistente"})
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "solo un amministratore può assegnare o modificare i ruoli")
	case errors.Is(err, ErrDeactivationForbidden):
		httpx.Error(w, http.StatusForbidden, "solo un amministratore può disattivare gli utenti")
	case errors.Is(err, ErrSelfDeactivation):
		httpx.Error(w, http.StatusForbidden, "non puoi disattivare il tuo account")
	default:
		httpx.InternalError(w, r, h.logger, err)
	}
}
