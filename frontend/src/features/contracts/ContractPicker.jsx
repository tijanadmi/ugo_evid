import ContractRegistrationForm from "./ContractRegistrationForm";
import ConfirmAction from "../../ui/ConfirmAction";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";
import Icon from "../../ui/Icon";
import { getUnregisteredContracts } from "../../services/apiContractRegistration";

export default function ContractPicker({ organizations, onClose, onCreated }) {
  const { api, user } = useAuth();
  const dialog = useRef(null);
  const [draftFilter, setDraftFilter] = useState("");
  const [filter, setFilter] = useState("");
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState(null);
  const [confirmClose, setConfirmClose] = useState(false);
  const [busy, setBusy] = useState(false);
  const query = useQuery({
    queryKey: ["unregistered-contracts", user.username, page, filter],
    queryFn: ({ signal }) =>
      getUnregisteredContracts(api, page, filter, signal),
    enabled: !selected,
    retry: false,
  });
  useEffect(() => {
    const element = dialog.current;
    element.showModal();
    return () => element.close();
  }, []);
  function close() {
    if (busy) return;
    if (selected) {
      setConfirmClose(true);
      return;
    }
    onClose();
  }
  const pages = Math.max(1, Math.ceil((query.data?.total || 0) / 20));
  return (
    <dialog
      ref={dialog}
      className={`detail-dialog contract-picker${selected ? " is-registering" : ""}`}
      aria-labelledby="contract-picker-title"
      onCancel={(event) => {
        event.preventDefault();
        close();
      }}
      onClick={(event) => {
        if (event.target === dialog.current) close();
      }}
    >
      {confirmClose && (
        <ConfirmAction
          modal
          title="Odustati od unosa ugovora?"
          onCancel={() => setConfirmClose(false)}
          onConfirm={onClose}
        >
          <p>Nesačuvani podaci biće izgubljeni.</p>
        </ConfirmAction>
      )}
      <div className="detail-header">
        <h2 id="contract-picker-title">Izaberi ugovor</h2>
        <button
          type="button"
          className="icon-button"
          disabled={busy}
          aria-label="Zatvori izbor ugovora"
          onClick={close}
        >
          <Icon name="close" />
        </button>
      </div>
      {selected ? (
        <ContractRegistrationForm
          key={selected.id}
          contract={selected}
          organizations={organizations}
          onBusy={setBusy}
          onCreated={onCreated}
          onBack={() => {
            setSelected(null);
            query.refetch();
          }}
        />
      ) : (
        <div className="contract-picker-body">
          <form
            className="partner-filter"
            onSubmit={(event) => {
              event.preventDefault();
              setFilter(draftFilter.trim());
              setPage(1);
            }}
          >
            <label htmlFor="contract-picker-filter">
              Broj ugovora, predmet ili dobavljač
            </label>
            <div className="partner-filter-controls">
              <input
                id="contract-picker-filter"
                type="search"
                maxLength={200}
                value={draftFilter}
                onChange={(event) => setDraftFilter(event.target.value)}
                placeholder="Pretraži SAP ugovore…"
              />
              <button className="button" type="submit">
                Pretraži
              </button>
            </div>
          </form>
          <p className="muted">
            Prikazani su otvoreni ugovori nosioca nabavke vaše organizacione
            jedinice koji još nisu evidentirani.
          </p>
          {query.isFetching ? (
            <p role="status">Učitavanje ugovora…</p>
          ) : query.error ? (
            <div className="contact-error">
              Pregled nije dostupan.{" "}
              <button className="button" onClick={() => query.refetch()}>
                Pokušaj ponovo
              </button>
            </div>
          ) : query.data?.items?.length ? (
            <div
              className="table-scroll contract-picker-scroll"
              tabIndex={0}
              role="region"
              aria-label="Ugovori za izbor"
            >
              <table className="contract-picker-table">
                <colgroup>
                  <col className="picker-action-column" />
                  <col className="picker-number-column" />
                  <col className="picker-subject-column" />
                  <col className="picker-supplier-column" />
                  <col className="picker-status-column" />
                </colgroup>
                <thead>
                  <tr>
                    <th scope="col">Izbor</th>
                    <th>Broj / godina</th>
                    <th>Predmet</th>
                    <th>Dobavljač</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {query.data.items.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <button
                          type="button"
                          className="button"
                          aria-label={`Izaberi ugovor ${item.br_ugovor}`}
                          onClick={() => setSelected(item)}
                        >
                          Izaberi
                        </button>
                      </td>
                      <th scope="row">
                        {item.br_ugovor}
                        <small>{item.godina}</small>
                      </th>
                      <td>{item.predmet_ugovora}</td>
                      <td>{item.dobavljac}</td>
                      <td>
                        {item.otvoren_ug === "X" ? "Otvoren" : "Zatvoren"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p>Nema neevidentiranih ugovora za izabrani filter.</p>
          )}
          <nav className="pagination" aria-label="Stranice izbora ugovora">
            <span>
              Stranica {page} od {pages}
            </span>
            <button
              type="button"
              className="button"
              disabled={page <= 1 || query.isFetching}
              onClick={() => setPage((value) => value - 1)}
            >
              Prethodna
            </button>
            <button
              type="button"
              className="button"
              disabled={page >= pages || query.isFetching || !!query.error}
              onClick={() => setPage((value) => value + 1)}
            >
              Sledeća
            </button>
          </nav>
        </div>
      )}
    </dialog>
  );
}
