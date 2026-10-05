"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import { SelectField, TextareaField, TextField } from "@/components/FormFields";
import Modal from "@/components/Modal";
import { api } from "@/lib/api";
import { applyServerErrors } from "@/lib/forms";
import { productSchema } from "@/lib/validation";

const FIELDS = ["product_type_id", "name", "description", "price", "min_stock_threshold", "initial_quantity"];

// Form di creazione (product assente) o modifica di un prodotto. La
// giacenza si indica solo alla creazione (diventa il carico iniziale): poi
// cambia esclusivamente con carichi e scarichi.
export default function ProductFormModal({ product, types, onSaved, onClose }) {
  const editing = Boolean(product);
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(productSchema),
    defaultValues: {
      product_type_id: product?.product_type_id ?? "",
      name: product?.name ?? "",
      description: product?.description ?? "",
      price: product?.price ?? "",
      min_stock_threshold: product?.min_stock_threshold ?? 0,
      initial_quantity: editing ? undefined : 0,
    },
  });

  const onSubmit = async ({ initial_quantity: initialQuantity, ...details }) => {
    setServerError("");
    try {
      const saved = editing
        ? await api.put(`/api/products/${product.id}`, details)
        : await api.post("/api/products", { ...details, initial_quantity: initialQuantity });
      onSaved(saved, editing ? "updated" : "created");
    } catch (error) {
      setServerError(applyServerErrors(error, setError, FIELDS));
    }
  };

  return (
    <Modal title={editing ? `Modifica ${product.name}` : "Nuovo prodotto"} onClose={onClose} size="lg">
      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <Alert type="danger">{serverError}</Alert>

        <div className="row">
          <div className="col-md-6">
            <SelectField
              id="product_type_id"
              label="Tipo"
              required
              placeholder="Scegli il tipo"
              options={types.map((t) => ({ value: t.id, label: t.name }))}
              registration={register("product_type_id")}
              error={errors.product_type_id?.message}
            />
          </div>
          <div className="col-md-6">
            <TextField id="name" label="Nome" required registration={register("name")} error={errors.name?.message} />
          </div>
          <div className="col-12">
            <TextareaField id="description" label="Descrizione" registration={register("description")} error={errors.description?.message} />
          </div>
          <div className="col-md-4">
            <TextField
              id="price"
              label="Prezzo unitario (€)"
              required
              inputMode="decimal"
              hint="Usa il punto per i decimali, es. 12.50"
              registration={register("price")}
              error={errors.price?.message}
            />
          </div>
          <div className="col-md-4">
            <TextField
              id="min_stock_threshold"
              label="Soglia minima"
              type="number"
              min={0}
              required
              registration={register("min_stock_threshold")}
              error={errors.min_stock_threshold?.message}
            />
          </div>
          {!editing && (
            <div className="col-md-4">
              <TextField
                id="initial_quantity"
                label="Quantità iniziale"
                type="number"
                min={0}
                required
                hint="Registrata come carico iniziale"
                registration={register("initial_quantity")}
                error={errors.initial_quantity?.message}
              />
            </div>
          )}
        </div>

        <div className="d-flex justify-content-end gap-2 mt-2">
          <button type="button" className="btn btn-outline-secondary" onClick={onClose} disabled={isSubmitting}>
            Annulla
          </button>
          <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
            {isSubmitting ? "Salvataggio…" : editing ? "Salva modifiche" : "Crea prodotto"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
