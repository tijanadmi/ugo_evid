import { formatDate } from "../../utils/contractFormatting";
import TrackingRowMenu from "./TrackingRowMenu";
import { trackingLabels as labels } from "./trackingConstants";

export default function DevelopmentRequestsTable({
  requests,
  workTypes,
  canWrite,
  busy,
  open,
  onDocuments,
}) {
  return (
    <div className="tracking-table">
      <table>
        <thead>
          <tr>
            <th>Radnja</th>
            <th>Zahtev / datum</th>
            <th>Vrsta rada</th>
            <th>Časovi / radni dani</th>
            <th>Potpisati pre</th>
            <th>Realizovan</th>
            <th>Zapisnik potpisan</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((p) => (
            <tr key={p.id}>
              <td>
                <TrackingRowMenu
                  row={p}
                  kind="zahtevi"
                  canWrite={canWrite}
                  busy={busy}
                  onEdit={() => open("zahtevi", p)}
                  onDelete={() => open("zahtevi", p, false, true)}
                  onDocuments={(upload) => onDocuments({ row: p, upload })}
                />
              </td>
              <td>
                {p.broj_zahteva}
                <br />
                {formatDate(p.datum_zahteva)}
              </td>
              <td>{workTypes.find((w) => w.id === p.id_vrsta_rada)?.naziv}</td>
              <td>
                {p.broj_casova} / {p.rok_radnih_dana}
              </td>
              <td>{formatDate(p.rok)}</td>
              <td>{formatDate(p.datum_realizacije)}</td>
              <td>{formatDate(p.datum_potpisa)}</td>
              <td>{labels[p.status]}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
