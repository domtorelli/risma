package product

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/matteotorelli/risma/backend/internal/mail"
	"github.com/matteotorelli/risma/backend/internal/role"
	"github.com/matteotorelli/risma/backend/internal/user"
	"github.com/matteotorelli/risma/backend/internal/validation"
)

// fakeStore tiene in memoria un solo prodotto e registra le chiamate.
type fakeStore struct {
	product     Product
	createCalls int
	initialNote string
	movements   []Movement
	lastFilter  ListFilter
}

func (f *fakeStore) ListTypes(ctx context.Context) ([]Type, error) { return nil, nil }

func (f *fakeStore) FindByID(ctx context.Context, id string) (Product, error) {
	return f.product, nil
}

func (f *fakeStore) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	f.lastFilter = filter
	return nil, nil
}

func (f *fakeStore) Count(ctx context.Context, filter ListFilter) (int, error) { return 0, nil }

func (f *fakeStore) Create(ctx context.Context, p Product, initialNote string) (Product, error) {
	f.createCalls++
	f.initialNote = initialNote
	p.ID = "product-1"
	f.product = p
	return p, nil
}

func (f *fakeStore) Update(ctx context.Context, p Product) (Product, error) { return p, nil }

// RecordMovement imita il CHECK del database sulla giacenza.
func (f *fakeStore) RecordMovement(ctx context.Context, m Movement) (Product, error) {
	if f.product.StockQuantity+m.delta() < 0 {
		return Product{}, ErrInsufficientStock
	}
	f.movements = append(f.movements, m)
	f.product.StockQuantity += m.delta()
	return f.product, nil
}

// SoftDelete imita la condizione della query: solo con giacenza zero.
func (f *fakeStore) SoftDelete(ctx context.Context, id string) error {
	if f.product.StockQuantity > 0 {
		return ErrStockNotEmpty
	}
	return nil
}

type fakeAdmins struct {
	requestedRole string
}

func (f *fakeAdmins) ListByRole(ctx context.Context, roleName string) ([]user.User, error) {
	f.requestedRole = roleName
	return []user.User{
		{Email: "admin1@example.com", FirstName: "Anna"},
		{Email: "admin2@example.com", FirstName: "Bruno"},
	}, nil
}

type fakeMailer struct {
	sent []mail.Message
}

func (f *fakeMailer) SendAsync(ctx context.Context, msg mail.Message) {
	f.sent = append(f.sent, msg)
}

func newTestService() (*Service, *fakeStore, *fakeAdmins, *fakeMailer) {
	store := &fakeStore{}
	admins := &fakeAdmins{}
	mailer := &fakeMailer{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewService(store, admins, mailer, logger), store, admins, mailer
}

func validCreateInput() CreateInput {
	return CreateInput{
		TypeID:            "type-1",
		Name:              "Risma A4 80g",
		Price:             "4.90",
		InitialQuantity:   50,
		MinStockThreshold: 10,
	}
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name      string
		edit      func(*CreateInput)
		wantField string // vuoto: input valido
	}{
		{"valid", func(in *CreateInput) {}, ""},
		{"price without decimals", func(in *CreateInput) { in.Price = "12" }, ""},
		{"price with one decimal", func(in *CreateInput) { in.Price = "12.5" }, ""},
		{"price with comma", func(in *CreateInput) { in.Price = "12,50" }, "price"},
		{"price with three decimals", func(in *CreateInput) { in.Price = "1.999" }, "price"},
		{"negative price", func(in *CreateInput) { in.Price = "-1.00" }, "price"},
		{"price too large", func(in *CreateInput) { in.Price = "123456789.00" }, "price"},
		{"empty price", func(in *CreateInput) { in.Price = "" }, "price"},
		{"missing type", func(in *CreateInput) { in.TypeID = "" }, "product_type_id"},
		{"empty name", func(in *CreateInput) { in.Name = "  " }, "name"},
		{"name too long", func(in *CreateInput) { in.Name = strings.Repeat("a", 201) }, "name"},
		{"name with newline", func(in *CreateInput) { in.Name = "Risma\r\nBcc: x@example.com" }, "name"},
		{"description too long", func(in *CreateInput) { in.Description = strings.Repeat("a", 1001) }, "description"},
		{"negative initial quantity", func(in *CreateInput) { in.InitialQuantity = -1 }, "initial_quantity"},
		{"negative threshold", func(in *CreateInput) { in.MinStockThreshold = -1 }, "min_stock_threshold"},
		{"threshold too large", func(in *CreateInput) { in.MinStockThreshold = MaxQuantity + 1 }, "min_stock_threshold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, _ := newTestService()

			in := validCreateInput()
			tt.edit(&in)

			_, err := svc.Create(context.Background(), "user-1", in)

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}

			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("err = %v, want validation.Error", err)
			}

			if _, ok := validationErr.Fields[tt.wantField]; !ok {
				t.Errorf("fields = %v, want %q", validationErr.Fields, tt.wantField)
			}

			if store.createCalls != 0 {
				t.Error("invalid product saved")
			}
		})
	}
}

func TestCreateRecordsCreatorAndInitialLoad(t *testing.T) {
	svc, store, _, _ := newTestService()

	in := validCreateInput()
	in.Description = "  "

	created, err := svc.Create(context.Background(), "user-1", in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if created.CreatorID != "user-1" || created.StockQuantity != 50 || store.initialNote != initialLoadNote {
		t.Errorf("created = %+v, note = %q", created, store.initialNote)
	}

	if created.Description != nil {
		t.Errorf("empty description saved as %q, want NULL", *created.Description)
	}
}

func TestLowStockAlertOnCreate(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int
		threshold int
		wantMails int
	}{
		{"above threshold", 50, 10, 0},
		{"equal to threshold", 10, 10, 0},
		{"below threshold", 5, 10, 2},
		{"no threshold", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, mailer := newTestService()

			in := validCreateInput()
			in.InitialQuantity, in.MinStockThreshold = tt.quantity, tt.threshold

			if _, err := svc.Create(context.Background(), "user-1", in); err != nil {
				t.Fatalf("Create: %v", err)
			}

			if len(mailer.sent) != tt.wantMails {
				t.Errorf("sent %d emails, want %d", len(mailer.sent), tt.wantMails)
			}
		})
	}
}

func TestRecordMovement(t *testing.T) {
	svc, store, admins, mailer := newTestService()
	store.product = Product{ID: "product-1", Name: "Toner nero", StockQuantity: 12, MinStockThreshold: 10}
	ctx := context.Background()

	// 12 - 1 = 11: sopra la soglia, nessun avviso.
	if _, err := svc.RecordMovement(ctx, "user-1", "product-1", MovementInput{Type: MovementOut, Quantity: 1}); err != nil {
		t.Fatalf("first movement: %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Fatalf("sent %d emails above threshold", len(mailer.sent))
	}

	// 11 - 3 = 8: sotto la soglia, avviso a entrambi gli Admin.
	updated, err := svc.RecordMovement(ctx, "user-1", "product-1", MovementInput{Type: MovementOut, Quantity: 3, Note: " vendita "})
	if err != nil {
		t.Fatalf("second movement: %v", err)
	}
	if updated.StockQuantity != 8 || len(mailer.sent) != 2 || admins.requestedRole != role.Admin {
		t.Fatalf("quantity = %d, emails = %d, role = %q", updated.StockQuantity, len(mailer.sent), admins.requestedRole)
	}
	if mailer.sent[0].Type != mail.TypeLowStock || !strings.Contains(mailer.sent[0].Subject, "Toner nero") {
		t.Errorf("email = %+v", mailer.sent[0])
	}
	if note := store.movements[1].Note; note == nil || *note != "vendita" {
		t.Errorf("note not trimmed: %v", note)
	}

	// 8 + 1 = 9: ancora sotto la soglia, nuovo avviso a ogni movimento.
	if _, err := svc.RecordMovement(ctx, "user-1", "product-1", MovementInput{Type: MovementIn, Quantity: 1}); err != nil {
		t.Fatalf("third movement: %v", err)
	}
	if len(mailer.sent) != 4 {
		t.Errorf("sent %d emails, want 4 (one more alert per admin)", len(mailer.sent))
	}

	// Scarico oltre la giacenza: rifiutato, nessun movimento né avviso.
	if _, err := svc.RecordMovement(ctx, "user-1", "product-1", MovementInput{Type: MovementOut, Quantity: 100}); !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}
	if len(store.movements) != 3 || len(mailer.sent) != 4 {
		t.Errorf("movements = %d, emails = %d after rejected movement", len(store.movements), len(mailer.sent))
	}
}

func TestRecordMovementValidation(t *testing.T) {
	tests := []struct {
		name      string
		in        MovementInput
		wantField string
	}{
		{"unknown type", MovementInput{Type: "sideways", Quantity: 1}, "type"},
		{"zero quantity", MovementInput{Type: MovementIn, Quantity: 0}, "quantity"},
		{"negative quantity", MovementInput{Type: MovementOut, Quantity: -5}, "quantity"},
		{"quantity too large", MovementInput{Type: MovementIn, Quantity: MaxQuantity + 1}, "quantity"},
		{"note too long", MovementInput{Type: MovementIn, Quantity: 1, Note: strings.Repeat("a", 501)}, "note"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, _ := newTestService()

			_, err := svc.RecordMovement(context.Background(), "user-1", "product-1", tt.in)

			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("err = %v, want validation.Error", err)
			}
			if _, ok := validationErr.Fields[tt.wantField]; !ok {
				t.Errorf("fields = %v, want %q", validationErr.Fields, tt.wantField)
			}
			if len(store.movements) != 0 {
				t.Error("invalid movement recorded")
			}
		})
	}
}

func TestMovementDelta(t *testing.T) {
	if d := (Movement{Type: MovementIn, Quantity: 5}).delta(); d != 5 {
		t.Errorf("in delta = %d, want 5", d)
	}
	if d := (Movement{Type: MovementOut, Quantity: 5}).delta(); d != -5 {
		t.Errorf("out delta = %d, want -5", d)
	}
}

func TestListOrder(t *testing.T) {
	tests := []struct {
		name      string
		in        ListInput
		wantOrder string
		wantErr   bool
	}{
		{"default", ListInput{}, "name asc", false},
		{"price desc", ListInput{Sort: "price", Order: "desc"}, "price desc", false},
		{"quantity default order", ListInput{Sort: "quantity"}, "quantity asc", false},
		{"unknown sort", ListInput{Sort: "price; DROP TABLE products"}, "", true},
		{"unknown order", ListInput{Sort: "price", Order: "up"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, _ := newTestService()

			_, err := svc.List(context.Background(), tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && store.lastFilter.Order != tt.wantOrder {
				t.Errorf("order = %q, want %q", store.lastFilter.Order, tt.wantOrder)
			}

			if !tt.wantErr && orderClauses[store.lastFilter.Order] == "" {
				t.Errorf("order %q has no SQL clause", store.lastFilter.Order)
			}
		})
	}
}

func TestDeleteRequiresEmptyStock(t *testing.T) {
	svc, store, _, _ := newTestService()
	store.product = Product{ID: "product-1", StockQuantity: 3}
	ctx := context.Background()

	if err := svc.Delete(ctx, "product-1"); !errors.Is(err, ErrStockNotEmpty) {
		t.Fatalf("delete with stock: err = %v, want ErrStockNotEmpty", err)
	}

	if _, err := svc.RecordMovement(ctx, "user-1", "product-1", MovementInput{Type: MovementOut, Quantity: 3}); err != nil {
		t.Fatalf("emptying stock: %v", err)
	}

	if err := svc.Delete(ctx, "product-1"); err != nil {
		t.Errorf("delete with zero stock: %v", err)
	}
}
