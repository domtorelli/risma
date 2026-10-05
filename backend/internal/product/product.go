package product

import "time"

// Valori del tipo ENUM stock_movement_type.
const (
	MovementIn  = "in"  // carico
	MovementOut = "out" // scarico
)

// Type è un tipo di prodotto (Buste, Carta, Toner).
type Type struct {
	ID          string
	Name        string
	Description *string
}

type Product struct {
	ID              string
	CreatorID       string
	CreatorUsername string // letto con una JOIN su users
	TypeID          string
	TypeName        string // letto con una JOIN su product_types
	Name            string
	Description     *string
	// Price è il prezzo unitario come stringa decimale (es. "12.50"), così
	// il valore NUMERIC non passa mai da un float.
	Price             string
	StockQuantity     int
	MinStockThreshold int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// BelowThreshold indica se la quantità disponibile è sotto la soglia minima.
func (p Product) BelowThreshold() bool {
	return p.StockQuantity < p.MinStockThreshold
}

// Movement è un carico o scarico di magazzino. Quantity è sempre positiva:
// il segno della variazione dipende dal tipo.
type Movement struct {
	ProductID string
	UserID    string
	Type      string
	Quantity  int
	Note      *string
}

// delta è la variazione della quantità disponibile prodotta dal movimento.
func (m Movement) delta() int {
	if m.Type == MovementOut {
		return -m.Quantity
	}

	return m.Quantity
}
