package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/matteotorelli/risma/backend/internal/role"
)

const operatorRoleID = "operator-role-id"

// fakeStore registra le chiamate al posto del database.
type fakeStore struct {
	created     User
	createCalls int

	updateRoleCalls int

	listLimit, listOffset int
	total                 int
}

func (f *fakeStore) FindByID(ctx context.Context, id string) (User, error) {
	return User{}, ErrNotFound
}

func (f *fakeStore) List(ctx context.Context, search string, limit, offset int) ([]User, error) {
	f.listLimit, f.listOffset = limit, offset
	return nil, nil
}

func (f *fakeStore) Count(ctx context.Context, search string) (int, error) {
	return f.total, nil
}

func (f *fakeStore) Create(ctx context.Context, u User) (User, error) {
	f.createCalls++
	f.created = u
	return u, nil
}

func (f *fakeStore) Update(ctx context.Context, u User) (User, error) {
	return u, nil
}

func (f *fakeStore) UpdateRole(ctx context.Context, id, roleID string) (User, error) {
	f.updateRoleCalls++
	return User{ID: id, RoleID: roleID}, nil
}

func (f *fakeStore) SoftDelete(ctx context.Context, id string) error {
	return nil
}

type fakeRoles struct{}

func (fakeRoles) FindByName(ctx context.Context, name string) (role.Role, error) {
	if name == role.Operator {
		return role.Role{ID: operatorRoleID, Name: role.Operator}, nil
	}

	return role.Role{}, role.ErrNotFound
}

func validInput() CreateInput {
	return CreateInput{
		Username:  "mrossi",
		Email:     "mario.rossi@example.com",
		Password:  "Password1!",
		FirstName: "Mario",
		LastName:  "Rossi",
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
			store := &fakeStore{}
			svc := NewService(store, fakeRoles{})

			in := validInput()
			in.RoleID = tt.roleID

			_, err := svc.Create(context.Background(), tt.actorRole, in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if store.createCalls != 0 {
					t.Fatal("user saved despite error")
				}
				return
			}

			if store.created.RoleID != tt.wantRoleID {
				t.Errorf("RoleID = %q, want %q", store.created.RoleID, tt.wantRoleID)
			}
		})
	}
}

func TestCreateHashesPassword(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, fakeRoles{})
	in := validInput()

	if _, err := svc.Create(context.Background(), role.Admin, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if store.created.PasswordHash == in.Password {
		t.Fatal("password saved in clear text")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(store.created.PasswordHash), []byte(in.Password)); err != nil {
		t.Errorf("hash does not match password: %v", err)
	}
}

func TestCreateRejectsWeakPassword(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, fakeRoles{})
	in := validInput()
	in.Password = "password"

	_, err := svc.Create(context.Background(), role.Admin, in)
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("err = %v, want ErrWeakPassword", err)
	}

	if store.createCalls != 0 {
		t.Fatal("user saved despite weak password")
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
			store := &fakeStore{}
			svc := NewService(store, fakeRoles{})

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
			store := &fakeStore{total: 42}
			svc := NewService(store, fakeRoles{})

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
