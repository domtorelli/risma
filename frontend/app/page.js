import { redirect } from "next/navigation";

// La radice porta alla dashboard; proxy.js manda al login chi non è autenticato.
export default function Home() {
  redirect("/dashboard");
}
