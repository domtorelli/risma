import { Suspense } from "react";

import LoginForm from "./LoginForm";

export const metadata = { title: "Accesso" };

// LoginForm legge i parametri dell'URL (useSearchParams), che richiedono un
// confine Suspense per non rendere dinamica l'intera pagina.
export default function LoginPage() {
  return (
    <Suspense>
      <LoginForm />
    </Suspense>
  );
}
