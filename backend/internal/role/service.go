package role

import (
	"context"
	"fmt"
)

// I ruoli sono fissi (inseriti da migrazione): il service espone solo la
// lettura. La scheda riserva all'Admin la modifica dei ruoli degli utenti,
// che sta nel pacchetto user.
type roleStore interface {
	FindByName(ctx context.Context, name string) (Role, error)
	List(ctx context.Context) ([]Role, error)
}

// Il compilatore verifica che *Repository soddisfi roleStore.
var _ roleStore = (*Repository)(nil)

type Service struct {
	store roleStore
}

func NewService(store roleStore) *Service {
	return &Service{store: store}
}

// FindByName restituisce il ruolo con il nome indicato.
func (s *Service) FindByName(ctx context.Context, name string) (Role, error) {
	r, err := s.store.FindByName(ctx, name)
	if err != nil {
		return Role{}, fmt.Errorf("finding role %q: %w", name, err)
	}

	return r, nil
}

// List restituisce tutti i ruoli.
func (s *Service) List(ctx context.Context) ([]Role, error) {
	roles, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}

	return roles, nil
}
