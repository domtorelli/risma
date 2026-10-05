"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import { TextareaField, TextField } from "@/components/FormFields";
import Modal from "@/components/Modal";
import { api } from "@/lib/api";
import { applyServerErrors } from "@/lib/forms";
import { movementSchema } from "@/lib/validation";

const FIELDS = ["type", "quantity", "note"];

// Registrazione di un carico o scarico: il backend aggiorna la giacenza e
// salva il movimento nella stessa transazione.
export default function MovementModal({ product, onSaved, onClose }) {
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(movementSchema),
    defaultValues: { type: "in", quantity: 1, note: "" },
  });

  const onSubmit = async (values) => {
    setServerError("");
    try {
      const updated = await api.post(`/api/products/${product.id}/movements`, values);
      onSaved(updated, values);
    } catch (error) {
      setServerError(applyServerErrors(error, setError, FIELDS));
    }
  };

  return (
    <Modal title={`Carico o scarico: ${product.name}`} onClose={onClose}>
      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <Alert type="danger">{serverError}</Alert>
        <p>
          Quantità disponibile: <strong>{product.stock_quantity}</strong> (soglia minima {product.min_stock_threshold})
        </p>

        <fieldset className="mb-4">
          <legend className="h6">Tipo di movimento</legend>
          {[
            ["in", "Carico"],
            ["out", "Scarico"],
          ].map(([value, label]) => (
            <div className="form-check form-check-inline" key={value}>
              <input className="form-check-input" type="radio" id={`type-${value}`} value={value} {...register("type")} />
              <label className="form-check-label" htmlFor={`type-${value}`}>
                {label}
              </label>
            </div>
          ))}
          {errors.type && <div className="invalid-feedback d-block">{errors.type.message}</div>}
        </fieldset>

        <TextField
          id="quantity"
          label="Quantità"
          type="number"
          min={1}
          required
          registration={register("quantity")}
          error={errors.quantity?.message}
        />
        <TextareaField id="note" label="Nota" rows={2} registration={register("note")} error={errors.note?.message} />

        <div className="d-flex justify-content-end gap-2">
          <button type="button" className="btn btn-outline-secondary" onClick={onClose} disabled={isSubmitting}>
            Annulla
          </button>
          <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
            {isSubmitting ? "Registrazione…" : "Registra movimento"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
