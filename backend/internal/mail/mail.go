package mail

// Valori del tipo ENUM email_type.
const (
	TypeNewUser       = "new_user"
	TypeLowStock      = "low_stock"
	TypePasswordReset = "password_reset"
)

// Valori del tipo ENUM email_status.
const (
	statusSent   = "sent"
	statusFailed = "failed"
)

// Message è un'email in solo testo.
type Message struct {
	Type    string
	To      string
	Subject string
	Body    string
	// Secrets sono i valori del corpo (es. il link di reset) da sostituire con
	// un segnaposto nella copia salvata in email_logs: l'email inviata li
	// contiene, il registro no.
	Secrets []string
}

// logEntry è una riga di email_logs.
type logEntry struct {
	Recipient string
	Type      string
	Subject   string
	Body      string
	Status    string
}
