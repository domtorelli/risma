"use client";

import { useState } from "react";

import Alert from "./Alert";
import Modal from "./Modal";

// Richiesta di conferma per un'azione (es. disattivare un utente). onConfirm
// è asincrona: se fallisce il messaggio d'errore resta nella finestra.
export default function ConfirmModal({ title, children, confirmLabel, danger = false, onConfirm, onClose }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const confirm = async () => {
    setBusy(true);
    setError("");
    try {
      await onConfirm();
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  };

  return (
    <Modal
      title={title}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="btn btn-outline-secondary" onClick={onClose} disabled={busy}>
            Annulla
          </button>
          <button type="button" className={`btn ${danger ? "btn-danger" : "btn-primary"}`} onClick={confirm} disabled={busy}>
            {confirmLabel}
          </button>
        </>
      }
    >
      <Alert type="danger">{error}</Alert>
      {children}
    </Modal>
  );
}
