import Brand from "@/components/Brand";

// Layout delle pagine pubbliche (accesso e recupero password): intestazione
// essenziale e contenuto centrato in una card.
export default function AuthLayout({ children }) {
  return (
    <>
      <header className="it-header-wrapper">
        <div className="it-header-center-wrapper it-small-header">
          <div className="container">
            <div className="it-header-center-content-wrapper">
              <Brand href="/login" />
            </div>
          </div>
        </div>
      </header>
      <main id="contenuto" className="container auth-container my-5">
        <div className="card card-bg no-after">
          <div className="card-body p-md-5">{children}</div>
        </div>
      </main>
    </>
  );
}
