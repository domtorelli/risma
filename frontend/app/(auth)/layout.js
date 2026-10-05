import Icon from "@/components/Icon";

// Layout delle pagine pubbliche (accesso e recupero password): intestazione
// essenziale e contenuto centrato.
export default function AuthLayout({ children }) {
  return (
    <>
      <header className="it-header-wrapper">
        <div className="it-header-center-wrapper">
          <div className="container">
            <div className="it-header-center-content-wrapper">
              <div className="it-brand-wrapper">
                <span className="d-flex align-items-center text-white">
                  <Icon name="it-pa" size="lg" color="white" className="me-2" />
                  <span className="it-brand-text">
                    <span className="it-brand-title">Risma</span>
                    <span className="it-brand-tagline d-none d-md-block">Gestionale cancelleria</span>
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </header>
      <main id="contenuto" className="container auth-container my-5">
        {children}
      </main>
    </>
  );
}
