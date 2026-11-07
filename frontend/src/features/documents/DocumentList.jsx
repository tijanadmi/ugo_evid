import RowMenu from "../../ui/RowMenu";
import { value } from "../../utils/contractFormatting";

export default function DocumentList({
  items,
  busy,
  canWrite,
  content,
  onEdit,
}) {
  return (
    <ul className="document-list">
      {items.map((doc) => (
        <li key={doc.id}>
          <div>
            <strong>{doc.naziv}</strong>
            <span>
              {doc.tip_naziv}
              {doc.broj_dokumenta && ` · ${doc.broj_dokumenta}`}
              {doc.datpri && ` · Datum unosa: ${value(doc, "datpri")}`}
            </span>
            <small>
              {doc.originalni_naziv} · {Math.ceil(doc.velicina_bajt / 1024)} KB
            </small>
            {doc.napomena && <p>{doc.napomena}</p>}
          </div>
          <div className="document-buttons">
            {["application/pdf", "image/jpeg", "image/png"].includes(
              doc.mime_tip,
            ) && (
              <button
                className="button"
                disabled={busy}
                onClick={() => content(doc, true)}
              >
                Pregled
              </button>
            )}
            <button
              className="button"
              disabled={busy}
              onClick={() => content(doc, false)}
            >
              Preuzmi
            </button>
            {canWrite && (
              <RowMenu
                label={`Akcije za dokument ${doc.naziv}`}
                actions={[
                  {
                    label: "Izmeni",
                    icon: "edit",
                    onClick: () => onEdit(doc, "edit"),
                  },
                  {
                    label: "Obriši",
                    icon: "trash",
                    danger: true,
                    onClick: () => onEdit(doc, "delete"),
                  },
                ]}
              />
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}
