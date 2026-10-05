#!/bin/sh
# Genera un certificato autofirmato per localhost, valido un anno, se non
# esiste già. Docker lo esegue con il servizio "certs" (immagine con
# openssl) e salva i file nel volume nginx_certs; si può anche lanciare a
# mano indicando la cartella di destinazione.
set -eu

dir="${1:?indica la cartella di destinazione}"

if [ -f "$dir/cert.pem" ] && [ -f "$dir/key.pem" ]; then
    echo "certificato già presente in $dir"
    exit 0
fi

mkdir -p "$dir"

openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 365 \
    -keyout "$dir/key.pem" -out "$dir/cert.pem" \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

# La chiave privata è leggibile solo dal proprietario (root nel container nginx).
chmod 600 "$dir/key.pem"
chmod 644 "$dir/cert.pem"

echo "certificato creato in $dir"
