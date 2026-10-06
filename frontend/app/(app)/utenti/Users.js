"use client";

import { useRef, useState } from "react";

import Alert from "@/components/Alert";
import { useAuth } from "@/components/AuthProvider";
import ConfirmModal from "@/components/ConfirmModal";
import Icon from "@/components/Icon";
import Pagination from "@/components/Pagination";
import Spinner from "@/components/Spinner";
import { api, query } from "@/lib/api";
import { formatDateTime, roleLabels } from "@/lib/format";
import { useApiData } from "@/lib/useApiData";
import { useDebouncedValue } from "@/lib/useDebouncedValue";

import UserFormModal from "./UserFormModal";

const PAGE_SIZE = 10;

export default function Users() {
  const { user: me, isAdmin } = useAuth();

  const [showDeactivated, setShowDeactivated] = useState(false);
  const [search, setSearch] = useState("");
  const searchRef = useRef(null);
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebouncedValue(search.trim());

  // Finestra aperta: { type: "create" | "edit" | "deactivate" | "reactivate", user }.
  const [dialog, setDialog] = useState(null);
  const [notice, setNotice] = useState("");

  const { data, error, loading, reload } = useApiData(
    `/api/users${query({ q: debouncedSearch, deactivated: showDeactivated ? "true" : "", page, page_size: PAGE_SIZE })}`,
  );
  // I ruoli servono solo al form dell'Admin.
  const roles = useApiData(isAdmin ? "/api/roles" : null);

  // Se la ricerca cambia mentre si è su una pagina successiva alla prima, il
  // risultato potrebbe avere meno pagine: si riparte dalla prima.
  const [searchedFor, setSearchedFor] = useState(debouncedSearch);
  if (searchedFor !== debouncedSearch) {
    setSearchedFor(debouncedSearch);
    setPage(1);
  }

  const switchTab = (deactivated) => {
    setShowDeactivated(deactivated);
    setPage(1);
  };

  // Il pulsante sparisce appena la ricerca è vuota: il focus passa al campo.
  const resetSearch = () => {
    setSearch("");
    searchRef.current?.focus();
  };

  const closeDialog = () => setDialog(null);

  const done = (message) => {
    setDialog(null);
    setNotice(message);
    reload();
  };

  return (
    <>
      <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-2">
        <h1 className="mb-0">Utenti</h1>
        <button type="button" className="btn btn-primary btn-icon" onClick={() => setDialog({ type: "create" })}>
          <Icon name="it-plus-circle" color="white" className="me-2" />
          <span>Nuovo utente</span>
        </button>
      </div>

      <Alert type="success">{notice}</Alert>

      <ul className="nav nav-tabs mb-3" role="tablist">
        {[
          [false, "Attivi"],
          [true, "Disattivati"],
        ].map(([deactivated, label]) => (
          <li className="nav-item" key={label} role="presentation">
            <button
              type="button"
              role="tab"
              className={`nav-link ${showDeactivated === deactivated ? "active" : ""}`}
              aria-selected={showDeactivated === deactivated}
              onClick={() => switchTab(deactivated)}
            >
              {label}
            </button>
          </li>
        ))}
      </ul>

      <div className="row">
        <div className="col-12 col-md-6 col-lg-5">
          <div className="form-group mb-4">
            <label htmlFor="user-search" className="active">
              Cerca per username o email
            </label>
            <input
              id="user-search"
              ref={searchRef}
              type="search"
              className="form-control"
              value={search}
              maxLength={100}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
        </div>
      </div>

      <Alert type="danger">{error}</Alert>

      {!data ? (
        loading && <Spinner />
      ) : (
        <>
          <div className="d-flex flex-wrap align-items-center gap-3 mb-2">
            <p className="small text-muted mb-0" aria-live="polite">
              {data.total === 1 ? "1 utente trovato" : `${data.total} utenti trovati`}
            </p>
            {search !== "" && (
              <button type="button" className="btn btn-link btn-xs p-0" onClick={resetSearch}>
                Azzera ricerca
              </button>
            )}
          </div>
          <div className="table-responsive">
            <table className="table table-hover align-middle" aria-busy={loading}>
              <caption className="visually-hidden">Utenti {showDeactivated ? "disattivati" : "attivi"}</caption>
              <thead>
                <tr>
                  <th scope="col">Nome</th>
                  <th scope="col">Username</th>
                  <th scope="col">Email</th>
                  <th scope="col">Ruolo</th>
                  <th scope="col">Stato</th>
                  <th scope="col">Ultimo accesso</th>
                  <th scope="col" className="table-actions">
                    <span className="visually-hidden">Azioni</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.users.map((u) => (
                  <tr key={u.id}>
                    <td>
                      {u.first_name} {u.last_name}
                    </td>
                    <td>{u.username}</td>
                    <td>{u.email}</td>
                    <td>{roleLabels[u.role_name] ?? u.role_name}</td>
                    <td>
                      {u.deactivated_at ? (
                        <span className="badge bg-secondary">Disattivato</span>
                      ) : u.status === "blocked" ? (
                        <span className="badge bg-danger">Bloccato</span>
                      ) : (
                        <span className="badge bg-success">Attivo</span>
                      )}
                    </td>
                    <td className="text-nowrap">{formatDateTime(u.last_login_at)}</td>
                    <td className="table-actions">
                      {u.deactivated_at ? (
                        <button type="button" className="btn btn-outline-primary btn-xs" onClick={() => setDialog({ type: "reactivate", user: u })}>
                          Riattiva<span className="visually-hidden"> {u.username}</span>
                        </button>
                      ) : (
                        <>
                          <button type="button" className="btn btn-outline-primary btn-xs" onClick={() => setDialog({ type: "edit", user: u })}>
                            Modifica<span className="visually-hidden"> {u.username}</span>
                          </button>
                          {u.id !== me.id && (
                            <button type="button" className="btn btn-outline-danger btn-xs" onClick={() => setDialog({ type: "deactivate", user: u })}>
                              Disattiva<span className="visually-hidden"> {u.username}</span>
                            </button>
                          )}
                        </>
                      )}
                    </td>
                  </tr>
                ))}
                {data.users.length === 0 && (
                  <tr>
                    <td colSpan={7} className="text-center text-muted py-4">
                      Nessun utente trovato.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          <Pagination page={data.page} pageSize={data.page_size} total={data.total} onChange={setPage} label="Pagine degli utenti" />
        </>
      )}

      {(dialog?.type === "create" || dialog?.type === "edit") && (
        <UserFormModal
          user={dialog.user}
          roles={roles.data ?? []}
          isAdmin={isAdmin}
          onClose={closeDialog}
          onSaved={(saved, action) =>
            done(action === "created" ? `Utente ${saved.username} creato: riceverà l'email di benvenuto.` : `Utente ${saved.username} aggiornato.`)
          }
        />
      )}

      {dialog?.type === "deactivate" && (
        <ConfirmModal
          title="Disattiva utente"
          confirmLabel="Disattiva"
          danger
          onClose={closeDialog}
          onConfirm={async () => {
            await api.delete(`/api/users/${dialog.user.id}`);
            done(`Utente ${dialog.user.username} disattivato.`);
          }}
        >
          <p>
            Disattivare <strong>{dialog.user.username}</strong>? Non potrà più accedere e le sue sessioni verranno chiuse.
            Potrai riattivarlo dalla scheda &laquo;Disattivati&raquo;.
          </p>
        </ConfirmModal>
      )}

      {dialog?.type === "reactivate" && (
        <ConfirmModal
          title="Riattiva utente"
          confirmLabel="Riattiva"
          onClose={closeDialog}
          onConfirm={async () => {
            await api.post(`/api/users/${dialog.user.id}/reactivate`);
            done(`Utente ${dialog.user.username} riattivato.`);
          }}
        >
          <p>
            Riattivare <strong>{dialog.user.username}</strong>? Potrà accedere di nuovo con la sua password.
          </p>
        </ConfirmModal>
      )}
    </>
  );
}
