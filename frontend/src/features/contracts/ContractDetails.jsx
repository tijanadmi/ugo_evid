import Icon from "../../ui/Icon";
import { value } from "../../utils/contractFormatting";
import ResponsiblePersons from "./ResponsiblePersons";
import SupplierAddress from "./SupplierAddress";
import ContractPeople from "./ContractPeople";

const fields = [
  ["godina", "Godina"],
  ["jn", "Broj nabavke"],
  ["br_poz_plana", "Pozicija plana"],
  ["ugovor_dms", "DMS broj"],
  ["pocetak_ug", "Važi od"],
  ["kraj_ug", "Važi do"],
];

export default function ContractDetails({
  item,
  onUploadResolution,
  onViewResolutions,
}) {
  return (
    <div className="contract-summary-grid">
      <section
        className="contract-card contract-facts"
        aria-labelledby="contract-facts-title"
      >
        <h2 id="contract-facts-title">O ugovoru</h2>
        <dl>
          {fields.map(([key, label]) => (
            <div key={key}>
              <dt>{label}</dt>
              <dd>{value(item, key)}</dd>
            </div>
          ))}
          <div className="commercial-contact">
            <dt>Zaduženi komercijalista</dt>
            <dd>
              {value(item, "naz_komerc")}
              {item.m_br_komerc && (
                <span className="commercial-code"> · {item.m_br_komerc}</span>
              )}
            </dd>
          </div>
        </dl>
      </section>
      <section
        className="contract-card operational-contacts"
        aria-labelledby="operational-title"
      >
        <h2 id="operational-title">
          Lica za operativno praćenje ugovora
          {onUploadResolution && (
            <button
              className="icon-button"
              aria-label="Dodaj rešenje"
              title="Dodaj rešenje"
              onClick={onUploadResolution}
            >
              <Icon name="upload" />
            </button>
          )}
          {onViewResolutions && (
            <button
              type="button"
              className="icon-button"
              aria-label="Prikaži rešenja za operativno praćenje"
              title="Prikaži rešenja za operativno praćenje"
              onClick={onViewResolutions}
            >
              <Icon name="eye" />
            </button>
          )}
        </h2>
        <ResponsiblePersons item={item} />
      </section>
      <section
        className="contract-card partner-card"
        aria-labelledby="partner-title"
      >
        <div className="partner-heading">
          <div>
            <h2 id="partner-title">Podaci o partneru / dobavljaču</h2>
            <p className="partner-name">{value(item, "naziv")}</p>
            <SupplierAddress item={item} />
          </div>
        </div>
        <ContractPeople item={item} />
      </section>
    </div>
  );
}
