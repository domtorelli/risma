// Client delle API del backend. Tutte le chiamate passano da qui:
//   - i cookie di sessione (HttpOnly) li invia il browser, il JavaScript non li vede;
//   - per le richieste che modificano dati copia il token CSRF dal cookie
//     all'header X-CSRF-Token (protezione double-submit del backend);
//   - se l'access token è scaduto (401) rinnova la sessione una volta e
//     ripete la richiesta.

// Il backend usa il prefisso __Host- solo in HTTPS (i browser non lo
// accettano su http://localhost): in sviluppo il cookie è csrf_token.
const CSRF_COOKIES = ["__Host-csrf_token", "csrf_token"];
const CSRF_HEADER = "X-CSRF-Token";
const SAFE_METHODS = ["GET", "HEAD", "OPTIONS"];

// Rotte per cui un 401 non va gestito con un rinnovo: sono il login e il
// rinnovo stesso, e un loro 401 è la risposta definitiva.
const NO_REFRESH_PATHS = ["/api/auth/login", "/api/auth/refresh"];

// Errore restituito dall'API: message è il messaggio del backend, fields
// contiene gli errori per campo ({ campo: messaggio }) quando ci sono.
export class ApiError extends Error {
  constructor(status, message, fields = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fields = fields;
  }
}

// Errore di sessione assente o scaduta: chi lo riceve rimanda al login.
export class AuthError extends ApiError {
  constructor(message = "Sessione scaduta, accedi di nuovo") {
    super(401, message);
    this.name = "AuthError";
  }
}

function readCookie(name) {
  const prefix = `${name}=`;
  const cookie = document.cookie.split("; ").find((c) => c.startsWith(prefix));
  return cookie ? decodeURIComponent(cookie.slice(prefix.length)) : "";
}

// Rinnovo in corso: se più richieste ricevono 401 insieme, aspettano tutte
// lo stesso rinnovo invece di ruotare il refresh token più volte.
let refreshing = null;

function refreshSession() {
  if (!refreshing) {
    refreshing = fetch("/api/auth/refresh", { method: "POST", credentials: "same-origin" })
      .then((response) => response.ok)
      .catch(() => false)
      .finally(() => {
        refreshing = null;
      });
  }
  return refreshing;
}

async function send(path, { method = "GET", body } = {}) {
  const headers = { Accept: "application/json" };

  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  if (!SAFE_METHODS.includes(method)) {
    const token = CSRF_COOKIES.map(readCookie).find(Boolean);
    if (token) {
      headers[CSRF_HEADER] = token;
    }
  }

  return fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "same-origin",
    cache: "no-store",
  });
}

async function parse(response) {
  if (response.status === 204) {
    return null;
  }

  const data = await response.json().catch(() => null);

  if (response.ok) {
    return data;
  }

  const message = data?.error ?? "Si è verificato un errore imprevisto, riprova";
  if (response.status === 401) {
    throw new AuthError(message);
  }
  throw new ApiError(response.status, message, data?.fields ?? {});
}

// apiFetch esegue una richiesta all'API e restituisce il corpo JSON (null
// per le risposte senza corpo). Lancia ApiError per le risposte di errore e
// AuthError se la sessione non è valida nemmeno dopo il rinnovo.
export async function apiFetch(path, options = {}) {
  let response;
  try {
    response = await send(path, options);
  } catch {
    throw new ApiError(0, "Impossibile contattare il server, controlla la connessione");
  }

  if (response.status === 401 && !NO_REFRESH_PATHS.includes(path)) {
    if (await refreshSession()) {
      response = await send(path, options);
    }
  }

  return parse(response);
}

// Scorciatoie per i metodi usati dall'applicazione.
export const api = {
  get: (path) => apiFetch(path),
  post: (path, body) => apiFetch(path, { method: "POST", body }),
  put: (path, body) => apiFetch(path, { method: "PUT", body }),
  delete: (path) => apiFetch(path, { method: "DELETE" }),
};

// query costruisce la parte ?a=1&b=2 di un URL, tralasciando i valori vuoti.
export function query(params) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") {
      search.set(key, String(value));
    }
  }
  const text = search.toString();
  return text ? `?${text}` : "";
}
