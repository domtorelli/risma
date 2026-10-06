# Risma

Gestionale web per un negozio di cancelleria: accesso con login, gestione di
utenti e ruoli, inventario dei prodotti (Buste, Carta, Toner) con movimenti di
magazzino e dashboard.

- **Backend**: Go, API REST, PostgreSQL con migrazioni
- **Frontend**: Next.js (App Router, JavaScript) con Bootstrap Italia
- **Autenticazione**: JWT (access token + refresh token) in cookie `HttpOnly`
- **Infrastruttura**: Docker Compose, nginx come reverse proxy HTTPS

Le scelte progettuali, con le motivazioni e i limiti noti, sono in
[`docs/decisions.md`](docs/decisions.md).

---

## Indice

1. [Avvio rapido con Docker](#avvio-rapido-con-docker)
2. [Sviluppo locale](#sviluppo-locale)
3. [Test](#test)
4. [Requisiti della scheda e dove sono implementati](#requisiti-della-scheda-e-dove-sono-implementati)
5. [Architettura](#architettura)
6. [Database](#database)
7. [Sicurezza](#sicurezza)
8. [API](#api)
9. [Struttura del repository](#struttura-del-repository)
10. [Limiti noti](#limiti-noti)

---

## Avvio rapido con Docker

Requisiti: Docker con Docker Compose v2.

```bash
git clone <url-del-repository> risma && cd risma

# 1. Configurazione: copia l'esempio e completa i due segreti
cp .env.example .env
#    DB_PASSWORD=<una password a scelta>
#    JWT_SECRET=<almeno 32 byte casuali, es. openssl rand -base64 48>

# 2. Avvio: certificato HTTPS, migrazioni, backend, frontend e nginx
docker compose up -d --build

# 3. Primo amministratore (la password viene chiesta nel terminale)
docker compose run --rm backend seed \
  -username admin -email admin@example.com \
  -first-name Mario -last-name Rossi -birth-date 1980-01-31
```

| Indirizzo | Cosa c'è |
|---|---|
| https://localhost | l'applicazione |
| http://localhost:8025 | Mailpit: le email inviate dall'applicazione (benvenuto, recupero password, scorta minima) |

Il certificato HTTPS è autofirmato e generato al primo avvio: la prima volta
il browser mostra un avviso ("Avanzate → Procedi su localhost").

Cosa succede con `docker compose up`:

```
certs ──► genera il certificato per localhost (solo se manca), poi termina
db ─────► PostgreSQL
migrate ► goose up (dopo che il database è pronto), poi termina
backend ► parte solo se le migrazioni sono riuscite
frontend
nginx ──► unico servizio raggiungibile, solo da questo computer (127.0.0.1:80 e :443)
mailpit ► cattura le email (http://localhost:8025)
```

Per fermare tutto: `docker compose down` (i dati restano nel volume
`postgres_data`; con `docker compose down -v` si cancellano anche quelli e il
certificato).

---

## Sviluppo locale

Requisiti: Go 1.27, Node.js 22, [goose](https://github.com/pressly/goose)
v3.28 (`go install github.com/pressly/goose/v3/cmd/goose@v3.28.0`), Docker.

```bash
cp .env.example .env              # completa DB_PASSWORD e JWT_SECRET
docker compose up -d db mailpit   # solo database e Mailpit

cd backend
make migrate-up                   # applica le migrazioni
make seed ARGS="-username admin -email admin@example.com -first-name Mario -last-name Rossi -birth-date 1980-01-31"
make run                          # API su http://localhost:8080

cd ../frontend                    # in un altro terminale
npm ci
npm run dev                       # interfaccia su http://localhost:3000
```

In sviluppo il browser parla solo con Next.js, che inoltra `/api/*` al
backend: pagine e API restano sulla stessa origine, come in Docker dietro
nginx.

Comandi del backend (cartella `backend`):

| Comando | Cosa fa |
|---|---|
| `make run` | avvia l'API caricando `../.env` |
| `make seed ARGS="…"` | crea un amministratore (password chiesta nel terminale) |
| `make check` | `gofmt`, `go vet` e test |
| `make migrate-up` / `migrate-down` / `migrate-status` | migrazioni con goose |

---

## Test

**Backend**: test del livello service con repository finti in memoria (nessun
database necessario), più i middleware e gli helper HTTP.

```bash
cd backend && make check
```

Coprono, tra l'altro: regole AGID sulle password, assegnazione dei ruoli e
disattivazione degli utenti riservate all'Admin, blocco dell'account al
quinto tentativo, rotazione e
revoca dei refresh token, token JWT manomessi, scaduti o con `alg: none`,
recupero password (token scaduto, già usato, link precedente invalidato),
oscuramento del link nel log delle email, protezione CSRF, limite di
richieste per IP, validazione di utenti e prodotti, movimenti di magazzino e
avvisi di scorta minima.

**Frontend**: test end-to-end con Playwright sull'applicazione reale (backend,
database e Mailpit attivi, un amministratore creato con `make seed`).

```bash
cd frontend
npx playwright install chromium   # la prima volta
E2E_ADMIN_USERNAME=admin E2E_ADMIN_PASSWORD='…' npm run test:e2e

# contro lo stack Docker, in HTTPS dietro nginx
E2E_BASE_URL=https://localhost E2E_ADMIN_USERNAME=admin E2E_ADMIN_PASSWORD='…' npm run test:e2e
```

Coprono accesso e permessi, inventario con carichi, scarichi ed eliminazione,
invito di un nuovo utente fino al primo accesso (leggendo l'email da Mailpit),
rinnovo automatico della sessione e logout.

---

## Requisiti della scheda e dove sono implementati

### Login (§4.1)

| Requisito | Implementazione |
|---|---|
| Username e password obbligatori | `frontend/lib/validation.js`, `backend/internal/auth/handler.go` |
| Password conformi AGID (8 caratteri, maiuscola, numero, carattere speciale), validate in frontend e backend | `passwordSchema` (frontend) e `user.HashPassword` / `validatePassword` (backend), applicate a ogni nuova password |
| Password salvate solo come hash | bcrypt (`backend/internal/user/service.go`) |
| Blocco dopo 5 tentativi falliti | `auth.Service.Login`; contatore, blocco e log scritti in una sola istruzione SQL |
| Registrazione dei tentativi in Log Accessi | tabella `access_logs`, con l'IP del client |
| Recupera password con link valido 1 ora | `POST /api/auth/forgot-password`, token casuale salvato come hash SHA-256 |
| Token invalidato dopo l'uso | `POST /api/auth/reset-password`: token consumato, password cambiata, account sbloccato e sessioni revocate in una sola istruzione SQL |

### Dashboard (§4.2)

Totale utenti, totale prodotti, valore dell'inventario, ultimi 5 movimenti e
grafico dei prodotti per categoria: `GET /api/dashboard`
(`backend/internal/dashboard`), letto in una transazione in sola lettura
perché tutti i numeri vengano dalla stessa istantanea; grafico con Chart.js.

### Gestione utenti (§4.3)

| Requisito | Implementazione |
|---|---|
| Tabella con paginazione, ricerca per username o email | `GET /api/users?q=&page=&page_size=` |
| Inserimento, modifica | `POST /api/users`, `PUT /api/users/{id}` |
| Disattivazione (soft delete) | `DELETE /api/users/{id}` riservato all'Admin (imposta `deleted_at` e chiude le sessioni); riattivazione con `POST /api/users/{id}/reactivate` |
| Assegnazione e modifica del ruolo; solo l'Admin modifica i ruoli | `PUT /api/users/{id}/role` riservato all'Admin; alla creazione solo l'Admin sceglie il ruolo |
| Email di benvenuto alla creazione | l'email contiene il link per scegliere la password: nessuno conosce la password di un altro utente |

### Gestione inventario (§4.4)

| Requisito | Implementazione |
|---|---|
| Inserimento, modifica | `POST /api/products`, `PUT /api/products/{id}` |
| Eliminazione logica | `DELETE /api/products/{id}`, consentita solo con quantità zero |
| Filtro per tipo, ricerca per nome, ordinamento per prezzo o quantità | `GET /api/products?type_id=&q=&sort=price\|quantity&order=asc\|desc` |
| Registrazione automatica dei movimenti | ogni variazione di giacenza è un movimento di carico o scarico, nella stessa transazione dell'aggiornamento del prodotto; la quantità iniziale è registrata come carico |
| Email all'amministratore sotto la soglia minima | a ogni movimento che lascia la giacenza sotto soglia, a tutti gli Admin attivi |

### Sicurezza (§5)

| Requisito | Implementazione |
|---|---|
| Rotte protette da JWT | middleware `RequireAuth` su tutte le rotte tranne login, rinnovo, logout e recupero password |
| RBAC | ruolo nel token, `RequireRole` e controllo nel service |
| Validazione contro SQL injection e XSS | query sempre parametrizzate; validazione dell'input nel service; output sempre come testo (JSON, React, email in solo testo); CSP |
| Protezione CSRF | token double-submit, cookie `SameSite=Strict` |
| Logging centralizzato degli errori | `log/slog` in JSON; gli errori 500 registrano la causa e mostrano al client un messaggio generico |
| HTTPS obbligatorio | nginx: HTTP reindirizzato a HTTPS; cookie `Secure`; HSTS |
| Refresh token con scadenza | 7 giorni, ruotato a ogni rinnovo, revocabile |

---

## Architettura

```
                        ┌──────────────────────── Docker ────────────────────────┐
 browser ──HTTPS──► nginx ──/api/*──► backend (Go) ──► PostgreSQL               │
                    │  └──── /* ────► frontend (Next.js)                         │
                    │                  backend ──SMTP──► Mailpit                  │
                    └─────────────────────────────────────────────────────────────┘
```

**Backend** (`backend/`), organizzato per dominio, un file per livello:

```
handler.go     HTTP: lettura e validazione della richiesta, risposta JSON, codici di stato
   │
service.go     regole di business (nessun SQL, nessun net/http); definisce le
   │           interfacce dei repository che usa
repository.go  SQL con pgx, query parametrizzate
```

Domini: `auth` (login, sessioni, CSRF, recupero password), `user`, `role`,
`product` (prodotti e movimenti), `dashboard`, `mail` (invio SMTP e log delle
email). Pacchetti di supporto: `httpx` (risposte JSON, middleware, limite di
richieste), `config`, `database`, `identity` (utente autenticato nel
contesto), `validation`. Tutte le dipendenze sono assemblate in
`cmd/api/main.go`, senza stato globale.

**Frontend** (`frontend/`): le pagine dell'area riservata sono Client
Component che chiamano `/api` dal browser. I cookie di sessione hanno
`Path=/api`, quindi il server di Next.js non vede mai i token. Un solo client
(`lib/api.js`) aggiunge l'header CSRF e rinnova la sessione su un 401.
Bootstrap Italia è usato con il solo CSS, con componenti React per modali,
schede e messaggi.

---

## Database

PostgreSQL 18, migrazioni SQL con goose (`backend/migrations`), ognuna con la
sua sezione `Down`.

| Tabella | Contenuto |
|---|---|
| `roles` | ruoli (Admin, Operatore, inseriti da migrazione) |
| `users` | utenti: credenziali, ruolo, tentativi falliti, stato (attivo/bloccato), ultimo login, disattivazione |
| `product_types` | tipi di prodotto (Buste, Carta, Toner, inseriti da migrazione) |
| `products` | prodotti: prezzo `NUMERIC(10,2)`, giacenza, soglia minima, creatore, eliminazione logica |
| `stock_movements` | movimenti di carico e scarico (immutabili) |
| `access_logs` | tentativi di accesso con esito e IP |
| `password_resets` | token di recupero (hash), scadenza, utilizzo |
| `refresh_tokens` | sessioni (hash del refresh token), scadenza, revoca |
| `email_logs` | email inviate con esito, link segreti oscurati |

Scelte principali: chiavi primarie UUIDv7; `updated_at` aggiornato da un
trigger; vincoli nel database (`NOT NULL`, `CHECK`, chiavi esterne con
`ON DELETE` esplicito e indice); unicità di username ed email senza
distinzione tra maiuscole e minuscole; nome del prodotto unico solo tra i
prodotti non eliminati.

---

## Sicurezza

- **Password**: bcrypt; regole AGID per ogni nuova password; nessuno sceglie
  la password di un altro utente.
- **Sessioni**: access token JWT di 15 minuti (HS256, algoritmo verificato),
  refresh token opaco di 7 giorni salvato come hash e ruotato a ogni rinnovo;
  disattivare o bloccare un utente chiude le sue sessioni.
- **Cookie**: `HttpOnly`, `Secure`, `SameSite=Strict`; il refresh token è
  inviato solo alle rotte `/api/auth`.
- **CSRF**: token double-submit nel cookie `__Host-csrf_token` (in HTTPS),
  confrontato in tempo costante con l'header `X-CSRF-Token`.
- **Accessi**: blocco dopo 5 tentativi; limite di 10 tentativi al minuto per
  IP su login e reset, 5 ogni 15 minuti sul recupero password; tempi di
  risposta uguali per username esistenti e inesistenti.
- **Recupero password**: stessa risposta per email registrate e non, link
  valido 1 ora e utilizzabile una volta, oscurato nei log.
- **Input**: query parametrizzate, validazione nel service, id controllati
  prima della query, corpo delle richieste limitato a 1 MiB, campi sconosciuti
  rifiutati.
- **Trasporto e browser** (nginx): HTTPS con TLS 1.2/1.3, HSTS,
  Content-Security-Policy, `X-Frame-Options`, `X-Content-Type-Options`,
  `Referrer-Policy: no-referrer`.
- **Infrastruttura**: porte esposte solo su `127.0.0.1`; immagini con versione
  fissata e utente senza privilegi; segreti solo nelle variabili d'ambiente.

---

## API

Tutte le risposte sono JSON; gli errori hanno la forma
`{"error": "messaggio"}` e, per gli errori di validazione,
`{"error": "...", "fields": {"campo": "messaggio"}}`. Le richieste che
modificano dati sulle rotte autenticate richiedono l'header `X-CSRF-Token`.

| Metodo e percorso | Accesso | Descrizione |
|---|---|---|
| `POST /api/auth/login` | pubblico, limitato | accesso, imposta i cookie di sessione |
| `POST /api/auth/refresh` | refresh token | rinnovo della sessione |
| `POST /api/auth/logout` | refresh token | chiusura della sessione corrente |
| `POST /api/auth/forgot-password` | pubblico, limitato | invio del link di recupero |
| `POST /api/auth/reset-password` | pubblico, limitato | nuova password con il token del link |
| `GET /api/auth/me` | autenticato | utente corrente |
| `GET /api/dashboard` | autenticato | dati della dashboard |
| `GET /api/roles` | autenticato | ruoli |
| `GET /api/users` | autenticato | elenco utenti (`q`, `page`, `page_size`, `deactivated`) |
| `POST /api/users` | autenticato | nuovo utente (il ruolo lo sceglie solo l'Admin) |
| `GET /api/users/{id}` | autenticato | dettaglio utente |
| `PUT /api/users/{id}` | autenticato | modifica dei dati |
| `DELETE /api/users/{id}` | **Admin** | disattivazione |
| `POST /api/users/{id}/reactivate` | autenticato | riattivazione |
| `PUT /api/users/{id}/role` | **Admin** | cambio di ruolo |
| `GET /api/product-types` | autenticato | tipi di prodotto |
| `GET /api/products` | autenticato | elenco (`q`, `type_id`, `sort`, `order`, `page`, `page_size`) |
| `POST /api/products` | autenticato | nuovo prodotto con quantità iniziale |
| `GET /api/products/{id}` | autenticato | dettaglio prodotto |
| `PUT /api/products/{id}` | autenticato | modifica (la giacenza cambia solo con i movimenti) |
| `DELETE /api/products/{id}` | autenticato | eliminazione logica, solo con quantità zero |
| `POST /api/products/{id}/movements` | autenticato | carico o scarico |

---

## Struttura del repository

```
risma/
├── backend/
│   ├── cmd/api/          # punto d'ingresso: configurazione, dipendenze, server, arresto ordinato
│   ├── cmd/seed/         # creazione del primo amministratore
│   ├── internal/         # domini (auth, user, role, product, dashboard, mail) e supporto
│   ├── migrations/       # migrazioni SQL (goose)
│   ├── Dockerfile
│   └── Makefile
├── frontend/
│   ├── app/              # pagine: (auth) pubbliche, (app) riservate
│   ├── components/       # componenti con il markup di Bootstrap Italia
│   ├── lib/              # client API, validazione, formattazione
│   ├── e2e/              # test Playwright
│   └── Dockerfile
├── nginx/                # reverse proxy HTTPS e script del certificato
├── docs/decisions.md     # decisioni progettuali, motivazioni e limiti noti
├── docker-compose.yml
└── .env.example
```

---

## Limiti noti

I principali, descritti con le possibili evoluzioni in
[`docs/decisions.md`](docs/decisions.md#limiti-noti-e-possibili-evoluzioni):

- invio delle email in background nello stesso processo, senza coda né nuovi
  tentativi;
- limite di richieste per IP in memoria, valido per una sola istanza del
  backend;
- Content-Security-Policy con `'unsafe-inline'`, richiesto dagli script di
  Next.js e dagli stili di Chart.js;
- un Operatore può modificare i dati di un Admin (la scheda non lo vieta);
- certificato HTTPS autofirmato, adatto solo all'uso locale.
