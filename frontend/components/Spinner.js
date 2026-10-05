// Indicatore di caricamento con testo per i lettori di schermo.
export default function Spinner({ label = "Caricamento in corso" }) {
  return (
    <div className="d-flex justify-content-center my-5">
      <div className="progress-spinner progress-spinner-active" role="status">
        <span className="visually-hidden">{label}</span>
      </div>
    </div>
  );
}
