# Risma — frontend

Interfaccia di Risma in Next.js (App Router, JavaScript) con Bootstrap Italia.
Le scelte progettuali sono in [`../docs/decisions.md`](../docs/decisions.md).

## Avvio in sviluppo

Serve il backend in esecuzione (`make run` nella cartella `backend`).

```bash
npm install
npm run dev        # http://localhost:3000
```

`npm run dev` e `npm run build` copiano prima font e icone di Bootstrap Italia
in `public/bootstrap-italia/` (`npm run assets`): non sono nel repository.

In sviluppo Next.js inoltra le richieste `/api/*` al backend
(`BACKEND_URL`, predefinito `http://localhost:8080`), così browser, pagine e
API sono sulla stessa origine.

## Test end-to-end

I test Playwright usano l'applicazione reale: backend, database, Mailpit e
un Admin creato con `make seed`.

```bash
E2E_ADMIN_USERNAME=admin E2E_ADMIN_PASSWORD='...' npm run test:e2e
```

Il backend limita i tentativi di login a 10 al minuto per IP, e in sviluppo
tutte le richieste arrivano da Next.js con lo stesso IP: tra due esecuzioni
ravvicinate attendi un minuto.

## Struttura

```
app/
├── (auth)/          # pagine pubbliche: login, recupero e reset password
├── (app)/           # area riservata: dashboard, utenti, inventario
├── layout.js        # CSS di Bootstrap Italia, lingua, link "vai al contenuto"
└── globals.css      # font e pochi adattamenti
components/          # campi dei form, modale, paginazione, intestazione…
lib/
├── api.js           # client delle API: header CSRF e rinnovo della sessione
├── validation.js    # schemi zod, stesse regole del backend
└── format.js        # importi, date ed etichette in italiano
proxy.js             # rimanda al login chi non ha una sessione
e2e/                 # test Playwright
```
