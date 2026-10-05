package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"
	"time"
)

// sendTimeout limita l'intero dialogo con il server SMTP.
const sendTimeout = 15 * time.Second

var errHeaderInjection = errors.New("header value contains a line break")

// SMTPSender invia email con il pacchetto net/smtp della libreria standard.
type SMTPSender struct {
	addr string
	host string
	from *netmail.Address
	auth smtp.Auth
}

// NewSMTPSender prepara l'invio verso host:port. Con username vuoto non si
// autentica (es. Mailpit in sviluppo). smtp.PlainAuth rifiuta di inviare le
// credenziali su una connessione non cifrata, tranne verso localhost.
func NewSMTPSender(host, port, username, password, from string) (*SMTPSender, error) {
	fromAddr, err := netmail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("parsing sender address: %w", err)
	}

	s := &SMTPSender{
		addr: net.JoinHostPort(host, port),
		host: host,
		from: fromAddr,
	}

	if username != "" {
		s.auth = smtp.PlainAuth("", username, password, host)
	}

	return s, nil
}

// Send invia un'email in solo testo a un destinatario. Usa STARTTLS se il
// server lo offre.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	toAddr, err := netmail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("parsing recipient address: %w", err)
	}

	msg, err := buildMessage(s.from, toAddr, subject, body, time.Now())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("connecting to smtp server: %w", err)
	}
	defer conn.Close()

	// net/smtp non usa il context: la scadenza sulla connessione limita anche
	// i comandi successivi alla connessione.
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("setting smtp deadline: %w", err)
		}
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("starting smtp session: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return fmt.Errorf("starting tls: %w", err)
		}
	}

	if s.auth != nil {
		if err := client.Auth(s.auth); err != nil {
			return fmt.Errorf("authenticating: %w", err)
		}
	}

	if err := client.Mail(s.from.Address); err != nil {
		return fmt.Errorf("setting sender: %w", err)
	}

	if err := client.Rcpt(toAddr.Address); err != nil {
		return fmt.Errorf("setting recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("starting message data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("closing smtp session: %w", err)
	}

	return nil
}

// buildMessage compone il messaggio RFC 5322 in UTF-8. Il corpo è codificato
// quoted-printable, l'oggetto con la codifica MIME degli header, così le
// lettere accentate arrivano intatte. Un a capo nell'oggetto permetterebbe
// di aggiungere header arbitrari (header injection), quindi viene rifiutato.
func buildMessage(from, to *netmail.Address, subject, body string, date time.Time) ([]byte, error) {
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errHeaderInjection
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "From: %s\r\n", from.String())
	fmt.Fprintf(&buf, "To: %s\r\n", to.String())
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", date.Format(time.RFC1123Z))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return nil, fmt.Errorf("encoding body: %w", err)
	}
	if err := qp.Close(); err != nil {
		return nil, fmt.Errorf("encoding body: %w", err)
	}

	return buf.Bytes(), nil
}
