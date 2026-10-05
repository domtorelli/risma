package mail

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	netmail "net/mail"
	"strings"
	"testing"
	"time"
)

type fakeSender struct {
	err  error
	sent []string // corpi inviati
}

func (f *fakeSender) Send(ctx context.Context, to, subject, body string) error {
	f.sent = append(f.sent, body)
	return f.err
}

type fakeLogStore struct {
	entries []logEntry
}

func (f *fakeLogStore) CreateLog(ctx context.Context, e logEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func newTestService(sendErr error) (*Service, *fakeSender, *fakeLogStore) {
	sender := &fakeSender{err: sendErr}
	store := &fakeLogStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewService(sender, store, logger), sender, store
}

func TestSendRedactsSecretsInLog(t *testing.T) {
	svc, sender, store := newTestService(nil)
	link := "https://risma.example.it/reset-password/SECRETTOKEN"

	err := svc.Send(context.Background(), Message{
		Type:    TypePasswordReset,
		To:      "mario@example.com",
		Subject: "Recupero password",
		Body:    "Apri il link: " + link + "\n",
		Secrets: []string{link, ""},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if !strings.Contains(sender.sent[0], link) {
		t.Error("sent email does not contain the link")
	}

	logged := store.entries[0]
	if strings.Contains(logged.Body, "SECRETTOKEN") {
		t.Errorf("secret written to email log: %q", logged.Body)
	}

	if logged.Body != "Apri il link: "+redactedPlaceholder+"\n" {
		t.Errorf("logged body = %q", logged.Body)
	}

	if logged.Status != statusSent {
		t.Errorf("status = %q, want %q", logged.Status, statusSent)
	}
}

func TestSendLogsFailure(t *testing.T) {
	smtpErr := errors.New("connection refused")
	svc, _, store := newTestService(smtpErr)

	err := svc.Send(context.Background(), Message{Type: TypeNewUser, To: "mario@example.com", Subject: "Benvenuto", Body: "Ciao"})
	if !errors.Is(err, smtpErr) {
		t.Fatalf("err = %v, want %v", err, smtpErr)
	}

	if len(store.entries) != 1 || store.entries[0].Status != statusFailed {
		t.Errorf("entries = %+v, want one failed entry", store.entries)
	}
}

func TestSendAsyncAndWait(t *testing.T) {
	svc, sender, store := newTestService(nil)

	svc.SendAsync(context.Background(), Message{Type: TypeNewUser, To: "mario@example.com", Subject: "Benvenuto", Body: "Ciao"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := svc.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if len(sender.sent) != 1 || len(store.entries) != 1 {
		t.Errorf("sent = %d, logged = %d, want 1 and 1", len(sender.sent), len(store.entries))
	}
}

func TestBuildMessage(t *testing.T) {
	from := &netmail.Address{Name: "Risma", Address: "noreply@risma.local"}
	to := &netmail.Address{Address: "mario@example.com"}

	msg, err := buildMessage(from, to, "Già fatto", "Città\n", time.Now())
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}

	parsed, err := netmail.ReadMessage(strings.NewReader(string(msg)))
	if err != nil {
		t.Fatalf("message is not valid RFC 5322: %v", err)
	}

	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil || subject != "Già fatto" {
		t.Errorf("subject = %q, err = %v", subject, err)
	}

	if _, err := buildMessage(from, to, "Oggetto\r\nBcc: attacker@example.com", "x", time.Now()); !errors.Is(err, errHeaderInjection) {
		t.Errorf("header injection: err = %v, want errHeaderInjection", err)
	}
}
