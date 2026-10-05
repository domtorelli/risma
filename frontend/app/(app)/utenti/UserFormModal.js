"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import { SelectField, TextField } from "@/components/FormFields";
import Modal from "@/components/Modal";
import { api } from "@/lib/api";
import { applyServerErrors } from "@/lib/forms";
import { roleLabels } from "@/lib/format";
import { userSchema } from "@/lib/validation";

const FIELDS = ["username", "email", "first_name", "last_name", "date_of_birth", "role_id"];

// Form di creazione (user assente) o modifica di un utente. La password non
// c'è: il nuovo utente la sceglie con il link dell'email di benvenuto. Il
// ruolo si può scegliere solo se chi usa il form è Admin (scheda, §4.3).
export default function UserFormModal({ user, roles, isAdmin, onSaved, onClose }) {
  const editing = Boolean(user);
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(userSchema),
    defaultValues: {
      username: user?.username ?? "",
      email: user?.email ?? "",
      first_name: user?.first_name ?? "",
      last_name: user?.last_name ?? "",
      date_of_birth: user?.date_of_birth ?? "",
      role_id: user?.role_id ?? "",
    },
  });

  const onSubmit = async ({ role_id: roleID, ...profile }) => {
    setServerError("");
    try {
      if (!editing) {
        const created = await api.post("/api/users", isAdmin && roleID ? { ...profile, role_id: roleID } : profile);
        onSaved(created, "created");
        return;
      }

      let saved = await api.put(`/api/users/${user.id}`, profile);
      if (isAdmin && roleID && roleID !== user.role_id) {
        saved = await api.put(`/api/users/${user.id}/role`, { role_id: roleID });
      }
      onSaved(saved, "updated");
    } catch (error) {
      setServerError(applyServerErrors(error, setError, FIELDS));
    }
  };

  return (
    <Modal title={editing ? `Modifica ${user.username}` : "Nuovo utente"} onClose={onClose} size="lg">
      <form onSubmit={handleSubmit(onSubmit)} noValidate id="user-form">
        <Alert type="danger">{serverError}</Alert>
        {!editing && (
          <Alert type="info">L&apos;utente riceverà un&apos;email con il link per scegliere la password (valido 24 ore).</Alert>
        )}

        <div className="row">
          <div className="col-md-6">
            <TextField id="first_name" label="Nome" required registration={register("first_name")} error={errors.first_name?.message} />
          </div>
          <div className="col-md-6">
            <TextField id="last_name" label="Cognome" required registration={register("last_name")} error={errors.last_name?.message} />
          </div>
          <div className="col-md-6">
            <TextField id="username" label="Username" required autoComplete="off" registration={register("username")} error={errors.username?.message} />
          </div>
          <div className="col-md-6">
            <TextField id="email" label="Email" type="email" required autoComplete="off" registration={register("email")} error={errors.email?.message} />
          </div>
          <div className="col-md-6">
            <TextField
              id="date_of_birth"
              label="Data di nascita"
              type="date"
              required
              registration={register("date_of_birth")}
              error={errors.date_of_birth?.message}
            />
          </div>
          {isAdmin && (
            <div className="col-md-6">
              <SelectField
                id="role_id"
                label="Ruolo"
                placeholder={editing ? undefined : "Operatore (predefinito)"}
                options={roles.map((r) => ({ value: r.id, label: roleLabels[r.name] ?? r.name }))}
                registration={register("role_id")}
                error={errors.role_id?.message}
              />
            </div>
          )}
        </div>

        <div className="d-flex justify-content-end gap-2 mt-2">
          <button type="button" className="btn btn-outline-secondary" onClick={onClose} disabled={isSubmitting}>
            Annulla
          </button>
          <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
            {isSubmitting ? "Salvataggio…" : editing ? "Salva modifiche" : "Crea utente"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
