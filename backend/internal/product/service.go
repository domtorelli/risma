package product

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/matteotorelli/risma/backend/internal/mail"
	"github.com/matteotorelli/risma/backend/internal/role"
	"github.com/matteotorelli/risma/backend/internal/user"
	"github.com/matteotorelli/risma/backend/internal/validation"
)

// Paginazione dell'elenco prodotti.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// initialLoadNote è la nota del movimento di carico creato insieme al prodotto.
const initialLoadNote = "Carico iniziale"

type productStore interface {
	ListTypes(ctx context.Context) ([]Type, error)
	FindByID(ctx context.Context, id string) (Product, error)
	List(ctx context.Context, f ListFilter) ([]Product, error)
	Count(ctx context.Context, f ListFilter) (int, error)
	Create(ctx context.Context, p Product, initialNote string) (Product, error)
	Update(ctx context.Context, p Product) (Product, error)
	RecordMovement(ctx context.Context, m Movement) (Product, error)
	SoftDelete(ctx context.Context, id string) error
}

// adminLister trova gli Admin a cui inviare gli avvisi di scorta minima.
type adminLister interface {
	ListByRole(ctx context.Context, roleName string) ([]user.User, error)
}

type mailer interface {
	SendAsync(ctx context.Context, msg mail.Message)
}

// Il compilatore verifica che i tipi concreti soddisfino le interfacce.
var (
	_ productStore = (*Repository)(nil)
	_ adminLister  = (*user.Repository)(nil)
	_ mailer       = (*mail.Service)(nil)
)

type Service struct {
	store  productStore
	admins adminLister
	mailer mailer
	logger *slog.Logger
}

func NewService(store productStore, admins adminLister, mailer mailer, logger *slog.Logger) *Service {
	return &Service{store: store, admins: admins, mailer: mailer, logger: logger}
}

// CreateInput contiene i dati di un nuovo prodotto; InitialQuantity è la
// giacenza iniziale, registrata come carico.
type CreateInput struct {
	TypeID            string
	Name              string
	Description       string
	Price             string
	InitialQuantity   int
	MinStockThreshold int
}

// UpdateInput contiene i dati modificabili di un prodotto. La giacenza non
// c'è: cambia solo con un movimento di carico o scarico.
type UpdateInput struct {
	TypeID            string
	Name              string
	Description       string
	Price             string
	MinStockThreshold int
}

// MovementInput è un carico (MovementIn) o scarico (MovementOut).
type MovementInput struct {
	Type     string
	Quantity int
	Note     string
}

// ListInput contiene ricerca, filtro, ordinamento e paginazione.
// Sort è "name", "price" o "quantity"; Order è "asc" o "desc"; vuoti
// ordinano per nome crescente. Page parte da 1.
type ListInput struct {
	Search   string
	TypeID   string
	Sort     string
	Order    string
	Page     int
	PageSize int
}

// Page è una pagina dell'elenco prodotti con il totale dei risultati.
type Page struct {
	Products []Product
	Total    int
	Page     int
	PageSize int
}

// ListTypes restituisce i tipi di prodotto.
func (s *Service) ListTypes(ctx context.Context) ([]Type, error) {
	types, err := s.store.ListTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing product types: %w", err)
	}

	return types, nil
}

// Get restituisce un prodotto non eliminato.
func (s *Service) Get(ctx context.Context, id string) (Product, error) {
	p, err := s.store.FindByID(ctx, id)
	if err != nil {
		return Product{}, fmt.Errorf("getting product: %w", err)
	}

	return p, nil
}

// List restituisce una pagina di prodotti. Un criterio di ordinamento non
// previsto è un errore di validazione; valori di pagina non validi vengono
// ricondotti a quelli predefiniti.
func (s *Service) List(ctx context.Context, in ListInput) (Page, error) {
	sortBy := in.Sort
	if sortBy == "" {
		sortBy = "name"
	}

	order := in.Order
	if order == "" {
		order = "asc"
	}

	var errs validation.Collector
	if sortBy != "name" && sortBy != "price" && sortBy != "quantity" {
		errs.Add("sort", "ordinamento ammesso: name, price, quantity")
	}
	if order != "asc" && order != "desc" {
		errs.Add("order", "verso ammesso: asc, desc")
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}

	page := max(in.Page, 1)

	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	pageSize = min(pageSize, MaxPageSize)

	filter := ListFilter{
		Search: in.Search,
		TypeID: in.TypeID,
		Order:  sortBy + " " + order,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	total, err := s.store.Count(ctx, filter)
	if err != nil {
		return Page{}, fmt.Errorf("listing products: %w", err)
	}

	products, err := s.store.List(ctx, filter)
	if err != nil {
		return Page{}, fmt.Errorf("listing products: %w", err)
	}

	return Page{Products: products, Total: total, Page: page, PageSize: pageSize}, nil
}

// Create crea un prodotto. La giacenza iniziale, se positiva, viene
// registrata come carico nella stessa transazione. Se il prodotto nasce
// sotto la soglia minima gli Admin ricevono un avviso.
func (s *Service) Create(ctx context.Context, actorID string, in CreateInput) (Product, error) {
	d := details{
		TypeID:            in.TypeID,
		Name:              in.Name,
		Description:       in.Description,
		Price:             in.Price,
		MinStockThreshold: in.MinStockThreshold,
	}.normalize()

	var errs validation.Collector
	d.validate(&errs)
	if in.InitialQuantity < 0 || in.InitialQuantity > MaxQuantity {
		errs.Add("initial_quantity", "valore tra 0 e 1.000.000")
	}
	if err := errs.Err(); err != nil {
		return Product{}, err
	}

	created, err := s.store.Create(ctx, Product{
		CreatorID:         actorID,
		TypeID:            d.TypeID,
		Name:              d.Name,
		Description:       optional(d.Description),
		Price:             d.Price,
		StockQuantity:     in.InitialQuantity,
		MinStockThreshold: d.MinStockThreshold,
	}, initialLoadNote)
	if err != nil {
		return Product{}, fmt.Errorf("creating product: %w", err)
	}

	if created.BelowThreshold() {
		s.notifyLowStock(ctx, created)
	}

	return created, nil
}

// Update modifica i dati di un prodotto; non tocca la giacenza.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (Product, error) {
	d := details(in).normalize()

	var errs validation.Collector
	d.validate(&errs)
	if err := errs.Err(); err != nil {
		return Product{}, err
	}

	updated, err := s.store.Update(ctx, Product{
		ID:                id,
		TypeID:            d.TypeID,
		Name:              d.Name,
		Description:       optional(d.Description),
		Price:             d.Price,
		MinStockThreshold: d.MinStockThreshold,
	})
	if err != nil {
		return Product{}, fmt.Errorf("updating product: %w", err)
	}

	return updated, nil
}

// RecordMovement registra un carico o scarico e aggiorna la giacenza nella
// stessa transazione. Se dopo il movimento la giacenza è sotto la soglia
// minima gli Admin ricevono un avviso, a ogni movimento.
func (s *Service) RecordMovement(ctx context.Context, actorID, productID string, in MovementInput) (Product, error) {
	note := strings.TrimSpace(in.Note)

	var errs validation.Collector
	if in.Type != MovementIn && in.Type != MovementOut {
		errs.Add("type", "scegli carico o scarico")
	}
	if in.Quantity < 1 || in.Quantity > MaxQuantity {
		errs.Add("quantity", "valore tra 1 e 1.000.000")
	}
	if utf8.RuneCountInString(note) > maxNoteRunes {
		errs.Add("note", "al massimo 500 caratteri")
	}
	if err := errs.Err(); err != nil {
		return Product{}, err
	}

	updated, err := s.store.RecordMovement(ctx, Movement{
		ProductID: productID,
		UserID:    actorID,
		Type:      in.Type,
		Quantity:  in.Quantity,
		Note:      optional(note),
	})
	if err != nil {
		return Product{}, fmt.Errorf("recording movement: %w", err)
	}

	if updated.BelowThreshold() {
		s.notifyLowStock(ctx, updated)
	}

	return updated, nil
}

// Delete elimina logicamente un prodotto, anche con giacenza positiva:
// i movimenti restano nello storico.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("deleting product: %w", err)
	}

	return nil
}

// notifyLowStock invia l'avviso di scorta minima a ogni Admin attivo. Il
// movimento è già salvato: un errore qui viene solo registrato nei log,
// perché annullare l'operazione per un'email mancata sarebbe peggio.
func (s *Service) notifyLowStock(ctx context.Context, p Product) {
	admins, err := s.admins.ListByRole(ctx, role.Admin)
	if err != nil {
		s.logger.Error("finding admins for low stock alert", "product_id", p.ID, "error", err)
		return
	}

	for _, admin := range admins {
		s.mailer.SendAsync(ctx, lowStockEmail(admin, p))
	}
}
