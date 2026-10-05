"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";

import Alert from "@/components/Alert";
import { TextField } from "@/components/FormFields";
import PasswordField from "@/components/PasswordField";
import { api } from "@/lib/api";
import { safeNextPath } from "@/lib/navigation";
import { loginSchema } from "@/lib/validation";

export default function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (values) => {
    setServerError("");
    try {
      await api.post("/api/auth/login", values);
      router.replace(safeNextPath(searchParams.get("next")));
    } catch (error) {
      setServerError(error.message);
    }
  };

  return (
    <>
      <h1 className="h2 mb-4">Accedi a Risma</h1>

      {searchParams.get("reset") === "1" && (
        <Alert type="success">Password impostata: ora puoi accedere con la nuova password.</Alert>
      )}
      <Alert type="danger">{serverError}</Alert>

      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <TextField
          id="username"
          label="Username"
          autoComplete="username"
          required
          registration={register("username")}
          error={errors.username?.message}
        />
        <PasswordField
          id="password"
          label="Password"
          autoComplete="current-password"
          registration={register("password")}
          error={errors.password?.message}
        />
        <button type="submit" className="btn btn-primary w-100" disabled={isSubmitting}>
          {isSubmitting ? "Accesso in corso…" : "Accedi"}
        </button>
      </form>

      <p className="mt-4">
        <Link href="/recupera-password">Hai dimenticato la password?</Link>
      </p>
    </>
  );
}
