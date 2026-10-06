package product

import (
	"fmt"

	"github.com/domtorelli/risma/backend/internal/mail"
	"github.com/domtorelli/risma/backend/internal/user"
)

// lowStockEmail compone l'avviso di scorta sotto la soglia minima per un
// Admin. Il nome del prodotto è già validato senza caratteri di controllo,
// quindi può stare nell'oggetto senza rischio di header injection.
func lowStockEmail(admin user.User, p Product) mail.Message {
	body := fmt.Sprintf(`Gentile %s %s,

la quantità disponibile del prodotto "%s" (%s) è sotto la soglia minima di magazzino.

Quantità disponibile: %d
Soglia minima: %d

Risma
`, admin.FirstName, admin.LastName, p.Name, p.TypeName, p.StockQuantity, p.MinStockThreshold)

	return mail.Message{
		Type:    mail.TypeLowStock,
		To:      admin.Email,
		Subject: "Risma - Scorta sotto la soglia minima: " + p.Name,
		Body:    body,
	}
}
