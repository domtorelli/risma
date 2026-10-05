"use client";

import { useState } from "react";

import Alert from "@/components/Alert";
import ConfirmModal from "@/components/ConfirmModal";
import Icon from "@/components/Icon";
import Pagination from "@/components/Pagination";
import Spinner from "@/components/Spinner";
import { api, query } from "@/lib/api";
import { formatEuro, movementLabels } from "@/lib/format";
import { useApiData } from "@/lib/useApiData";
import { useDebouncedValue } from "@/lib/useDebouncedValue";

import MovementModal from "./MovementModal";
import ProductFormModal from "./ProductFormModal";

const PAGE_SIZE = 10;

// Ordinamenti offerti: valore della select → parametri sort e order dell'API.
const SORT_OPTIONS = [
  { value: "name-asc", label: "Nome (A-Z)" },
  { value: "price-asc", label: "Prezzo crescente" },
  { value: "price-desc", label: "Prezzo decrescente" },
  { value: "quantity-asc", label: "Quantità crescente" },
  { value: "quantity-desc", label: "Quantità decrescente" },
];

export default function Inventory() {
  const [search, setSearch] = useState("");
  const [typeID, setTypeID] = useState("");
  const [sortValue, setSortValue] = useState("name-asc");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebouncedValue(search.trim());

  // Finestra aperta: { type: "create" | "edit" | "movement" | "delete", product }.
  const [dialog, setDialog] = useState(null);
  const [notice, setNotice] = useState({ type: "success", text: "" });

  const [sort, order] = sortValue.split("-");
  const { data, error, loading, reload } = useApiData(
    `/api/products${query({ q: debouncedSearch, type_id: typeID, sort, order, page, page_size: PAGE_SIZE })}`,
  );
  const types = useApiData("/api/product-types");

  // Cambiando ricerca si riparte dalla prima pagina (vedi Users.js).
  const [searchedFor, setSearchedFor] = useState(debouncedSearch);
  if (searchedFor !== debouncedSearch) {
    setSearchedFor(debouncedSearch);
    setPage(1);
  }

  const closeDialog = () => setDialog(null);

  const done = (text, type = "success") => {
    setDialog(null);
    setNotice({ type, text });
    reload();
  };

  const movementDone = (product, movement) => {
    const text = `${movementLabels[movement.type]} di ${movement.quantity} registrato per ${product.name}: quantità disponibile ${product.stock_quantity}.`;
    if (product.below_threshold) {
      done(`${text} La quantità è sotto la soglia minima: gli amministratori sono stati avvisati via email.`, "warning");
    } else {
      done(text);
    }
  };

  return (
    <>
      <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-2">
        <h1 className="mb-0">Inventario</h1>
        <button type="button" className="btn btn-primary btn-icon" onClick={() => setDialog({ type: "create" })}>
          <Icon name="it-plus-circle" color="white" className="me-2" />
          <span>Nuovo prodotto</span>
        </button>
      </div>

      <Alert type={notice.type}>{notice.text}</Alert>

      <div className="row">
        <div className="col-md-5">
          <div className="form-group">
            <label htmlFor="product-search" className="active">
              Cerca per nome
            </label>
            <input
              id="product-search"
              type="search"
              className="form-control"
              value={search}
              maxLength={100}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
        </div>
        <div className="col-md-3">
          <div className="select-wrapper mb-4">
            <label htmlFor="product-type">Tipo</label>
            <select
              id="product-type"
              value={typeID}
              onChange={(event) => {
                setTypeID(event.target.value);
                setPage(1);
              }}
            >
              <option value="">Tutti i tipi</option>
              {(types.data ?? []).map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="col-md-4">
          <div className="select-wrapper mb-4">
            <label htmlFor="product-sort">Ordina per</label>
            <select
              id="product-sort"
              value={sortValue}
              onChange={(event) => {
                setSortValue(event.target.value);
                setPage(1);
              }}
            >
              {SORT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <Alert type="danger">{error}</Alert>

      {!data ? (
        loading && <Spinner />
      ) : (
        <>
          <p className="small text-muted" aria-live="polite">
            {data.total === 1 ? "1 prodotto trovato" : `${data.total} prodotti trovati`}
          </p>
          <div className="table-responsive">
            <table className="table table-hover" aria-busy={loading}>
              <caption className="visually-hidden">Prodotti in inventario</caption>
              <thead>
                <tr>
                  <th scope="col">Nome</th>
                  <th scope="col">Tipo</th>
                  <th scope="col" className="text-end">
                    Prezzo unitario
                  </th>
                  <th scope="col" className="text-end">
                    Quantità
                  </th>
                  <th scope="col" className="text-end">
                    Soglia minima
                  </th>
                  <th scope="col" className="table-actions">
                    <span className="visually-hidden">Azioni</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.products.map((p) => (
                  <tr key={p.id}>
                    <td>
                      {p.name}
                      {p.description && <div className="small text-muted">{p.description}</div>}
                    </td>
                    <td>{p.product_type_name}</td>
                    <td className="text-end">{formatEuro(p.price)}</td>
                    <td className="text-end">
                      {p.stock_quantity}
                      {p.below_threshold && <span className="badge bg-danger ms-2">sotto soglia</span>}
                    </td>
                    <td className="text-end">{p.min_stock_threshold}</td>
                    <td className="table-actions">
                      <button type="button" className="btn btn-outline-primary btn-xs me-2" onClick={() => setDialog({ type: "movement", product: p })}>
                        Carico/Scarico<span className="visually-hidden"> {p.name}</span>
                      </button>
                      <button type="button" className="btn btn-outline-primary btn-xs me-2" onClick={() => setDialog({ type: "edit", product: p })}>
                        Modifica<span className="visually-hidden"> {p.name}</span>
                      </button>
                      <button type="button" className="btn btn-outline-danger btn-xs" onClick={() => setDialog({ type: "delete", product: p })}>
                        Elimina<span className="visually-hidden"> {p.name}</span>
                      </button>
                    </td>
                  </tr>
                ))}
                {data.products.length === 0 && (
                  <tr>
                    <td colSpan={6} className="text-center">
                      Nessun prodotto trovato.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          <Pagination page={data.page} pageSize={data.page_size} total={data.total} onChange={setPage} label="Pagine dei prodotti" />
        </>
      )}

      {(dialog?.type === "create" || dialog?.type === "edit") && (
        <ProductFormModal
          product={dialog.product}
          types={types.data ?? []}
          onClose={closeDialog}
          onSaved={(saved, action) =>
            saved.below_threshold
              ? done(`Prodotto ${saved.name} salvato. La quantità è sotto la soglia minima.`, "warning")
              : done(action === "created" ? `Prodotto ${saved.name} creato.` : `Prodotto ${saved.name} aggiornato.`)
          }
        />
      )}

      {dialog?.type === "movement" && <MovementModal product={dialog.product} onClose={closeDialog} onSaved={movementDone} />}

      {dialog?.type === "delete" && (
        <ConfirmModal
          title="Elimina prodotto"
          confirmLabel="Elimina"
          danger
          onClose={closeDialog}
          onConfirm={async () => {
            await api.delete(`/api/products/${dialog.product.id}`);
            done(`Prodotto ${dialog.product.name} eliminato.`);
          }}
        >
          <p>
            Eliminare <strong>{dialog.product.name}</strong>? Il prodotto non comparirà più a catalogo; i suoi movimenti restano
            nello storico.
          </p>
        </ConfirmModal>
      )}
    </>
  );
}
