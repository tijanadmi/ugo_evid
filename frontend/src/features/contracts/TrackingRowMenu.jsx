import RowMenu from "../../ui/RowMenu";
import { formatDate } from "../../utils/contractFormatting";

export default function TrackingRowMenu({
  row,
  kind,
  canWrite,
  busy,
  onEdit,
  onDelete,
  onDocuments,
}) {
  const isPeriod = kind === "periodi";
  const canDelete = isPeriod
    ? row.status === "PLANIRAN"
    : row.status !== "ZAVRSEN";
  const label = isPeriod
    ? `Akcije za period ${formatDate(row.period_od)} - ${formatDate(row.period_do)}`
    : `Akcije za zahtev ${row.broj_zahteva}`;
  const actions = [];

  if (canWrite) {
    actions.push({
      label: isPeriod ? "Izmeni period" : "Izmeni",
      icon: "edit",
      onClick: onEdit,
    });
    if (canDelete) {
      actions.push({
        label: isPeriod ? "Obriši period" : "Obriši",
        icon: "trash",
        danger: true,
        onClick: onDelete,
      });
    }
    actions.push({
      label: "Dodaj dokument",
      icon: "upload",
      onClick: () => onDocuments(true),
    });
  }

  // Reading documents remains available for closed contracts too.
  actions.push({
    label: "Prikaži dokumenta",
    icon: "eye",
    onClick: () => onDocuments(false),
  });

  return <RowMenu label={label} actions={actions} disabled={busy} />;
}
