"use client";

import { useEffect, useState } from "react";

// useDebouncedValue restituisce value solo dopo che è rimasto invariato per
// delay millisecondi: la ricerca parte quando l'utente smette di scrivere,
// non a ogni tasto.
export function useDebouncedValue(value, delay = 300) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}
