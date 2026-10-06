# CLAUDE.md — Risma

Convenzioni del progetto. Il file fa da guida per chi scrive il codice e da
istruzioni per l'assistente AI usato durante lo sviluppo.

## Contesto

Risma è un gestionale web (utenti, ruoli, inventario di un negozio di cancelleria: Buste, Carta, Toner).

Stack richiesto dalla scheda dei requisiti: frontend Next.js (App Router) in JavaScript, backend Go con API REST, PostgreSQL con migrazioni, stile Bootstrap Italia, autenticazione JWT (access + refresh token).

Le decisioni progettuali già prese, con le motivazioni, sono in `docs/decisions.md`. Leggilo prima di proporre scelte architetturali.

---

## Modo di lavorare

- **Segui i pattern esistenti** nel repository (struttura, nomi, gestione degli errori) e indica quale file hai usato come modello.
- **Non introdurre librerie, pattern o strutture nuove** senza chiedere.
- Modifiche piccole e coese. Prima di dichiarare concluso un task: `make check` nel backend (gofmt, vet, test) e `npm run lint` nel frontend.

### Decisioni progettuali

Quando emerge una scelta di progettazione: presenta le opzioni con pro e contro, dai una raccomandazione motivata, **lascia decidere Matteo**. Poi aggiungi la decisione in `docs/decisions.md` nella sezione giusta (scelte progettuali, requisiti interpretati, limiti noti).

### Git

Non fare commit da solo. A fine task proponi un messaggio in formato Conventional Commits (es. `feat(backend): repository utenti`). I commit devono essere piccoli e raccontare un lavoro ragionato.

---

## Struttura del repository

```
risma/
├── backend/
│   ├── cmd/api/main.go          # punto d'ingresso: configurazione, DI, server, shutdown
│   ├── cmd/seed/                # creazione del primo amministratore
│   ├── internal/
│   │   ├── config/              # lettura variabili d'ambiente
│   │   ├── database/            # connessione pgxpool
│   │   ├── httpx/               # risposte JSON, middleware, limite di richieste
│   │   └── <dominio>/           # auth, user, role, product, dashboard, mail
│   │       ├── handler.go
│   │       ├── service.go
│   │       └── repository.go
│   ├── migrations/              # goose, numerazione progressiva
│   └── Makefile
├── frontend/                    # Next.js (App Router) + Bootstrap Italia, test e2e Playwright
├── docs/decisions.md
├── nginx/                       # reverse proxy HTTPS e script del certificato
├── docker-compose.yml           # db, mailpit, certs, migrate, backend, frontend, nginx (porte solo su 127.0.0.1)
├── .env.example
└── README.md
```

Comandi (dalla cartella `backend`):

- `make run` — avvia il backend caricando `../.env`
- `make check` — `gofmt`, `go vet` e test
- `make seed ARGS="…"` — crea un amministratore
- `make migrate-up` / `make migrate-down` / `make migrate-status`
- `go build ./...`, `go vet ./...`, `go mod tidy`
- Dalla radice: `docker compose up -d db mailpit` (solo i servizi per lo sviluppo) oppure `docker compose up -d --build` (stack completo su https://localhost)

Comandi (dalla cartella `frontend`): `npm run dev`, `npm run lint`, `npm run test:e2e`.

---

## Convenzioni Go

- **Livelli**: handler (HTTP, validazione input, JSON) → service (logica di business) → repository (SQL). Nel service non compaiono mai `net/http` né SQL. Ogni livello parla solo con quello sotto.
- **Dependency injection esplicita**: costruttori `NewXxx(...)`, tutto viene assemblato in `main.go`. Niente stato globale.
- **Interfacce definite da chi le usa** (il service definisce l'interfaccia del repository che gli serve), piccole.
- **Errori**:
  - messaggi in minuscolo, senza punto finale, senza "failed to"/"error": descrivono l'azione (`creating connection pool: %w`);
  - chi genera l'errore descrive solo il proprio problema, ogni livello sopra aggiunge il contesto;
  - `%w` dentro le funzioni, `%v` solo dove si stampa;
  - confronti con `errors.Is` / `errors.As`, mai `==`.
- **`context.Context`** primo parametro (`ctx`) di ogni funzione che fa I/O.
- **`defer` subito dopo il controllo dell'errore**, nel percorso in cui la risorsa è valida.
- **`log.Fatal` solo all'avvio**; mai durante il servizio delle richieste, perché salta i `defer`.
- **Nomi**: camelCase, sigle tutte maiuscole (`HTTPPort`, `userID`), niente nomi che nascondono pacchetti (`cfg`, `db`, `pool`, non `config`, `database`). Evita le ripetizioni col nome del pacchetto (`config.Load`, non `config.LoadConfig`).
- **Import** in tre gruppi separati da riga vuota: libreria standard, librerie esterne, pacchetti del modulo.
- **Libreria standard prima di tutto**; chiedi prima di aggiungere una dipendenza. Dopo ogni modifica alle dipendenze: `go mod tidy`.
- Logging strutturato con `log/slog` in JSON; gli errori 500 registrano la causa nei log e rispondono al client con un messaggio generico.
- Codice formattato con `gofmt`.

## Convenzioni database e migrazioni

- goose con numerazione progressiva (`-s`); un file per modifica; `-- +goose Up` / `-- +goose Down`.
- **Ogni `CREATE` nella parte Up ha il suo `DROP` nella parte Down, in ordine inverso** (tipi enum e funzioni compresi).
- `-- +goose StatementBegin/End` **solo** per istruzioni con punti e virgola interni (funzioni PL/pgSQL).
- Mai modificare una migrazione già condivisa (pushata): se ne scrive una nuova.
- Verificare sempre il giro `up` → `down` fino a zero → `up`.
- **Colonne standard** (copiarle, non riscriverle a memoria):
  - `id UUID PRIMARY KEY DEFAULT uuidv7()`
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` + trigger `<tabella>_set_updated_at` che chiama `set_updated_at()`
  - `deleted_at TIMESTAMPTZ` dove serve il soft delete
- Sempre `TIMESTAMPTZ`, sempre `now()` (non `CURRENT_TIMESTAMP`), prezzi in `NUMERIC(10, 2)`.
- Chiavi esterne: suffisso `_id`, `ON DELETE` esplicito, **indice esplicito** (PostgreSQL non lo crea).
- Nomi indici: `<tabella>_<colonna>_idx`, `<tabella>_<colonna>_lower_idx` per gli indici su `lower()`.
- Valori ENUM in minuscolo. ENUM per valori stabili, tabelle per valori gestiti dagli utenti.
- Vincoli nel database (`NOT NULL`, `CHECK`, FK): il database è l'ultima linea di difesa.
- Registri (`stock_movements`, `access_logs`, `email_logs`): nessun `updated_at`/`deleted_at`.

## Regole di sicurezza (non negoziabili)

- Query **sempre parametrizzate** (`$1`, `$2`…), mai concatenazione di stringhe.
- Password con bcrypt; regole AGID (min. 8 caratteri, 1 maiuscola, 1 numero, 1 carattere speciale) validate **sia nel frontend sia nel backend**.
- Blocco account dopo 5 tentativi falliti; ogni tentativo registrato in `access_logs`.
- Token di reset e refresh token salvati come **hash SHA-256**, mai in chiaro; token di reset valido 1 ora e invalidato dopo l'uso.
- Nel corpo salvato in `email_logs` il link di reset va sostituito da un segnaposto **prima** della scrittura.
- Segreti solo da variabili d'ambiente; mai password, token o hash nei log.
- Tutte le rotte protette da JWT, permessi con RBAC; solo l'Admin gestisce i ruoli.
- Protezione CSRF, header di sicurezza, HTTPS obbligatorio (reverse proxy).

---

## Decisioni di architettura confermate

Il dettaglio e le motivazioni sono in `docs/decisions.md`.

1. **Organizzazione per dominio** (`internal/user`, `internal/product`, `internal/auth`…), un file per livello dentro ogni pacchetto.
2. **Query scritte con pgx direttamente**, senza sqlc (citato come possibile evoluzione).
3. **Token nel frontend in cookie `httpOnly`, `Secure`, `SameSite=Strict`**, con frontend e API sotto lo stesso dominio tramite reverse proxy (`/api` → backend) e protezione CSRF esplicita con token double-submit.
