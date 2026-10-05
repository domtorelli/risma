package dashboard

import "time"

// Summary sono i dati della dashboard, letti tutti dalla stessa istantanea
// del database.
type Summary struct {
	TotalUsers    int // utenti non disattivati
	TotalProducts int // prodotti non eliminati
	// InventoryValue è la somma di giacenza × prezzo unitario dei prodotti
	// non eliminati, come stringa decimale (es. "1234.50").
	InventoryValue  string
	RecentMovements []Movement
	ProductsByType  []TypeCount
}

// Movement è un movimento di magazzino con i dati da mostrare.
type Movement struct {
	ID             string
	Type           string
	Quantity       int
	Note           *string
	CreatedAt      time.Time
	ProductID      string
	ProductName    string
	ProductDeleted bool
	Username       string
}

// TypeCount è il numero di prodotti non eliminati di un tipo.
type TypeCount struct {
	TypeID   string
	TypeName string
	Count    int
}
