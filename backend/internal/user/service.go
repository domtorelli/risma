package user

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/domtorelli/risma/backend/internal/role"
)

// maxPasswordBytes è il limite oltre il quale bcrypt rifiuta la password.
const maxPasswordBytes = 72

// Paginazione dell'elenco utenti.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var (
	ErrWeakPassword          = errors.New("password must be 8 to 72 bytes long and contain an uppercase letter, a digit and a special character")
	ErrForbidden             = errors.New("only an admin can assign or change roles")
	ErrDeactivationForbidden = errors.New("only an admin can deactivate users")
	ErrSelfDeactivation      = errors.New("users cannot deactivate their own account")
)

type userStore interface {
	FindByID(ctx context.Context, id string) (User, error)
	List(ctx context.Context, search string, deactivated bool, limit, offset int) ([]User, error)
	Count(ctx context.Context, search string, deactivated bool) (int, error)
	Create(ctx context.Context, u User) (User, error)
	Update(ctx context.Context, u User) (User, error)
	UpdateRole(ctx context.Context, id, roleID string) (User, error)
	SoftDelete(ctx context.Context, id string) error
	Reactivate(ctx context.Context, id string) (User, error)
}

type roleFinder interface {
	FindByName(ctx context.Context, name string) (role.Role, error)
}

// inviter invia a un nuovo utente l'email di benvenuto con il link per
// scegliere la password. È implementato da auth.Service: l'interfaccia sta
// qui così user non importa auth (che importa già user).
type inviter interface {
	InviteUser(ctx context.Context, u User) error
}

// Il compilatore verifica che i tipi concreti soddisfino le interfacce.
var (
	_ userStore  = (*Repository)(nil)
	_ roleFinder = (*role.Service)(nil)
)

type Service struct {
	store   userStore
	roles   roleFinder
	inviter inviter
	now     func() time.Time
}

// NewService crea il service degli utenti. inviter può essere nil solo se
// non si usa Create (es. il comando seed, che usa CreateWithPassword).
func NewService(store userStore, roles roleFinder, inviter inviter) *Service {
	return &Service{store: store, roles: roles, inviter: inviter, now: time.Now}
}

// CreateInput contiene i dati per creare un utente.
// RoleID vuoto assegna il ruolo Operatore; sceglierne uno è riservato all'Admin.
type CreateInput struct {
	RoleID      string
	Username    string
	Email       string
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

// ListInput contiene ricerca, filtro e paginazione; Page parte da 1.
// Deactivated true elenca gli utenti disattivati invece di quelli attivi.
type ListInput struct {
	Search      string
	Deactivated bool
	Page        int
	PageSize    int
}

// Page è una pagina dell'elenco utenti con il totale dei risultati.
type Page struct {
	Users    []User
	Total    int
	Page     int
	PageSize int
}

// Create crea un utente senza che nessuno ne scelga la password: l'utente
// riceve un'email di benvenuto con il link per impostarla. Nel frattempo la
// password è casuale e nessuno la conosce. actorRole è il ruolo di chi
// esegue l'operazione: solo un Admin può scegliere il ruolo (scheda, §4.3).
//
// Se l'invio dell'invito fallisce l'utente resta creato e l'errore viene
// restituito: l'utente può comunque usare il recupero password.
func (s *Service) Create(ctx context.Context, actorRole string, in CreateInput) (User, error) {
	hash, err := randomPasswordHash()
	if err != nil {
		return User{}, fmt.Errorf("creating user: %w", err)
	}

	created, err := s.create(ctx, actorRole, in, hash)
	if err != nil {
		return User{}, err
	}

	if err := s.inviter.InviteUser(ctx, created); err != nil {
		return User{}, fmt.Errorf("inviting created user: %w", err)
	}

	return created, nil
}

// CreateWithPassword crea un utente con la password indicata, senza invito.
// Serve solo al comando seed, che crea il primo Admin quando ancora nessuno
// può riceverne o gestirne gli inviti. Password è in chiaro e non deve mai
// essere salvata né registrata nei log.
func (s *Service) CreateWithPassword(ctx context.Context, actorRole string, in CreateInput, password string) (User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	return s.create(ctx, actorRole, in, hash)
}

// create applica le regole comuni alla creazione e salva l'utente.
func (s *Service) create(ctx context.Context, actorRole string, in CreateInput, passwordHash string) (User, error) {
	if in.RoleID != "" && actorRole != role.Admin {
		return User{}, ErrForbidden
	}

	p := profile{
		Username:    in.Username,
		Email:       in.Email,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		DateOfBirth: in.DateOfBirth,
	}.normalize()

	if err := p.validate(s.today()); err != nil {
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

	created, err := s.store.Create(ctx, User{
		RoleID:       roleID,
		Username:     p.Username,
		Email:        p.Email,
		PasswordHash: passwordHash,
		FirstName:    p.FirstName,
		LastName:     p.LastName,
		DateOfBirth:  p.DateOfBirth,
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

	total, err := s.store.Count(ctx, in.Search, in.Deactivated)
	if err != nil {
		return Page{}, fmt.Errorf("listing users: %w", err)
	}

	users, err := s.store.List(ctx, in.Search, in.Deactivated, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, fmt.Errorf("listing users: %w", err)
	}

	return Page{Users: users, Total: total, Page: page, PageSize: pageSize}, nil
}

// Update modifica i dati di un utente; non tocca ruolo né password.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (User, error) {
	p := profile(in).normalize()

	if err := p.validate(s.today()); err != nil {
		return User{}, err
	}

	updated, err := s.store.Update(ctx, User{
		ID:          id,
		Username:    p.Username,
		Email:       p.Email,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		DateOfBirth: p.DateOfBirth,
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

// Deactivate disattiva un utente (soft delete) e ne chiude le sessioni.
// actorID e actorRole sono l'utente che esegue l'operazione e il suo ruolo:
// disattivare è riservato all'Admin (come in ChangeRole, il controllo vale
// anche se la rotta è già protetta) e nessuno può disattivare sé stesso,
// per non chiudersi fuori per errore.
func (s *Service) Deactivate(ctx context.Context, actorID, actorRole, id string) error {
	if actorRole != role.Admin {
		return ErrDeactivationForbidden
	}

	if actorID == id {
		return ErrSelfDeactivation
	}

	if err := s.store.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("deactivating user: %w", err)
	}

	return nil
}

// Reactivate riattiva un utente disattivato.
func (s *Service) Reactivate(ctx context.Context, id string) (User, error) {
	u, err := s.store.Reactivate(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("reactivating user: %w", err)
	}

	return u, nil
}

// today restituisce la data odierna (UTC, ore 00:00), confrontabile con
// le date di nascita che arrivano senza orario.
func (s *Service) today() time.Time {
	return s.now().UTC().Truncate(24 * time.Hour)
}

// randomPasswordHash restituisce l'hash bcrypt di una password casuale che
// nessuno conosce: l'account non è utilizzabile finché l'utente non sceglie
// la sua password con il link ricevuto.
func randomPasswordHash() (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(rand.Text()), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing random password: %w", err)
	}

	return string(hash), nil
}

// HashPassword verifica che la password rispetti le regole AGID e ne
// calcola l'hash bcrypt. È l'unico punto da cui passa una nuova password,
// sia alla creazione dell'utente sia al recupero password.
func HashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	return string(hash), nil
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
