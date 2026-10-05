// Formattazione dei valori per l'interfaccia, all'italiana.

// formatEuro trasforma un importo decimale del backend (es. "1234.5",
// "1382.97") in "1.234,50 €". Lavora sulla stringa, senza convertirla in
// numero: gli importi non passano mai da un float.
export function formatEuro(decimal) {
  const [integerPart, decimalPart = ""] = String(decimal).split(".");
  const thousands = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  return `${thousands},${decimalPart.padEnd(2, "0").slice(0, 2)} €`;
}

const dateTimeFormat = new Intl.DateTimeFormat("it-IT", { dateStyle: "short", timeStyle: "short" });
const dateFormat = new Intl.DateTimeFormat("it-IT", { dateStyle: "short", timeZone: "UTC" });

// formatDateTime mostra data e ora di un istante ISO 8601 nel fuso del browser.
export function formatDateTime(iso) {
  return iso ? dateTimeFormat.format(new Date(iso)) : "—";
}

// formatDate mostra una data AAAA-MM-GG (es. la data di nascita) senza
// spostarla di un giorno per effetto del fuso orario.
export function formatDate(isoDate) {
  return isoDate ? dateFormat.format(new Date(`${isoDate}T00:00:00Z`)) : "—";
}

export const movementLabels = { in: "Carico", out: "Scarico" };

export const roleLabels = { admin: "Amministratore", operatore: "Operatore" };
