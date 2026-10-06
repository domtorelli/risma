"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import { TextField } from "@/components/FormFields";
import { api } from "@/lib/api";
import { forgotPasswordSchema } from "@/lib/validation";

export default function ForgotPasswordForm() {
  const [sentMessage, setSentMessage] = useState("");
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(forgotPasswordSchema) });

  // Il backend risponde allo stesso modo che l'email esista o no: la pagina
  // mostra il suo messaggio senza aggiungere nulla.
  const onSubmit = async (values) => {
    setServerError("");
    try {
      const result = await api.post("/api/auth/forgot-password", values);
      setSentMessage(result.message);
    } catch (error) {
      setServerError(error.message);
    }
  };

  return (
    <>
      <h1 className="h2 mb-3">Recupera password</h1>

      {sentMessage ? (
        <Alert type="success">{sentMessage}</Alert>
      ) : (
        <>
          <p>Inserisci l&apos;email del tuo account: riceverai un link per scegliere una nuova password.</p>
          <Alert type="danger">{serverError}</Alert>
          <form onSubmit={handleSubmit(onSubmit)} noValidate>
            <TextField
              id="email"
              label="Email"
              type="email"
              autoComplete="email"
              required
              registration={register("email")}
              error={errors.email?.message}
            />
            <button type="submit" className="btn btn-primary w-100" disabled={isSubmitting}>
              {isSubmitting ? "Invio in corso…" : "Invia il link"}
            </button>
          </form>
        </>
      )}

      <p className="mt-4 mb-0 text-center">
        <Link href="/login">Torna all&apos;accesso</Link>
      </p>
    </>
  );
}
