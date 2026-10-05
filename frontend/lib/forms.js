import { ApiError } from "@/lib/api";

// applyServerErrors mostra sotto ai campi gli errori restituiti dal backend
// (ApiError.fields) e restituisce il messaggio generale da mostrare sopra
// il form, se l'errore non riguarda campi del form.
export function applyServerErrors(error, setError, formFields) {
  const fields = error instanceof ApiError ? error.fields : {};
  const known = Object.keys(fields).filter((field) => formFields.includes(field));

  for (const field of known) {
    setError(field, { message: fields[field] });
  }

  return known.length > 0 ? "" : error.message;
}
