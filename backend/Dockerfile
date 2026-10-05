# Immagine del backend: API, comando seed e goose per le migrazioni.
# Build a più stadi: si compila in un'immagine Go completa, si esegue in
# un'immagine minima (distroless) senza shell né gestore di pacchetti.

# --- Compilazione ---
FROM golang:1.27.1-alpine3.23@sha256:0908ac9b9319e09d7c238aabe914e0395c51d63c4e3d0ae8c554fda9158a5769 AS build

WORKDIR /src

# Prima le sole dipendenze: questo strato resta in cache finché go.mod e
# go.sum non cambiano, anche se cambia il codice.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Binari statici (senza libreria C), eseguibili nell'immagine distroless.
ENV CGO_ENABLED=0

RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# goose alla stessa versione usata in sviluppo, compilato con il solo
# driver PostgreSQL.
RUN go install -trimpath -ldflags="-s -w" \
      -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
      github.com/pressly/goose/v3/cmd/goose@v3.28.0 \
 && cp /go/bin/goose /out/goose

# --- Esecuzione ---
# Versione fissata con il digest: il tag "nonroot" cambia nel tempo.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY --from=build /out/ /usr/local/bin/
COPY migrations /migrations

# Utente senza privilegi (uid 65532) già definito dall'immagine.
USER nonroot:nonroot

EXPOSE 8080

# Il comando predefinito è l'API; "seed" e "goose" si lanciano con
# docker compose run (vedi README).
CMD ["api"]
