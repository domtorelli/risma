package dashboard

import (
	"context"
	"fmt"
)

type summaryStore interface {
	Summary(ctx context.Context) (Summary, error)
}

// Il compilatore verifica che *Repository soddisfi summaryStore.
var _ summaryStore = (*Repository)(nil)

// Service espone i dati della dashboard. Non ha regole proprie: esiste per
// mantenere la stessa separazione handler → service → repository degli
// altri domini, così l'handler non dipende dal database.
type Service struct {
	store summaryStore
}

func NewService(store summaryStore) *Service {
	return &Service{store: store}
}

// Summary restituisce totali, valore dell'inventario, ultimi movimenti e
// prodotti per tipo.
func (s *Service) Summary(ctx context.Context) (Summary, error) {
	summary, err := s.store.Summary(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("getting dashboard summary: %w", err)
	}

	return summary, nil
}
