package user

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/matteotorelli/risma/backend/internal/role"
)

// maxPasswordBytes è il limite oltre il quale bcrypt rifiuta la password.
const maxPasswordBytes = 72

// Paginazione dell'elenco utenti.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var (
	ErrWeakPassword = errors.New("password must be 8 to 72 bytes long and contain an uppercase letter, a digit and a special character")
	ErrForbidden    = errors.New("only an admin can assign or change roles")
)

type userStore interface {
	FindByID(ctx context.Context, id string) (User, error)
	List(ctx context.Context, search string, limit, offset int) ([]User, error)
	Count(ctx context.Context, search string) (int, error)
	Create(ctx context.Context, u User) (User, error)
	Update(ctx context.Context, u User) (User, error)
	UpdateRole(ctx context.Context, id, roleID string) (User, error)
	SoftDelete(ctx context.Context, id string) error
}

type roleFinder interface {
	FindByName(ctx context.Context, name string) (role.Role, error)
}

// Il compilatore verifica che i tipi concreti soddisfino le interfacce.
var (
	_ userStore  = (*Repository)(nil)
	_ roleFinder = (*role.Service)(nil)
)

type Service struct {
	store userStore
	roles roleFinder
}

func NewService(store userStore, roles roleFinder) *Service {
	return &Service{store: store, roles: roles}
}

// CreateInput contiene i dati per creare un utente; Password è in chiaro
// e non deve mai essere salvata né registrata nei log.
// RoleID vuoto assegna il ruolo Operatore; sceglierne uno è riservato all'Admin.
type CreateInput struct {
	RoleID      string
	Username    string
	Email       string
	Password    string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
}

// UpdateInput contiene i dati modificabili di un utente. Il ruolo si cambia
// con ChangeRole, la password con il recupero password.
type UpdateInput struct {
	Username    string
	Email       string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
}

// ListInput contiene ricerca e paginazione; Page parte da 1.
type ListInput struct {
	Search   string
	Page     int
	PageSize int
}

// Page è una pagina dell'elenco utenti con il totale dei risultati.
type Page struct {
	Users    []User
	Total    int
	Page     int
	PageSize int
}

// Create valida la password, ne calcola l'hash con bcrypt e salva l'utente.
// actorRole è il ruolo di chi esegue l'operazione: solo un Admin può
// scegliere il ruolo del nuovo utente (scheda, §4.3).
func (s *Service) Create(ctx context.Context, actorRole string, in CreateInput) (User, error) {
	if in.RoleID != "" && actorRole != role.Admin {
		return User{}, ErrForbidden
	}

	if err := validatePassword(in.Password); err != nil {
		return User{}, err
	}

	roleID := in.RoleID
	if roleID == "" {
		operator, err := s.roles.FindByName(ctx, role.Operator)
		if err != nil {
			return User{}, fmt.Errorf("finding default role: %w", err)
		}

		roleID = operator.ID
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hashing password: %w", err)
	}

	created, err := s.store.Create(ctx, User{
		RoleID:       roleID,
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: string(hash),
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		DateOfBirth:  in.DateOfBirth,
	})
	if err != nil {
		return User{}, fmt.Errorf("creating user: %w", err)
	}

	return created, nil
}

// Get restituisce un utente non disattivato.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("getting user: %w", err)
	}

	return u, nil
}

// List restituisce una pagina di utenti filtrata per username o email.
// Valori di pagina non validi vengono ricondotti a quelli predefiniti.
func (s *Service) List(ctx context.Context, in ListInput) (Page, error) {
	page := max(in.Page, 1)

	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	pageSize = min(pageSize, MaxPageSize)

	total, err := s.store.Count(ctx, in.Search)
	if err != nil {
		return Page{}, fmt.Errorf("listing users: %w", err)
	}

	users, err := s.store.List(ctx, in.Search, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, fmt.Errorf("listing users: %w", err)
	}

	return Page{Users: users, Total: total, Page: page, PageSize: pageSize}, nil
}

// Update modifica i dati di un utente; non tocca ruolo né password.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (User, error) {
	updated, err := s.store.Update(ctx, User{
		ID:          id,
		Username:    in.Username,
		Email:       in.Email,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		DateOfBirth: in.DateOfBirth,
	})
	if err != nil {
		return User{}, fmt.Errorf("updating user: %w", err)
	}

	return updated, nil
}

// ChangeRole assegna un nuovo ruolo a un utente; è riservato all'Admin
// (scheda, §4.3). Il controllo vale anche se la rotta è già protetta,
// così la regola non dipende da come il service viene chiamato.
func (s *Service) ChangeRole(ctx context.Context, actorRole, id, roleID string) (User, error) {
	if actorRole != role.Admin {
		return User{}, ErrForbidden
	}

	updated, err := s.store.UpdateRole(ctx, id, roleID)
	if err != nil {
		return User{}, fmt.Errorf("changing user role: %w", err)
	}

	return updated, nil
}

// Deactivate disattiva un utente (soft delete).
func (s *Service) Deactivate(ctx context.Context, id string) error {
	if err := s.store.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("deactivating user: %w", err)
	}

	return nil
}

// validatePassword applica le regole AGID: minimo 8 caratteri, almeno una
// maiuscola, un numero e un carattere speciale (né lettera, né cifra, né spazio).
func validatePassword(password string) error {
	if len([]rune(password)) < 8 || len(password) > maxPasswordBytes {
		return ErrWeakPassword
	}

	var upper, digit, special bool

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsLetter(r), unicode.IsSpace(r):
			// lettera minuscola o spazio: non conta per nessuna regola
		default:
			special = true
		}
	}

	if !upper || !digit || !special {
		return ErrWeakPassword
	}

	return nil
}
