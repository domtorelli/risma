import Link from "next/link";

import Icon from "./Icon";

// Marchio dell'intestazione di Bootstrap Italia. Gli stili di icona e titolo
// della libreria valgono solo dentro un link (.it-brand-wrapper a): per questo
// è sempre un link, anche nelle pagine di accesso, dove riporta all'accesso.
export default function Brand({ href }) {
  return (
    <div className="it-brand-wrapper">
      <Link href={href}>
        <Icon name="it-pa" size="lg" color="white" className="me-2" />
        <div className="it-brand-text">
          <div className="it-brand-title">Risma</div>
          <div className="it-brand-tagline d-none d-md-block">Gestionale cancelleria</div>
        </div>
      </Link>
    </div>
  );
}
