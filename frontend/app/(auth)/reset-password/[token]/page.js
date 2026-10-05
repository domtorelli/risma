import ResetPasswordForm from "./ResetPasswordForm";

// Il token è nell'URL: la pagina non invia il proprio indirizzo come
// Referer a eventuali risorse esterne.
export const metadata = { title: "Imposta la password", referrer: "no-referrer" };

// Da Next.js 15 params è una Promise.
export default async function ResetPasswordPage({ params }) {
  const { token } = await params;
  return <ResetPasswordForm token={token} />;
}
