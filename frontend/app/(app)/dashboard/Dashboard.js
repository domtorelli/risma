"use client";

import Alert from "@/components/Alert";
import Icon from "@/components/Icon";
import ProductsByTypeChart from "@/components/ProductsByTypeChart";
import Spinner from "@/components/Spinner";
import { formatDateTime, formatEuro, movementLabels } from "@/lib/format";
import { useApiData } from "@/lib/useApiData";

function StatCard({ icon, label, value }) {
  return (
    <div className="col-12 col-md-4">
      <div className="card card-bg h-100">
        <div className="card-body">
          <div className="d-flex align-items-center">
            <Icon name={icon} size="lg" color="primary" className="me-3" />
            <div>
              <p className="mb-1 text-secondary">{label}</p>
              <p className="h3 mb-0">{value}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function Dashboard() {
  const { data, error, loading } = useApiData("/api/dashboard");

  if (loading && !data) {
    return <Spinner />;
  }

  if (error) {
    return <Alert type="danger">{error}</Alert>;
  }

  return (
    <>
      <h1 className="mb-4">Dashboard</h1>

      <div className="row g-4 mb-4">
        <StatCard icon="it-user" label="Utenti attivi" value={data.total_users} />
        <StatCard icon="it-box" label="Prodotti a catalogo" value={data.total_products} />
        <StatCard icon="it-chart-line" label="Valore dell'inventario" value={formatEuro(data.inventory_value)} />
      </div>

      <div className="row g-4">
        <section className="col-12 col-lg-5" aria-labelledby="chart-title">
          <div className="card card-bg h-100">
            <div className="card-body">
              <h2 className="h4 card-title" id="chart-title">
                Prodotti per categoria
              </h2>
              <ProductsByTypeChart data={data.products_by_type} />
            </div>
          </div>
        </section>

        <section className="col-12 col-lg-7" aria-labelledby="movements-title">
          <div className="card card-bg h-100">
            <div className="card-body">
              <h2 className="h4 card-title" id="movements-title">
                Ultimi movimenti di magazzino
              </h2>
              {data.recent_movements.length === 0 ? (
                <p>Nessun movimento registrato.</p>
              ) : (
                <div className="table-responsive">
                  <table className="table">
                    <thead>
                      <tr>
                        <th scope="col">Data</th>
                        <th scope="col">Prodotto</th>
                        <th scope="col">Tipo</th>
                        <th scope="col" className="text-end">
                          Quantità
                        </th>
                        <th scope="col">Utente</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.recent_movements.map((m) => (
                        <tr key={m.id}>
                          <td>{formatDateTime(m.created_at)}</td>
                          <td>
                            {m.product_name}
                            {m.product_deleted && <span className="badge bg-secondary ms-2">eliminato</span>}
                            {m.note && <div className="small text-muted">{m.note}</div>}
                          </td>
                          <td>
                            <span className={`badge ${m.type === "in" ? "bg-success" : "bg-warning text-dark"}`}>
                              {movementLabels[m.type]}
                            </span>
                          </td>
                          <td className="text-end">{m.quantity}</td>
                          <td>{m.username}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </section>
      </div>
    </>
  );
}
