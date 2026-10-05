# Risma — Decisioni progettuali

Registro delle decisioni prese durante lo sviluppo di Risma, con il motivo di ciascuna. A fine progetto confluisce nelle sezioni "Scelte progettuali" e "Limiti noti e possibili evoluzioni" del README.

## Stack e strumenti

- **Go come backend**: la scheda permetteva Ruby o Go.
- **Monorepo** con `backend/` e `frontend/`: un solo clone e un solo `docker compose up` per avviare tutto.
- **Router della libreria standard (`net/http`)**: da Go 1.22 supporta metodo HTTP e parametri nel pattern delle rotte, quindi non serve un framework esterno.
- **goose per le migrazioni**: un file SQL per migrazione con sezioni Up/Down, numerazione progressiva, utilizzabile anche come libreria per eseguire le migrazioni all'avvio del backend.
- **Migrazioni eseguite dal Makefile, non all'avvio del backend**: nessuna dipendenza in più (goose come libreria richiederebbe anche il ponte `pgx/stdlib`) e le migrazioni si applicano quando si decide. In Docker servirà un passaggio dedicato nel compose.
- **Primo admin creato da un comando separato** (`cmd/seed`), non dal backend all'avvio e non da una migrazione: nessuna password nel repository né nell'ambiente del server. Si scriverà dopo il dominio utenti, perché usa bcrypt e il service.
- **Migrazioni in SQL puro**, senza ORM: lo schema è esplicito e sfrutta le funzionalità native di PostgreSQL.
- **pgx usato direttamente** invece di `database/sql`: il progetto usa solo PostgreSQL, pgx è più veloce e supporta meglio i suoi tipi (UUID, INET, enum).
- **Codice organizzato per dominio** (`internal/user`, `internal/product`, `internal/auth`…) con un file per livello (`handler.go`, `service.go`, `repository.go`) invece di pacchetti per livello: è l'impostazione idiomatica in Go, tiene insieme ciò che cambia insieme e lascia le interfacce nel pacchetto di chi le usa.
- **Query scritte con pgx a mano, senza sqlc**: meno strumenti e nessuna fase di generazione, e ogni riga di SQL e di `Scan` resta visibile. sqlc è una possibile evoluzione (vedi sotto).
- **Logging con `slog` in formato JSON su stdout**, un solo formato in tutti gli ambienti: nessuna configurazione e lo stesso comportamento in sviluppo e produzione. In produzione l'infrastruttura raccoglie lo stdout (`docker compose logs`, driver con rotazione), il backend non apre file. Il logger è creato in `main` e passato ai costruttori, senza stato globale. In sviluppo si legge meglio con `jq`.
- **Makefile nel backend** per avvio, migrazioni, seed e controlli (`make run`, `make seed`, `make check`, `make migrate-up`, `make migrate-down`, `make migrate-status`).
- **JWT con `golang-jwt/jwt/v5`**: la libreria standard non gestisce i JWT e scrivere a mano la verifica di firma, algoritmo e scadenza è un rischio evitabile. È la libreria di riferimento in Go.
- **Email con `net/smtp` della libreria standard** verso Mailpit in sviluppo: per inviare testo semplice non serve una dipendenza.
- **nginx come reverse proxy HTTPS** (`/api` → backend, `/` → frontend): diffuso, configurazione esplicita dei certificati e degli header.
- **Chart.js per il grafico della dashboard**, usato solo in un Client Component.
- **Test del livello service con fake in memoria** dei repository: le interfacce piccole definite dal service rendono i fake banali e i test non richiedono un database.
- **Mailpit in sviluppo**: server SMTP che cattura le email e le mostra in un'interfaccia web, senza spedirle davvero.
- **Versioni delle immagini Docker fissate** (mai `latest`), per avere lo stesso ambiente a ogni clone.

## Configurazione

- **Il backend legge solo variabili d'ambiente** (principio twelve-factor): in Docker le passa il compose, in sviluppo il Makefile carica il `.env` della radice prima di lanciare `go run`. Il codice è identico in tutti gli ambienti.
- **Nel `.env` solo valori di base**: tutto ciò che si ricava da essi (stringa di connessione per goose e per il backend) viene composto dove serve, per non duplicare le credenziali.
- **Stringa di connessione nel formato `chiave=valore`** invece che a URL: una password con caratteri speciali (`@`, `/`) non richiede codifica.
- **`DB_SSLMODE` obbligatoria**: un valore predefinito deve essere quello sicuro; senza la variabile il backend non parte, invece di connettersi in chiaro senza avvisare. `DB_HOST`, `DB_PORT` e `HTTP_PORT` hanno valori predefiniti perché non incidono sulla sicurezza.
- **Variabili obbligatorie nel compose** (`${VAR:?messaggio}`): l'avvio si blocca con un messaggio chiaro se mancano o sono vuote.
- **Ping del database all'avvio**: `pgxpool.New` non apre connessioni finché non serve, quindi il backend verifica subito che il database risponda e si ferma con un errore chiaro se non è raggiungibile.

## Database

- **UUIDv7 come chiavi primarie** (`uuidv7()` di PostgreSQL 18): ordinati nel tempo, mantengono gli indici più compatti rispetto ai UUID casuali.
- **ENUM per i valori stabili, tabelle per quelli gestiti dagli utenti**: stato utente, tipo movimento, esito accesso, tipo ed esito email sono ENUM; ruoli e tipi prodotto sono tabelle. Limite noto degli ENUM: un valore si aggiunge facilmente ma non si può rimuovere.
- **Nome del ruolo letto con una JOIN** in tutte le query sugli utenti; nelle scritture una CTE (`WITH u AS (INSERT/UPDATE ... RETURNING *)`) unisce la riga scritta a `roles`, così anche creazione e modifica restituiscono il nome del ruolo in un solo giro.
- **Ricerca utenti con `ILIKE` e caratteri speciali di LIKE resi letterali** (`%`, `_`, `\`): cercare `a_b` non deve trovare anche `axb`. Il testo cercato resta un parametro della query.
- **Ruoli iniziali Admin e Operatore inseriti da una migrazione dedicata** (la tabella `roles` esiste già in una migrazione condivisa, che non si modifica): servono al funzionamento, come i tipi prodotto. Il codice cerca il ruolo per nome, senza UUID fissi.
- **Tipi prodotto iniziali (Buste, Carta, Toner) inseriti dalla migrazione**: sono necessari al funzionamento, non dati di esempio.
- **Unicità di email e username senza distinzione maiuscole/minuscole** (indici unici su `lower()`), per evitare account che si confondono (es. `admin` / `Admin`).
- **Unicità totale degli utenti anche dopo la disattivazione**: email e username restano legati alla persona, così le tracce nei log restano univoche.
- **Unicità del nome prodotto solo tra prodotti non eliminati** (indice parziale `WHERE deleted_at IS NULL`): un prodotto eliminato può essere rimesso a catalogo con lo stesso nome.
- **Soft delete con `deleted_at`** per utenti e prodotti.
- **Utente bloccato e utente disattivato sono distinti**: `status` (attivo/bloccato) riguarda i tentativi di login falliti; `deleted_at` la disattivazione da parte dell'admin.
- **`updated_at` aggiornato da un trigger**: una sola funzione `set_updated_at()` creata nella prima migrazione, un trigger per tabella nella sua migrazione di creazione. Vale per qualunque modifica, anche fuori dal backend.
- **Movimenti di magazzino immutabili**: nessun `updated_at` né `deleted_at`; gli errori si correggono con un movimento di segno opposto. Quantità sempre positiva, segno dato dal tipo movimento.
- **Vincoli nel database** (`NOT NULL`, `CHECK`, chiavi esterne con `ON DELETE RESTRICT`): il database è l'ultima linea di difesa contro dati non validi.
- **Indici espliciti sulle chiavi esterne**: PostgreSQL, a differenza di MySQL, non li crea automaticamente.
- **Prezzi in `NUMERIC(10, 2)`**, mai in virgola mobile; **date in `TIMESTAMPTZ`** (con fuso orario).
- **Refresh token salvati come hash SHA-256 in `refresh_tokens`, con approccio allowlist**: un refresh token è valido solo se esiste in tabella, non è scaduto e `revoked_at` è `NULL`. Solo il refresh token passa dal database; l'access token si verifica con la sola firma, quindi una revoca ha effetto al più alla sua scadenza. `revoked_at` al posto della cancellazione conserva la traccia e distingue revocato da scaduto; `ON DELETE CASCADE` su `user_id` come in `password_resets`.
- **Log email scritto dopo il tentativo di invio**, con esito `sent` / `failed` e `sent_at` obbligatorio.
- **Log accessi solo per utenti esistenti** (`user_id` obbligatorio).

## Sicurezza

- **Password salvate come hash bcrypt** in colonna `password_hash TEXT`: un eventuale cambio di algoritmo non richiede migrazioni.
- **Token di reset password salvati come hash SHA-256**, non bcrypt: vanno cercati per valore, e un token casuale ad alta entropia non richiede un hash lento.
- **Link di reset oscurato nel corpo salvato in `email_logs`**: l'email inviata contiene il link reale, la copia nel log un segnaposto. La sostituzione avviene prima della scrittura nel database.
- **Token nel frontend in cookie `httpOnly`, `Secure`, `SameSite=Strict`**: il JavaScript non li può leggere, quindi un XSS non li ruba (con `localStorage` sì). Frontend e API sono serviti dallo stesso dominio tramite reverse proxy (`/api` → backend), così CORS è quasi superfluo.
- **Protezione CSRF con token double-submit**: il backend emette un token leggibile da JavaScript che il client rimanda in un header; il backend confronta i due valori. Più robusto di un semplice header personalizzato.
- **RBAC limitato a quanto prevede la scheda** (§4.3, "Solo ruolo Admin può eliminare o modificare ruoli"): l'unico privilegio riservato all'Admin è assegnare o cambiare il ruolo di un utente, anche alla creazione. Un Operatore che crea un utente non sceglie il ruolo: il nuovo utente è sempre Operatore, così nessuno può promuovere sé stesso o altri. Tutto il resto (dashboard, gestione utenti, inventario) è permesso a entrambi i ruoli.
- **Ruoli fissi**: Admin e Operatore arrivano dalla migrazione e non si modificano né si eliminano dall'applicazione; le API espongono i ruoli solo in lettura. Il codice confronta i ruoli per nome (`role.Admin`, `role.Operator`).
- **Controllo del ruolo nel service, oltre che sulla rotta**: `ChangeRole` e `Create` ricevono il ruolo di chi agisce e applicano la regola anche se la rotta è già protetta dal middleware, così la regola non dipende da come il service viene chiamato ed è coperta dai test.
- **Il ruolo si cambia solo con un'operazione dedicata**: la modifica dei dati utente non tocca `role_id`, quindi il controllo Admin sta in un punto solo.
- **Password del primo admin letta da stdin** dal comando `cmd/seed`: non compare nei flag, nella cronologia della shell né tra le variabili d'ambiente.
- **Corpo delle richieste JSON limitato a 1 MiB, campi sconosciuti rifiutati** (`httpx.DecodeJSON`): un errore di battitura nel client emerge subito invece di essere ignorato.
- **Porte dei servizi Docker esposte solo su `127.0.0.1`**.
- **`sslmode=disable` solo in sviluppo locale**.
- **Server HTTP con timeout espliciti** (`http.Server` invece di `http.ListenAndServe`, che non ne ha): header 5 s (difesa da Slowloris), lettura della richiesta 10 s, scrittura della risposta 15 s (margine per le query), connessioni inattive 60 s.
- **Arresto ordinato (graceful shutdown)** su `SIGINT` / `SIGTERM`: il server smette di accettare richieste, attende quelle in corso fino a 15 s (pari al `WriteTimeout`, così nessuna richiesta legittima viene interrotta) e chiude il pool del database. Il periodo di tolleranza di Docker (`stop_grace_period`) dovrà essere superiore, es. 20 s, perché il default di 10 s ucciderebbe il processo prima.

## Requisiti interpretati

- **Struttura di Tipo Prodotto ricostruita per ipotesi** (id e nome), perché non leggibile nella scheda ricevuta: nelle pagine fotografate il contenuto delle sezioni risultava mescolato.

## Limiti noti e possibili evoluzioni

- **Invio email sincrono**, senza coda né nuovi tentativi in caso di errore. Evoluzione: stato `pending`, data di registrazione e processo separato per l'invio.
- **Nessun messaggio di errore in `email_logs`**: la causa del fallimento resta nei log del backend.
- **Tentativi di accesso con username inesistenti non registrati**. Evoluzione: una migrazione che rende `user_id` facoltativo e aggiunge lo username tentato, per riconoscere chi prova a indovinare gli account.
- **Immutabilità dei movimenti garantita dal backend, non dal database**. Evoluzione: trigger che blocca `UPDATE` e `DELETE` su `stock_movements`.

- **`os.Exit(1)` nella goroutine del server** se `ListenAndServe` fallisce (es. porta occupata): salta i `defer`, accettabile perché accade all'avvio. Evoluzione: canale di errori e `select` in `main`.
- **Un Operatore può modificare o disattivare un Admin**: la scheda non lo vieta. Cambiando l'email di un Admin e chiedendo il recupero password, un Operatore potrebbe prenderne l'account. Evoluzione: impedire a un Operatore di agire su utenti Admin.
- **Nessuna protezione dell'ultimo Admin**: un Admin può declassare o disattivare sé stesso o l'ultimo Admin rimasto. Evoluzione: rifiutare l'operazione se non resterebbe alcun Admin attivo.
- **Password del seed visibile mentre si digita**: viene letta come una riga di stdin. Evoluzione: `golang.org/x/term` per disattivare l'eco.
- **Totale e pagina dell'elenco utenti letti con due query separate**: con scritture concorrenti il totale può differire di poco dalle righe mostrate. Accettabile per una paginazione.
- **Migrazioni non automatiche all'avvio**: è possibile avviare il backend su uno schema vecchio. Evoluzione: goose come libreria con i file incorporati (`go:embed`) e un lock per le istanze multiple.
- **Nessuna pulizia dei refresh token scaduti**: la tabella cresce. Evoluzione: routine periodica di cancellazione.
- **Nessun riconoscimento del riuso di un refresh token già ruotato**. Evoluzione: riferimento al token successivo e revoca dell'intera famiglia.
- **Query con pgx a mano**: più codice ripetitivo e controllo dei tipi solo a runtime. Evoluzione: sqlc per generare il codice dall'SQL.

## In sospeso

- [x] Tabella `refresh_tokens` (non prevista dalla scheda, necessaria per revocare i token): token salvato come hash, scadenza, revoca.
- [x] Server HTTP con timeout espliciti (difesa da client lenti, es. Slowloris).
- [x] Arresto ordinato del server (graceful shutdown).
- [ ] Quando il backend sarà containerizzato: `stop_grace_period` di almeno 20 s nel compose.
