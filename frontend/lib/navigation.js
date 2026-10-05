// safeNextPath restituisce la pagina a cui tornare dopo il login, solo se è
// un percorso interno: "//sito.it" o "https://sito.it" permetterebbero di
// mandare l'utente su un sito esterno (open redirect).
export function safeNextPath(next) {
  if (typeof next === "string" && next.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\")) {
    return next;
  }
  return "/dashboard";
}
