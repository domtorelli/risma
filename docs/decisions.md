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
- **Token CSRF nel cookie `__Host-csrf_token`**, creato al login, mantenuto e prolungato al rinnovo (le altre schede continuano a funzionare), cancellato al logout. Il prefisso `__Host-` obbliga `Secure`, `Path=/` e nessun `Domain`, così un sottodominio non può sovrascriverlo. Il confronto con l'header `X-CSRF-Token` è in tempo costante.
- **CSRF verificato sulle rotte autenticate**, per `POST`, `PUT` e `DELETE`. Le rotte pubbliche (login, rinnovo, logout, recupero e reset password) non lo richiedono: sono protette dai cookie `SameSite=Strict` e, dove servono, da credenziali o token che un sito esterno non conosce.
- **Limite di richieste per IP nell'applicazione**, con un token bucket scritto con la libreria standard: login e reset password 10 al minuto, recupero password 5 ogni 15 minuti (ogni richiesta può inviare un'email). Oltre il limite: 429 con `Retry-After`. L'IP è quello di `ClientIP`, quindi un `X-Real-IP` falso non aggira il limite. Una goroutine elimina i secchi tornati pieni e si ferma all'arresto del server.
- **Nessun CORS**: frontend e API sono sempre sulla stessa origine (in sviluppo tramite il proxy di Next.js, in Docker tramite nginx), quindi il browser blocca già le richieste da altre origini.
- **RBAC limitato a quanto prevede la scheda** (§4.3, "Solo ruolo Admin può eliminare o modificare ruoli"): l'unico privilegio riservato all'Admin è assegnare o cambiare il ruolo di un utente, anche alla creazione. Un Operatore che crea un utente non sceglie il ruolo: il nuovo utente è sempre Operatore, così nessuno può promuovere sé stesso o altri. Tutto il resto (dashboard, gestione utenti, inventario) è permesso a entrambi i ruoli.
- **Ruoli fissi**: Admin e Operatore arrivano dalla migrazione e non si modificano né si eliminano dall'applicazione; le API espongono i ruoli solo in lettura. Il codice confronta i ruoli per nome (`role.Admin`, `role.Operator`).
- **Controllo del ruolo nel service, oltre che sulla rotta**: `ChangeRole` e `Create` ricevono il ruolo di chi agisce e applicano la regola anche se la rotta è già protetta dal middleware, così la regola non dipende da come il service viene chiamato ed è coperta dai test.
- **Il ruolo si cambia solo con un'operazione dedicata**: la modifica dei dati utente non tocca `role_id`, quindi il controllo Admin sta in un punto solo.
- **Password del primo admin letta da stdin** dal comando `cmd/seed`: non compare nei flag, nella cronologia della shell né tra le variabili d'ambiente.
- **Access token di 15 minuti, refresh token di 7 giorni**: la revoca e il cambio di ruolo hanno effetto al più entro 15 minuti, e una settimana di sessione è comoda per un gestionale.
- **JWT firmati HS256, verificati accettando solo HS256**, con emittente e scadenza obbligatori: un token con `"alg": "none"` o un altro algoritmo viene rifiutato. `JWT_SECRET` è obbligatoria e lunga almeno 32 byte, altrimenti il backend non parte.
- **Refresh token opaco e casuale** (128 bit da `crypto/rand`), non un JWT: ha valore solo se il suo hash è nel database. **Ruota a ogni rinnovo**: il vecchio viene revocato e il nuovo inserito in una sola istruzione SQL, quindi due rinnovi concorrenti dello stesso token non producono due sessioni.
- **Il rinnovo controlla che l'utente sia ancora attivo e non disattivato**: bloccare o disattivare un utente chiude le sue sessioni al più alla scadenza dell'access token, senza revoche esplicite.
- **Cookie sempre `Secure`, anche in sviluppo**: i browser considerano sicuro `localhost` e in Docker si passa da HTTPS, quindi non serve un'opzione per disattivarlo (sarebbe un errore di configurazione in attesa di accadere). Il cookie del refresh token ha `Path=/api/auth`, così il browser lo invia solo a rinnovo e logout.
- **Blocco dopo 5 tentativi falliti consecutivi**: un accesso riuscito azzera il contatore. Il tentativo fallito, l'aggiornamento del contatore e il blocco avvengono in una sola istruzione SQL, quindi il log e lo stato dell'utente non possono divergere.
- **Su un account bloccato la password non viene verificata**: se lo fosse, una risposta diversa per la password giusta permetterebbe di continuare a indovinarla nonostante il blocco. Il tentativo viene comunque registrato.
- **Messaggio esplicito per l'account bloccato**: l'utente capisce cosa fare (recupero password). Rivela che lo username esiste, ma solo dopo 5 tentativi su quell'account.
- **Sblocco solo con il recupero password**: nessuna API di sblocco manuale.
- **Al login si controlla solo la presenza di username e password**: un utente nasce solo con una password conforme alle regole AGID, quindi una password non conforme è semplicemente errata e conta come tentativo fallito. Il frontend non applica le regole AGID al login.
- **Username inesistente: confronto bcrypt con un hash fittizio**, così il tempo di risposta non rivela quali username esistono.
- **Il logout revoca solo la sessione corrente**: le altre sessioni dell'utente restano valide.
- **IP del client da `X-Real-IP` solo se la connessione arriva da un proxy fidato** (`TRUSTED_PROXIES`, IP o reti CIDR); altrimenti si usa l'indirizzo della connessione. Un client che raggiunge il backend direttamente non può falsificare l'IP registrato in `access_logs`. Senza la variabile non ci si fida di nessun proxy.
- **Recupero password richiesto con l'email**, il canale a cui arriva il link. La risposta è sempre la stessa, che l'email sia registrata o no, così non si scopre quali email esistono. Un utente bloccato può chiedere il recupero: è l'unico modo per sbloccarsi.
- **Una nuova richiesta di recupero invalida i link precedenti** non ancora usati: resta valido solo l'ultimo inviato. I link precedenti vengono fatti scadere (`expires_at = now()`) invece di segnarli come usati, perché non lo sono stati.
- **Il reset consuma il token, cambia la password, sblocca l'account e revoca tutti i refresh token dell'utente in una sola istruzione SQL**: chi conosceva la vecchia password perde subito la sessione (al più entro la scadenza dell'access token). Dopo il reset l'utente torna al login: solo il login apre sessioni.
- **Email di recupero inviata in background**, dopo la risposta: il tempo di risposta non rivela se l'email esiste. Gli invii in corso sono contati e l'arresto ordinato del server li attende prima di chiudere il pool del database.
- **Email in solo testo**: nessun HTML da proteggere. Corpo codificato quoted-printable e oggetto con la codifica MIME degli header, per le lettere accentate; un a capo nell'oggetto viene rifiutato (header injection).
- **Valori segreti delle email dichiarati da chi compone il messaggio** (`mail.Message.Secrets`): il service delle email li sostituisce con `[oscurato]` nella copia salvata in `email_logs`, prima della scrittura. La regola sta in un punto solo e vale per ogni tipo di email.
- **Invio SMTP con timeout e STARTTLS se offerto**; `smtp.PlainAuth` rifiuta di inviare credenziali su una connessione non cifrata (tranne verso localhost). Senza `SMTP_USERNAME` non ci si autentica, come con Mailpit.
- **`APP_BASE_URL` obbligatoria**: serve per i link nelle email, e un valore predefinito sbagliato produrrebbe link non funzionanti.
- **Una sola funzione per le nuove password** (`user.HashPassword`): verifica le regole AGID e calcola l'hash bcrypt, sia alla creazione dell'utente sia al reset.
- **Nessuno sceglie la password di un altro utente**: chi crea un utente non indica la password. Il nuovo utente riceve un'email di benvenuto con un link per sceglierla, valido 24 ore (più del recupero, perché non se lo aspetta); fino ad allora la password è casuale e nessuno la conosce. Il link riusa `password_resets` e la pagina del recupero. Solo il comando seed imposta una password (`CreateWithPassword`), perché il primo Admin non ha nessuno che possa invitarlo.
- **Disattivare un utente ne revoca le sessioni** nella stessa istruzione SQL: senza la revoca, dopo una riattivazione i vecchi refresh token tornerebbero validi.
- **Utenti disattivati consultabili e riattivabili**: l'elenco ha un filtro `deactivated=true`; la riattivazione è un'operazione dedicata (`POST /api/users/{id}/reactivate`). Grazie all'unicità totale di username ed email, riattivare non può creare duplicati.
- **Nessuno può disattivare sé stesso**, per non chiudersi fuori per errore.
- **Modifica utente con `PUT`**: il client invia tutti i campi modificabili; non c'è ambiguità tra campo assente e campo vuoto.
- **Regole sui dati utente nel service, non nell'handler**: username di 3–50 caratteri tra lettere, numeri, `.`, `_` e `-`; email semplice senza nome visualizzato, al massimo 254 caratteri; nome e cognome obbligatori, al massimo 100 caratteri, senza caratteri di controllo; data di nascita dopo il 1900 e prima di oggi. Valgono per ogni chiamante (API e seed) e sono coperte dai test. Gli spazi iniziali e finali vengono tolti prima della validazione. Gli errori riportano tutti i campi non validi insieme, con messaggi da mostrare accanto ai campi del form.
- **Id controllati prima della query**: un id malformato nel percorso risponde 404, invece di arrivare a PostgreSQL come errore di sintassi (500).
- **Difesa dall'XSS sull'output, non filtrando l'input**: i testi sono salvati così come sono (le query sono parametrizzate, quindi non c'è rischio di SQL injection). L'encoder JSON rende `<`, `>` e `&` come sequenze `\u003c`; React mostra i valori come testo; le email sono in solo testo. Filtrare l'input altererebbe dati legittimi (es. "D'Annunzio") senza proteggere gli altri punti in cui un dato viene mostrato.
- **La giacenza cambia solo con un movimento di carico o scarico**: il form del prodotto non ha la quantità (un campo `stock_quantity` nella modifica viene rifiutato). Ogni variazione ha tipo, autore e nota. La quantità iniziale indicata alla creazione è registrata come carico ("Carico iniziale").
- **Prodotto e movimento nella stessa transazione** (`pgx.BeginFunc`): o vengono salvati entrambi o nessuno. Nel movimento l'`UPDATE` della giacenza blocca la riga fino al commit, quindi i movimenti concorrenti sullo stesso prodotto vengono applicati uno alla volta; il `CHECK (stock_quantity >= 0)` rifiuta lo scarico oltre la giacenza sul valore aggiornato, senza bisogno di un `SELECT ... FOR UPDATE` separato. Verificato con 20 scarichi concorrenti da 1 su una giacenza di 10: 10 riusciti, 10 rifiutati, giacenza 0.
- **Avviso di scorta minima a ogni movimento che lascia la giacenza sotto la soglia** (e alla creazione di un prodotto già sotto soglia), a tutti gli Admin non disattivati. "Sotto" significa strettamente minore: con soglia 0 non parte mai. Modificare la soglia non invia avvisi. L'email parte in background dopo il commit: un errore di invio non annulla il movimento.
- **Eliminazione logica anche con giacenza positiva**; i movimenti restano. I prodotti eliminati non si ripristinano: si può creare un nuovo prodotto con lo stesso nome (indice unico parziale).
- **Prezzo come stringa decimale nel JSON** (`"12.50"`): letto con `price_per_unit::text` e validato con un'espressione regolare che rispecchia `NUMERIC(10, 2)`, senza passare da un `float64`.
- **Ordinamento da un elenco chiuso**: `sort` (`name`, `price`, `quantity`) e `order` (`asc`, `desc`) sono validati dal service e tradotti in clausole SQL fisse; l'input dell'utente non entra mai nel testo della query. Un valore non previsto risponde 400.
- **Dashboard in un solo endpoint** (`GET /api/dashboard`), visibile a tutti gli utenti autenticati: totale utenti non disattivati, totale prodotti non eliminati, valore dell'inventario (somma di giacenza × prezzo, stringa decimale come i prezzi), ultimi 5 movimenti e numero di prodotti per tipo.
- **Query della dashboard in una transazione in sola lettura `REPEATABLE READ`**: tutte leggono la stessa istantanea del database, quindi un movimento salvato nel frattempo non produce numeri incoerenti tra loro (es. valore dell'inventario e ultimi movimenti). Una query per indicatore, aggregazioni nel database, nessun N+1.
- **Gli ultimi movimenti comprendono quelli di prodotti eliminati**, segnalati come tali: sono lo storico reale del magazzino.
- **Il grafico conta i prodotti per tipo partendo dai tipi** (`LEFT JOIN`): un tipo senza prodotti compare con zero invece di sparire.
- **Errori di validazione e ricerca `ILIKE` condivisi tra i domini** (`internal/validation`, `database.ContainsPattern`), invece di ripeterli in ogni pacchetto.
- **Pacchetto `identity` per l'utente autenticato nel contesto**: `auth` importa `user`, quindi `user` non può importare `auth`. `identity` contiene solo il tipo e le funzioni sul contesto e può essere importato da tutti i domini. Per lo stesso motivo l'invio dell'invito è un'interfaccia definita da `user` e implementata da `auth`, e i middleware arrivano agli handler come funzioni passate da `main`.
- **Messaggi di errore dell'API in italiano**, perché arrivano all'utente dell'interfaccia; gli errori interni di Go restano in inglese, come da convenzione. Gli errori 500 mostrano un messaggio generico e la causa va solo nei log.
- **Corpo delle richieste JSON limitato a 1 MiB, campi sconosciuti rifiutati** (`httpx.DecodeJSON`): un errore di battitura nel client emerge subito invece di essere ignorato.
- **Porte dei servizi Docker esposte solo su `127.0.0.1`**.
- **`sslmode=disable` solo in sviluppo locale**.
- **Server HTTP con timeout espliciti** (`http.Server` invece di `http.ListenAndServe`, che non ne ha): header 5 s (difesa da Slowloris), lettura della richiesta 10 s, scrittura della risposta 15 s (margine per le query), connessioni inattive 60 s.
- **Arresto ordinato (graceful shutdown)** su `SIGINT` / `SIGTERM`: il server smette di accettare richieste, attende quelle in corso fino a 15 s (pari al `WriteTimeout`, così nessuna richiesta legittima viene interrotta) e chiude il pool del database. Il periodo di tolleranza di Docker (`stop_grace_period`) dovrà essere superiore, es. 20 s, perché il default di 10 s ucciderebbe il processo prima.

## Requisiti interpretati

- **Struttura di Tipo Prodotto ricostruita per ipotesi** (id e nome), perché non leggibile nella scheda ricevuta: nelle pagine fotografate il contenuto delle sezioni risultava mescolato.

## Limiti noti e possibili evoluzioni

- **Invio email in background nello stesso processo, senza coda né nuovi tentativi**: se il backend si arresta oltre il tempo di tolleranza o l'SMTP non risponde, l'email è persa (resta la riga `failed` in `email_logs`). Evoluzione: stato `pending`, coda nel database e processo separato per l'invio con nuovi tentativi.
- **Il tempo di risposta del recupero password differisce di pochi millisecondi** tra email registrata e non registrata, per la scrittura del token nel database. Evoluzione: eseguire anche la scrittura in background.
- **Il token di reset viaggia nel percorso dell'URL**: può finire nei log di accesso del reverse proxy. Va escluso dai log di nginx e la pagina deve usare `Referrer-Policy: no-referrer`.
- **Nessun messaggio di errore in `email_logs`**: la causa del fallimento resta nei log del backend.
- **Tentativi di accesso con username inesistenti non registrati**. Evoluzione: una migrazione che rende `user_id` facoltativo e aggiunge lo username tentato, per riconoscere chi prova a indovinare gli account.
- **Immutabilità dei movimenti garantita dal backend, non dal database**. Evoluzione: trigger che blocca `UPDATE` e `DELETE` su `stock_movements`.

- **`os.Exit(1)` nella goroutine del server** se `ListenAndServe` fallisce (es. porta occupata): salta i `defer`, accettabile perché accade all'avvio. Evoluzione: canale di errori e `select` in `main`.
- **Un Operatore può modificare o disattivare un Admin**: la scheda non lo vieta. Cambiando l'email di un Admin e chiedendo il recupero password, un Operatore potrebbe prenderne l'account. Evoluzione: impedire a un Operatore di agire su utenti Admin.
- **Nessuna protezione dell'ultimo Admin**: un Admin può declassare o disattivare sé stesso o l'ultimo Admin rimasto. Evoluzione: rifiutare l'operazione se non resterebbe alcun Admin attivo.
- **Password del seed visibile mentre si digita**: viene letta come una riga di stdin. Evoluzione: `golang.org/x/term` per disattivare l'eco.
- **Chiunque conosca uno username può bloccarne l'account** con 5 tentativi errati: è una conseguenza del blocco richiesto dalla scheda. Evoluzione: limite di tentativi per IP prima del blocco per utente.
- **Una password resa non conforme alle regole AGID modificando direttamente il database** non viene riconosciuta come tale al login.
- **Un hash bcrypt corrotto nel database** viene trattato come password errata, senza un errore dedicato nei log.
- **Se l'invito fallisce dopo la creazione l'utente resta creato** e l'API risponde 500: l'utente può comunque usare "Recupera password". Evoluzione: creazione utente e token di invito nella stessa transazione.
- **Un link di invito o di recupero non ancora scaduto torna utilizzabile se l'utente viene riattivato**: il link è stato inviato alla sua email, quindi il rischio è basso.
- **Avvisi di scorta ripetuti**: finché la giacenza resta sotto soglia, ogni movimento invia un avviso a ogni Admin. Evoluzione: avviso solo quando la giacenza scende sotto la soglia.
- **Nessuno storico dei movimenti per prodotto nelle API**: i movimenti sono salvati e la dashboard mostra gli ultimi 5. Evoluzione: elenco paginato dei movimenti di un prodotto.
- **Limite di richieste in memoria**: vale per una sola istanza del backend e si azzera al riavvio; con IPv6 un attaccante che dispone di un'intera rete può cambiare indirizzo. Evoluzione: limite in nginx o in un archivio condiviso (es. Redis), raggruppando gli IPv6 per rete /64.
- **Nessun id di richiesta nei log**: un errore 500 si ritrova nei log per ora, metodo e percorso. Evoluzione: middleware con `X-Request-ID` riportato nei log e nella risposta.
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
- [ ] Header di sicurezza (HSTS, CSP, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy: no-referrer`): decisione presa di impostarli in nginx; da riprendere a fine progetto, valutando se aggiungerli anche al backend.
