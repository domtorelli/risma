"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import PasswordField from "@/components/PasswordField";
import { api } from "@/lib/api";
import { resetPasswordSchema } from "@/lib/validation";

const PASSWORD_HINT = "Almeno 8 caratteri, con una lettera maiuscola, un numero e un carattere speciale.";

// Serve sia al recupero password sia al primo accesso di un nuovo utente
// (link dell'email di benvenuto).
export default function ResetPasswordForm({ token }) {
  const router = useRouter();
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(resetPasswordSchema) });

  const onSubmit = async ({ password }) => {
    setServerError("");
    try {
      await api.post("/api/auth/reset-password", { token, password });
      router.replace("/login?reset=1");
    } catch (error) {
      if (error.fields?.password) {
        setError("password", { message: error.fields.password });
      } else {
        setServerError(error.message);
      }
    }
  };

  return (
    <>
      <h1 className="h2 mb-3">Imposta la password</h1>
      <p>Scegli la password con cui accederai a Risma.</p>

      <Alert type="danger">
        {serverError && (
          <>
            {serverError} <Link href="/recupera-password">Richiedi un nuovo link</Link>.
          </>
        )}
      </Alert>

      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <PasswordField
          id="password"
          label="Nuova password"
          autoComplete="new-password"
          hint={PASSWORD_HINT}
          registration={register("password")}
          error={errors.password?.message}
        />
        <PasswordField
          id="confirm"
          label="Ripeti la password"
          autoComplete="new-password"
          registration={register("confirm")}
          error={errors.confirm?.message}
        />
        <button type="submit" className="btn btn-primary w-100" disabled={isSubmitting}>
          {isSubmitting ? "Salvataggio…" : "Imposta la password"}
        </button>
      </form>
    </>
  );
}
