"use client";

import { useEffect, useState } from "react";

// useDebouncedValue restituisce value solo dopo che è rimasto invariato per
// delay millisecondi: la ricerca parte quando l'utente smette di scrivere,
// non a ogni tasto. Un valore vuoto (campo svuotato o azzerato) passa subito:
// non c'è nulla da cercare e l'attesa darebbe solo l'impressione di un blocco.
export function useDebouncedValue(value, delay = 300) {
  const [debounced, setDebounced] = useState(value);

  // Lo stato si aggiusta durante il rendering, come per la pagina in Users.js.
  if (value === "" && debounced !== "") {
    setDebounced("");
  }

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}
