"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { useAuth } from "@/components/AuthProvider";
import Icon from "@/components/Icon";
import { roleLabels } from "@/lib/format";

const sections = [
  { href: "/dashboard", label: "Dashboard", icon: "it-chart-line" },
  { href: "/utenti", label: "Utenti", icon: "it-user" },
  { href: "/inventario", label: "Inventario", icon: "it-box" },
];

// Intestazione dell'area riservata (Bootstrap Italia): marchio, utente
// collegato con il ruolo, uscita e navigazione tra le sezioni.
export default function AppHeader() {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  return (
    <header className="it-header-wrapper">
      <div className="it-header-center-wrapper">
        <div className="container">
          <div className="it-header-center-content-wrapper">
            <div className="it-brand-wrapper">
              <Link href="/dashboard">
                <Icon name="it-pa" size="lg" color="white" className="me-2" />
                <div className="it-brand-text">
                  <div className="it-brand-title">Risma</div>
                  <div className="it-brand-tagline d-none d-md-block">Gestionale cancelleria</div>
                </div>
              </Link>
            </div>
            <div className="it-right-zone">
              <span className="text-white me-3 d-none d-md-inline">
                {user.first_name} {user.last_name}
                <span className="badge bg-white text-primary ms-2">{roleLabels[user.role_name] ?? user.role_name}</span>
              </span>
              <button type="button" className="btn btn-outline-white btn-sm btn-icon" onClick={logout}>
                <Icon name="it-logout" size="sm" color="white" className="me-1" />
                <span>Esci</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      <div className="bg-white border-bottom">
        <div className="container">
          <nav aria-label="Sezioni dell'applicazione">
            <ul className="nav nav-tabs nav-tabs-icon-text border-0">
              {sections.map((section) => {
                const active = pathname.startsWith(section.href);
                return (
                  <li className="nav-item" key={section.href}>
                    <Link
                      href={section.href}
                      className={`nav-link ${active ? "active" : ""}`}
                      aria-current={active ? "page" : undefined}
                    >
                      <Icon name={section.icon} className="me-2" />
                      {section.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>
        </div>
      </div>
    </header>
  );
}
