"use client";

import { useEffect, useId, useRef } from "react";

import Icon from "./Icon";

// Finestra modale con il markup di Bootstrap Italia, aperta e chiusa da
// React. Si chiude con Esc o con il pulsante; all'apertura il focus va sul
// primo campo e alla chiusura torna dove era prima.
export default function Modal({ title, onClose, children, footer, size }) {
  const titleId = useId();
  const dialogRef = useRef(null);

  // onClose può cambiare a ogni rendering del padre: lo si legge da un ref,
  // così l'effetto sotto gira solo all'apertura e non sposta il focus mentre
  // l'utente scrive.
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });

  useEffect(() => {
    const previousFocus = document.activeElement;
    const firstField = dialogRef.current?.querySelector("input, select, textarea, button:not(.btn-close)");
    firstField?.focus();

    const onKeyDown = (event) => {
      if (event.key === "Escape") {
        onCloseRef.current();
      }
    };

    document.addEventListener("keydown", onKeyDown);
    document.body.classList.add("modal-open");

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.classList.remove("modal-open");
      previousFocus?.focus?.();
    };
  }, []);

  return (
    <>
      <div className="modal fade show d-block" tabIndex={-1} role="dialog" aria-modal="true" aria-labelledby={titleId}>
        <div className={`modal-dialog modal-dialog-centered ${size ? `modal-${size}` : ""}`} role="document" ref={dialogRef}>
          <div className="modal-content">
            <div className="modal-header">
              <h2 className="modal-title h5 no_toc" id={titleId}>
                {title}
              </h2>
              <button type="button" className="btn-close" onClick={onClose} aria-label="Chiudi finestra">
                <Icon name="it-close" />
              </button>
            </div>
            <div className="modal-body">{children}</div>
            {footer && <div className="modal-footer">{footer}</div>}
          </div>
        </div>
      </div>
      <div className="modal-backdrop fade show" />
    </>
  );
}
