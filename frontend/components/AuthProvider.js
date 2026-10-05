"use client";

import { usePathname, useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useState } from "react";

import Spinner from "@/components/Spinner";
import { api, AuthError } from "@/lib/api";

const AuthContext = createContext(null);

// AuthProvider carica l'utente autenticato e lo rende disponibile alle
// pagine dell'area riservata. Se la sessione non è valida rimanda al login.
// Il ruolo serve solo a mostrare o nascondere i controlli: i permessi li
// verifica sempre il backend.
export function AuthProvider({ children }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState(null);
  const [error, setError] = useState("");

  const expireSession = useCallback(() => {
    router.replace(`/login?next=${encodeURIComponent(pathname)}`);
  }, [router, pathname]);

  useEffect(() => {
    let cancelled = false;

    api
      .get("/api/auth/me")
      .then((me) => {
        if (!cancelled) setUser(me);
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof AuthError) {
          expireSession();
        } else {
          setError(err.message);
        }
      });

    return () => {
      cancelled = true;
    };
    // L'utente si carica una volta per sessione, non a ogni cambio di pagina.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Anche se la richiesta di logout fallisce si torna al login: al più la
  // sessione resta valida fino alla scadenza, ma l'utente ha chiesto di uscire.
  const logout = useCallback(async () => {
    await api.post("/api/auth/logout").catch(() => {});
    router.replace("/login");
  }, [router]);

  if (error) {
    return (
      <div className="container my-5">
        <div className="alert alert-danger" role="alert">
          {error}
        </div>
      </div>
    );
  }

  if (!user) {
    return <Spinner label="Verifica dell'accesso in corso" />;
  }

  const value = { user, isAdmin: user.role_name === "admin", logout, expireSession };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth va usato dentro AuthProvider");
  }
  return context;
}
