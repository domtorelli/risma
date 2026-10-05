"use client";

import { useState } from "react";

import Icon from "./Icon";

// Campo password con il pulsante per mostrare o nascondere il testo.
export default function PasswordField({ id, label, registration, error, autoComplete, hint }) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="form-group">
      <label htmlFor={id} className="active">
        {label} <span aria-hidden="true">*</span>
      </label>
      <div className="input-group">
        <input
          id={id}
          type={visible ? "text" : "password"}
          className={`form-control ${error ? "is-invalid" : ""}`}
          autoComplete={autoComplete}
          required
          aria-invalid={error ? true : undefined}
          aria-describedby={[hint && `${id}-hint`, error && `${id}-error`].filter(Boolean).join(" ") || undefined}
          {...registration}
        />
        <div className="input-group-append">
          <button
            type="button"
            className="btn btn-outline-secondary"
            onClick={() => setVisible((v) => !v)}
            aria-pressed={visible}
            aria-controls={id}
          >
            <Icon name={visible ? "it-password-invisible" : "it-password-visible"} size="sm" />
            <span className="visually-hidden">{visible ? "Nascondi password" : "Mostra password"}</span>
          </button>
        </div>
      </div>
      {hint && (
        <small className="form-text" id={`${id}-hint`}>
          {hint}
        </small>
      )}
      {error && (
        <div className="invalid-feedback d-block" id={`${id}-error`}>
          {error}
        </div>
      )}
    </div>
  );
}
