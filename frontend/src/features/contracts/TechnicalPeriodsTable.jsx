import { formatDate } from "../../utils/contractFormatting";
import TrackingRowMenu from "./TrackingRowMenu";
import { trackingLabels as labels } from "./trackingConstants";

export default function TechnicalPeriodsTable({
  periods,
  canWrite,
  busy,
  open,
  onDocuments,
  onReports,
}) {
  return (
    <div className="tracking-table">
      <table>
        <thead>
          <tr>
            <th>Radnja</th>
            <th>Period</th>
            <th>Kvartal</th>
            <th>Status</th>
            <th>Rok dokumenta</th>
            <th>Potpisano</th>
          </tr>
        </thead>
        <tbody>
          {periods.map((p) => (
            <tr key={p.id}>
              <td>
                <TrackingRowMenu
                  row={p}
                  kind="periodi"
                  canWrite={canWrite}
                  busy={busy}
                  onEdit={() => open("periodi", p)}
                  onDelete={() => open("periodi", p, false, true)}
                  onDocuments={(upload) => onDocuments({ row: p, upload })}
                  onReports={onReports ? () => onReports(p) : undefined}
                />
              </td>
              <td>
                {formatDate(p.period_od)} — {formatDate(p.period_do)}
              </td>
              <td>{p.broj_kvartala || "—"}</td>
              <td>
                {p.status === "ZAVRSEN" ? (
                  <strong>{labels[p.status]}</strong>
                ) : (
                  labels[p.status]
                )}
              </td>
              <td>{formatDate(p.rok)}</td>
              <td>{formatDate(p.datum_potpisa)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
