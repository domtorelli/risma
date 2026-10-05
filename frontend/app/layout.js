import "bootstrap-italia/dist/css/bootstrap-italia.min.css";
import "./globals.css";

export const metadata = {
  title: {
    default: "Risma",
    template: "%s | Risma",
  },
  description: "Gestionale di utenti e inventario per un negozio di cancelleria",
};

export default function RootLayout({ children }) {
  return (
    // Bootstrap Italia imposta scroll-behavior: smooth su <html>: dichiararlo
    // permette a Next.js di disattivarlo durante i cambi di pagina.
    <html lang="it" data-scroll-behavior="smooth">
      <body>
        <a className="visually-hidden-focusable" href="#contenuto">
          Vai al contenuto principale
        </a>
        {children}
      </body>
    </html>
  );
}
