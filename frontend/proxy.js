import { NextResponse } from "next/server";

// Controllo "ottimistico" sulle pagine riservate: chi non ha il cookie del
// token CSRF (impostato al login, cancellato al logout) va al login senza
// scaricare la pagina. Non è il controllo di sicurezza: i cookie di sessione
// hanno Path=/api e Next.js non li vede; ogni dato passa dal backend, che
// verifica il token a ogni richiesta e risponde 401 se la sessione non vale.
// Il cookie ha il prefisso __Host- solo in HTTPS (vedi lib/api.js).
const SESSION_HINT_COOKIES = ["__Host-csrf_token", "csrf_token"];

export function proxy(request) {
  if (SESSION_HINT_COOKIES.some((name) => request.cookies.has(name))) {
    return NextResponse.next();
  }

  const loginURL = new URL("/login", request.url);
  loginURL.searchParams.set("next", request.nextUrl.pathname);

  return NextResponse.redirect(loginURL);
}

export const config = {
  matcher: ["/dashboard/:path*", "/utenti/:path*", "/inventario/:path*"],
};
