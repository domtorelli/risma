package auth

import (
	"fmt"

	"github.com/matteotorelli/risma/backend/internal/mail"
	"github.com/matteotorelli/risma/backend/internal/user"
)

// passwordResetEmail compone l'email di recupero password. Il link compare
// in Secrets, quindi nella copia salvata in email_logs viene oscurato.
func passwordResetEmail(u user.User, link string) mail.Message {
	body := fmt.Sprintf(`Gentile %s %s,

abbiamo ricevuto una richiesta di recupero della password per il tuo account Risma (%s).

Per scegliere una nuova password apri questo link entro 1 ora:

%s

Il link si può usare una sola volta. Se non hai chiesto tu il recupero, ignora questa email: la password resta invariata.

Risma
`, u.FirstName, u.LastName, u.Username, link)

	return mail.Message{
		Type:    mail.TypePasswordReset,
		To:      u.Email,
		Subject: "Risma - Recupero password",
		Body:    body,
		Secrets: []string{link},
	}
}
