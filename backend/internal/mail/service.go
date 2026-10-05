package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// redactedPlaceholder sostituisce i valori segreti nella copia del corpo
// salvata in email_logs.
const redactedPlaceholder = "[oscurato]"

type sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type logStore interface {
	CreateLog(ctx context.Context, e logEntry) error
}

// Il compilatore verifica che i tipi concreti soddisfino le interfacce.
var (
	_ sender   = (*SMTPSender)(nil)
	_ logStore = (*Repository)(nil)
)

type Service struct {
	sender sender
	store  logStore
	logger *slog.Logger

	// background conta gli invii in corso avviati da SendAsync.
	background sync.WaitGroup
}

func NewService(sender sender, store logStore, logger *slog.Logger) *Service {
	return &Service{sender: sender, store: store, logger: logger}
}

// Send invia l'email e ne registra l'esito in email_logs, con i valori
// segreti già sostituiti dal segnaposto. Restituisce l'errore dell'invio
// e/o della registrazione.
func (s *Service) Send(ctx context.Context, msg Message) error {
	sendErr := s.sender.Send(ctx, msg.To, msg.Subject, msg.Body)

	status := statusSent
	if sendErr != nil {
		status = statusFailed
		sendErr = fmt.Errorf("sending %s email: %w", msg.Type, sendErr)
	}

	logErr := s.store.CreateLog(ctx, logEntry{
		Recipient: msg.To,
		Type:      msg.Type,
		Subject:   msg.Subject,
		Body:      redact(msg.Body, msg.Secrets),
		Status:    status,
	})
	if logErr != nil {
		logErr = fmt.Errorf("logging %s email: %w", msg.Type, logErr)
	}

	return errors.Join(sendErr, logErr)
}

// SendAsync invia l'email in background e registra nei log un eventuale
// errore. Il context viene staccato dalla cancellazione della richiesta
// (che termina subito dopo la risposta); Wait attende gli invii in corso.
func (s *Service) SendAsync(ctx context.Context, msg Message) {
	ctx = context.WithoutCancel(ctx)

	s.background.Go(func() {
		if err := s.Send(ctx, msg); err != nil {
			s.logger.Error("sending email in background", "type", msg.Type, "error", err)
		}
	})
}

// Wait attende la fine degli invii in background, al più fino alla
// scadenza di ctx. Va chiamato all'arresto, dopo che il server HTTP ha
// smesso di accettare richieste.
func (s *Service) Wait(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		s.background.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for background emails: %w", ctx.Err())
	}
}

// redact sostituisce ogni valore segreto con il segnaposto.
func redact(body string, secrets []string) string {
	for _, secret := range secrets {
		// Con una stringa vuota ReplaceAll inserirebbe il segnaposto tra
		// ogni carattere.
		if secret == "" {
			continue
		}

		body = strings.ReplaceAll(body, secret, redactedPlaceholder)
	}

	return body
}
