import AppHeader from "@/components/AppHeader";
import { AuthProvider } from "@/components/AuthProvider";

// Layout dell'area riservata: tutte le pagine sono dentro AuthProvider, che
// verifica la sessione prima di mostrarle.
export default function AppLayout({ children }) {
  return (
    <AuthProvider>
      <AppHeader />
      <main id="contenuto" className="container pt-4 pb-5">
        {children}
      </main>
    </AuthProvider>
  );
}
