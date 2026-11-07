import toast from "react-hot-toast";
import { useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";
import {
  prepareContract,
  registerContract,
} from "../../services/apiContractRegistration";

export default function ContractRegistrationForm({
  contract,
  organizations,
  onBack,
  onCreated,
  onBusy,
}) {
  const { api, user } = useAuth();
  const query = useQuery({
    queryKey: ["contract-preparation", user.username, contract.id],
    queryFn: ({ signal }) => prepareContract(api, contract.id, signal),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: false,
  });
  const types = useQuery({
    queryKey: ["contract-types", user.username],
    queryFn: ({ signal }) => api("/ugo_tip_ugovora", { signal }),
  });
  const [typeID, setTypeID] = useState("");
  const [draft, setDraft] = useState({
    ime: "",
    radno_mesto: "",
    telefon: "",
    email: "",
  });
  const [saving, setSaving] = useState(false);
  const [mustReload, setMustReload] = useState(false);
  const submitting = useRef(false);
  const data = query.data;
  const person = data?.contact;
  const org = organizations.find((item) => item.id === data?.id_ugo_org);
  async function save(event) {
    event.preventDefault();
    if (
      !typeID ||
      !data ||
      query.isFetching ||
      mustReload ||
      submitting.current
    )
      return;
    submitting.current = true;
    setSaving(true);
    onBusy(true);
    try {
      const response = await registerContract(api, {
        id_sap_ugovor: contract.id,
        id_tip_ugovora: Number(typeID),
        ...(person
          ? { id_ugo_dob_lica: person.id, contact_version: person.version }
          : { novo_lice: draft }),
      });
      onCreated(response.data);
    } catch (failure) {
      toast.error(failure.message, { id: "operation-error" });
      // A lost response can hide a committed insert. Recheck before another write.
      setMustReload(true);
    } finally {
      submitting.current = false;
      setSaving(false);
      onBusy(false);
    }
  }
  async function reload() {
    const result = await query.refetch();
    if (!result.isError) {
      setMustReload(false);
    }
  }
  return (
    <form
      className="contact-editor contract-registration-form"
      aria-label="Unos ugovora"
      onSubmit={save}
    >
      <div>
        <h3>Ugovor {contract.br_ugovor}</h3>
        <p>{contract.predmet_ugovora}</p>
        <p className="muted">{contract.dobavljac}</p>
      </div>
      <label>
        Tip ugovora
        <select
          aria-label="Tip ugovora"
          required
          value={typeID}
          disabled={saving || types.isPending}
          onChange={(event) => setTypeID(event.target.value)}
        >
          <option value="">Izaberite tip ugovora</option>
          {types.data?.items?.map((type) => (
            <option key={type.id} value={type.id}>
              {type.naziv}
            </option>
          ))}
        </select>
      </label>
      {types.error && (
        <button
          type="button"
          className="button"
          onClick={() => types.refetch()}
        >
          Ponovo učitaj tipove ugovora
        </button>
      )}
      {query.isFetching && <p role="status">Provera ugovora i kontakt lica…</p>}
      {data && !query.error && (
        <>
          <p>
            Evidencija se dodaje za organizacionu jedinicu:{" "}
            <strong>
              {org ? `${org.sifra} — ${org.naziv}` : data.id_ugo_org}
            </strong>
            .
          </p>
          {data.contract.otvoren_ug !== "X" && (
            <p className="muted">
              Izabrani ugovor je zatvoren i biće prikazan među zatvorenim
              ugovorima.
            </p>
          )}
          {person ? (
            <section
              className="registration-contact"
              aria-label="Postojeće kontakt lice"
            >
              <h3>Aktivni Service Level Manager</h3>
              <strong>{person.ime}</strong>
              <span>{person.radno_mesto}</span>
              <span>{person.telefon || "Telefon nije upisan"}</span>
              <span>{person.email || "Email nije upisan"}</span>
              <p className="muted">
                Ovi podaci biće preuzeti u evidenciju ugovora.
              </p>
            </section>
          ) : (
            <>
              <p>
                Dobavljač nema aktivnog Service Level Manager-a. Unesite
                kontakt; lice i evidencija ugovora biće sačuvani zajedno.
              </p>
              <fieldset disabled={saving || query.isFetching}>
                <label>
                  Ime i prezime
                  <input
                    aria-label="Ime i prezime"
                    required
                    maxLength={100}
                    value={draft.ime}
                    onChange={(e) =>
                      setDraft({ ...draft, ime: e.target.value })
                    }
                  />
                </label>
                <label>
                  Radno mesto (opciono)
                  <input
                    aria-label="Radno mesto (opciono)"
                    maxLength={200}
                    value={draft.radno_mesto}
                    onChange={(e) =>
                      setDraft({ ...draft, radno_mesto: e.target.value })
                    }
                  />
                </label>
                <label>
                  Telefon
                  <input
                    aria-label="Telefon"
                    required
                    type="tel"
                    maxLength={100}
                    value={draft.telefon}
                    onChange={(e) =>
                      setDraft({ ...draft, telefon: e.target.value })
                    }
                  />
                </label>
                <label>
                  Email
                  <input
                    aria-label="Email"
                    required
                    type="email"
                    maxLength={100}
                    value={draft.email}
                    onChange={(e) =>
                      setDraft({ ...draft, email: e.target.value })
                    }
                  />
                </label>
              </fieldset>
            </>
          )}
        </>
      )}
      {(query.error || mustReload) && (
        <button
          type="button"
          className="button"
          disabled={saving || query.isFetching}
          onClick={reload}
        >
          Ponovo proveri ugovor
        </button>
      )}
      <div className="contact-actions">
        <button
          type="button"
          className="button"
          disabled={saving}
          onClick={onBack}
        >
          Nazad na izbor
        </button>
        <button
          type="submit"
          className="button primary"
          disabled={
            !typeID ||
            types.isPending ||
            !!types.error ||
            !data ||
            !!query.error ||
            query.isFetching ||
            saving ||
            mustReload
          }
        >
          {saving ? "Čuvanje…" : "Evidentiraj ugovor"}
        </button>
      </div>
    </form>
  );
}
