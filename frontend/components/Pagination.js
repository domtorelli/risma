import Icon from "./Icon";

// Paginazione di Bootstrap Italia: pagina precedente/successiva e indicatore
// "Pagina X di Y". page parte da 1.
export default function Pagination({ page, pageSize, total, onChange, label = "Paginazione" }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));

  if (pages <= 1) {
    return null;
  }

  return (
    <nav className="pagination-wrapper justify-content-center" aria-label={label}>
      <ul className="pagination">
        <li className={`page-item ${page <= 1 ? "disabled" : ""}`}>
          <button type="button" className="page-link" onClick={() => onChange(page - 1)} disabled={page <= 1}>
            <Icon name="it-chevron-left" color="primary" />
            <span className="visually-hidden">Pagina precedente</span>
          </button>
        </li>
        <li className="page-item">
          <span className="page-link" aria-current="page">
            Pagina {page} di {pages}
          </span>
        </li>
        <li className={`page-item ${page >= pages ? "disabled" : ""}`}>
          <button type="button" className="page-link" onClick={() => onChange(page + 1)} disabled={page >= pages}>
            <span className="visually-hidden">Pagina successiva</span>
            <Icon name="it-chevron-right" color="primary" />
          </button>
        </li>
      </ul>
    </nav>
  );
}
