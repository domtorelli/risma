// Il comando seed crea il primo utente Admin. La password si legge da stdin,
// così non compare nella cronologia della shell, tra i processi o
// nell'ambiente del server.
//
//	go run ./cmd/seed -username admin -email admin@example.com \
//		-first-name Mario -last-name Rossi -birth-date 1980-01-31
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/matteotorelli/risma/backend/internal/config"
	"github.com/matteotorelli/risma/backend/internal/database"
	"github.com/matteotorelli/risma/backend/internal/role"
	"github.com/matteotorelli/risma/backend/internal/user"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	if err := run(context.Background()); err != nil {
		logger.Error("seeding admin", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var in user.CreateInput
	var birthDate string

	flag.StringVar(&in.Username, "username", "", "username dell'admin")
	flag.StringVar(&in.Email, "email", "", "email dell'admin")
	flag.StringVar(&in.FirstName, "first-name", "", "nome")
	flag.StringVar(&in.LastName, "last-name", "", "cognome")
	flag.StringVar(&birthDate, "birth-date", "", "data di nascita (AAAA-MM-GG)")
	flag.Parse()

	if in.Username == "" || in.Email == "" || in.FirstName == "" || in.LastName == "" || birthDate == "" {
		flag.Usage()
		return errors.New("all flags are required")
	}

	var err error
	in.DateOfBirth, err = time.Parse(time.DateOnly, birthDate)
	if err != nil {
		return fmt.Errorf("parsing birth date: %w", err)
	}

	in.Password, err = readPassword()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	roles := role.NewService(role.NewRepository(db))
	users := user.NewService(user.NewRepository(db), roles)

	admin, err := roles.FindByName(ctx, role.Admin)
	if err != nil {
		return err
	}
	in.RoleID = admin.ID

	// Il comando è eseguito da chi amministra il server: agisce come Admin.
	created, err := users.Create(ctx, role.Admin, in)
	if err != nil {
		return err
	}

	fmt.Printf("admin %q created (id %s)\n", created.Username, created.ID)

	return nil
}

// readPassword legge la password dalla prima riga di stdin.
func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}

		return "", errors.New("reading password: no input")
	}

	return strings.TrimRight(scanner.Text(), "\r"), nil
}
