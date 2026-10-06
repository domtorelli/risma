package user

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/domtorelli/risma/backend/internal/role"
	"github.com/domtorelli/risma/backend/internal/validation"
)

const operatorRoleID = "operator-role-id"

// fakeStore registra le chiamate al posto del database.
type fakeStore struct {
	created     User
	createCalls int

	updated     User
	updateCalls int

	updateRoleCalls int
	softDeleteCalls int

	listDeactivated       bool
	listLimit, listOffset int
	total                 int
}

func (f *fakeStore) FindByID(ctx context.Context, id string) (User, error) {
	return User{}, ErrNotFound
}

func (f *fakeStore) List(ctx context.Context, search string, deactivated bool, limit, offset int) ([]User, error) {
	f.listDeactivated, f.listLimit, f.listOffset = deactivated, limit, offset
	return nil, nil
}

func (f *fakeStore) Count(ctx context.Context, search string, deactivated bool) (int, error) {
	return f.total, nil
}

func (f *fakeStore) Create(ctx context.Context, u User) (User, error) {
	f.createCalls++
	u.ID = "new-user-id"
	f.created = u
	return u, nil
}

func (f *fakeStore) Update(ctx context.Context, u User) (User, error) {
	f.updateCalls++
	f.updated = u
	return u, nil
}

func (f *fakeStore) UpdateRole(ctx context.Context, id, roleID string) (User, error) {
	f.updateRoleCalls++
	return User{ID: id, RoleID: roleID}, nil
}

func (f *fakeStore) SoftDelete(ctx context.Context, id string) error {
	f.softDeleteCalls++
	return nil
}

func (f *fakeStore) Reactivate(ctx context.Context, id string) (User, error) {
	return User{ID: id}, nil
}

type fakeRoles struct{}

func (fakeRoles) FindByName(ctx context.Context, name string) (role.Role, error) {
	if name == role.Operator {
		return role.Role{ID: operatorRoleID, Name: role.Operator}, nil
	}

	return role.Role{}, role.ErrNotFound
}

type fakeInviter struct {
	invited []User
	err     error
}

func (f *fakeInviter) InviteUser(ctx context.Context, u User) error {
	f.invited = append(f.invited, u)
	return f.err
}

// today è la data fissa usata come "oggi" nei test.
var today = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

func newTestService() (*Service, *fakeStore, *fakeInviter) {
	store := &fakeStore{}
	inviter := &fakeInviter{}

	svc := NewService(store, fakeRoles{}, inviter)
	svc.now = func() time.Time { return today.Add(10 * time.Hour) }

	return svc, store, inviter
}

func validInput() CreateInput {
	return CreateInput{
		Username:    "mrossi",
		Email:       "mario.rossi@example.com",
		FirstName:   "Mario",
		LastName:    "Rossi",
		DateOfBirth: time.Date(1980, 1, 31, 0, 0, 0, 0, time.UTC),
	}
}

func TestCreateRoleAssignment(t *testing.T) {
	tests := []struct {
		name       string
		actorRole  string
		roleID     string
		wantErr    error
		wantRoleID string
	}{
		{"admin chooses role", role.Admin, "admin-role-id", nil, "admin-role-id"},
		{"admin without role gets operator", role.Admin, "", nil, operatorRoleID},
		{"operator without role gets operator", role.Operator, "", nil, operatorRoleID},
		{"operator cannot choose role", role.Operator, "admin-role-id", ErrForbidden, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, inviter := newTestService()

			in := validInput()
			in.RoleID = tt.roleID

			_, err := svc.Create(context.Background(), tt.actorRole, in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if store.createCalls != 0 || len(inviter.invited) != 0 {
					t.Fatal("user saved or invited despite error")
				}
				return
			}

			if store.created.RoleID != tt.wantRoleID {
				t.Errorf("RoleID = %q, want %q", store.created.RoleID, tt.wantRoleID)
			}
		})
	}
}

func TestCreateInvitesUserWithUnknownPassword(t *testing.T) {
	svc, store, inviter := newTestService()

	created, err := svc.Create(context.Background(), role.Admin, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(inviter.invited) != 1 || inviter.invited[0].ID != created.ID {
		t.Fatalf("invited = %+v, want the created user", inviter.invited)
	}

	if !strings.HasPrefix(store.created.PasswordHash, "$2") {
		t.Errorf("password hash %q is not a bcrypt hash", store.created.PasswordHash)
	}
}

func TestCreateReportsInviteFailure(t *testing.T) {
	svc, _, inviter := newTestService()
	inviter.err = errors.New("smtp down")

	if _, err := svc.Create(context.Background(), role.Admin, validInput()); !errors.Is(err, inviter.err) {
		t.Fatalf("err = %v, want invite error", err)
	}
}

func TestCreateWithPassword(t *testing.T) {
	svc, store, inviter := newTestService()

	if _, err := svc.CreateWithPassword(context.Background(), role.Admin, validInput(), "Password1!"); err != nil {
		t.Fatalf("CreateWithPassword: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(store.created.PasswordHash), []byte("Password1!")); err != nil {
		t.Errorf("hash does not match password: %v", err)
	}

	if len(inviter.invited) != 0 {
		t.Error("user with password must not be invited")
	}

	if _, err := svc.CreateWithPassword(context.Background(), role.Admin, validInput(), "password"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak password: err = %v, want ErrWeakPassword", err)
	}
}

func TestCreateNormalizesInput(t *testing.T) {
	svc, store, _ := newTestService()

	in := validInput()
	in.Username = "  mrossi "
	in.FirstName = " Mario"

	if _, err := svc.Create(context.Background(), role.Admin, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if store.created.Username != "mrossi" || store.created.FirstName != "Mario" {
		t.Errorf("fields not trimmed: %+v", store.created)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		edit      func(*CreateInput)
		wantField string // vuoto: input valido
	}{
		{"valid", func(in *CreateInput) {}, ""},
		{"username too short", func(in *CreateInput) { in.Username = "ab" }, "username"},
		{"username with space", func(in *CreateInput) { in.Username = "mario rossi" }, "username"},
		{"username with symbols", func(in *CreateInput) { in.Username = "<script>" }, "username"},
		{"username with dot and dash", func(in *CreateInput) { in.Username = "mario.rossi-1_a" }, ""},
		{"email missing at", func(in *CreateInput) { in.Email = "mario.example.com" }, "email"},
		{"email with display name", func(in *CreateInput) { in.Email = "Mario <mario@example.com>" }, "email"},
		{"email too long", func(in *CreateInput) { in.Email = strings.Repeat("a", 250) + "@example.com" }, "email"},
		{"first name empty", func(in *CreateInput) { in.FirstName = "   " }, "first_name"},
		{"last name too long", func(in *CreateInput) { in.LastName = strings.Repeat("à", 101) }, "last_name"},
		{"last name with apostrophe", func(in *CreateInput) { in.LastName = "D'Annunzio" }, ""},
		{"name with control char", func(in *CreateInput) { in.FirstName = "Mario\n" + "Rossi" }, "first_name"},
		{"birth date today", func(in *CreateInput) { in.DateOfBirth = today }, "date_of_birth"},
		{"birth date yesterday", func(in *CreateInput) { in.DateOfBirth = today.AddDate(0, 0, -1) }, ""},
		{"birth date before 1900", func(in *CreateInput) { in.DateOfBirth = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC) }, "date_of_birth"},
		{"birth date missing", func(in *CreateInput) { in.DateOfBirth = time.Time{} }, "date_of_birth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _ := newTestService()

			in := validInput()
			tt.edit(&in)

			_, err := svc.Create(context.Background(), role.Admin, in)

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
				t.Error("invalid user saved")
			}
		})
	}
}

func TestUpdateValidates(t *testing.T) {
	svc, store, _ := newTestService()

	_, err := svc.Update(context.Background(), "user-id", UpdateInput{Username: "x", Email: "mario@example.com", FirstName: "Mario", LastName: "Rossi"})

	var validationErr *validation.Error
	if !errors.As(err, &validationErr) || store.updateCalls != 0 {
		t.Fatalf("err = %v, updateCalls = %d", err, store.updateCalls)
	}
}

func TestChangeRole(t *testing.T) {
	tests := []struct {
		name      string
		actorRole string
		wantErr   error
		wantCalls int
	}{
		{"admin", role.Admin, nil, 1},
		{"operator", role.Operator, ErrForbidden, 0},
		{"unknown role", "", ErrForbidden, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _ := newTestService()

			_, err := svc.ChangeRole(context.Background(), tt.actorRole, "user-id", "role-id")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if store.updateRoleCalls != tt.wantCalls {
				t.Errorf("UpdateRole calls = %d, want %d", store.updateRoleCalls, tt.wantCalls)
			}
		})
	}
}

func TestDeactivate(t *testing.T) {
	tests := []struct {
		name      string
		actorRole string
		targetID  string
		wantErr   error
		wantCalls int
	}{
		{"admin", role.Admin, "someone-else", nil, 1},
		{"admin on self", role.Admin, "me", ErrSelfDeactivation, 0},
		{"operator", role.Operator, "someone-else", ErrDeactivationForbidden, 0},
		{"operator on self", role.Operator, "me", ErrDeactivationForbidden, 0},
		{"unknown role", "", "someone-else", ErrDeactivationForbidden, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _ := newTestService()

			err := svc.Deactivate(context.Background(), "me", tt.actorRole, tt.targetID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if store.softDeleteCalls != tt.wantCalls {
				t.Errorf("SoftDelete calls = %d, want %d", store.softDeleteCalls, tt.wantCalls)
			}
		})
	}
}

func TestListPagination(t *testing.T) {
	tests := []struct {
		name         string
		in           ListInput
		wantLimit    int
		wantOffset   int
		wantPage     int
		wantPageSize int
	}{
		{"defaults", ListInput{}, DefaultPageSize, 0, 1, DefaultPageSize},
		{"third page", ListInput{Page: 3, PageSize: 10}, 10, 20, 3, 10},
		{"negative page", ListInput{Page: -2, PageSize: 10}, 10, 0, 1, 10},
		{"page size capped", ListInput{Page: 1, PageSize: 1000}, MaxPageSize, 0, 1, MaxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _ := newTestService()
			store.total = 42

			page, err := svc.List(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			if store.listLimit != tt.wantLimit || store.listOffset != tt.wantOffset {
				t.Errorf("limit, offset = %d, %d, want %d, %d", store.listLimit, store.listOffset, tt.wantLimit, tt.wantOffset)
			}

			if page.Page != tt.wantPage || page.PageSize != tt.wantPageSize || page.Total != 42 {
				t.Errorf("page = %+v", page)
			}
		})
	}
}

func TestListDeactivated(t *testing.T) {
	svc, store, _ := newTestService()

	if _, err := svc.List(context.Background(), ListInput{Deactivated: true}); err != nil {
		t.Fatalf("List: %v", err)
	}

	if !store.listDeactivated {
		t.Error("deactivated filter not passed to the store")
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"Password1!", true},
		{"Àbcdefg1#", true},                       // maiuscola accentata
		{"Pass1!", false},                         // troppo corta
		{"password1!", false},                     // manca la maiuscola
		{"Password!!", false},                     // manca il numero
		{"Password12", false},                     // manca il carattere speciale
		{"Password 12", false},                    // lo spazio non è un carattere speciale
		{"Aa1!" + strings.Repeat("a", 69), false}, // oltre 72 byte
	}

	for _, tt := range tests {
		err := validatePassword(tt.password)
		if (err == nil) != tt.valid {
			t.Errorf("validatePassword(%q) = %v, want valid=%v", tt.password, err, tt.valid)
		}
	}
}
