"use client";

import { useCallback, useEffect, useState } from "react";

import { useAuth } from "@/components/AuthProvider";
import { api, AuthError } from "@/lib/api";

// useApiData carica i dati di una GET e li ricarica quando cambia path
// (es. filtri e pagina nella query string) o quando si chiama reload.
// Durante un nuovo caricamento restituisce ancora i dati precedenti, così
// una tabella non sparisce a ogni cambio di pagina. Se la sessione è
// scaduta rimanda al login. Con path null non carica nulla (dati che
// servono solo in certi casi, es. i ruoli per l'Admin).
export function useApiData(path) {
  const { expireSession } = useAuth();
  const [version, setVersion] = useState(0);
  const [result, setResult] = useState({ key: null, data: null, error: "" });

  // Identifica la richiesta corrente: "in caricamento" significa che
  // l'ultimo risultato arrivato non è quello della richiesta corrente.
  const key = `${path}#${version}`;

  useEffect(() => {
    if (!path) return;

    let cancelled = false;

    api
      .get(path)
      .then((data) => {
        if (!cancelled) setResult({ key, data, error: "" });
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof AuthError) {
          expireSession();
          return;
        }
        setResult((previous) => ({ key, data: previous.data, error: err.message }));
      });

    // Una risposta arrivata dopo un cambio di filtri non sovrascrive quella nuova.
    return () => {
      cancelled = true;
    };
  }, [path, key, expireSession]);

  const reload = useCallback(() => setVersion((v) => v + 1), []);

  return { data: result.data, error: result.error, loading: Boolean(path) && result.key !== key, reload };
}
