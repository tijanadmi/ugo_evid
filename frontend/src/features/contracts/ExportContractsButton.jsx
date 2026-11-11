import { useRef, useState } from "react";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import Icon from "../../ui/Icon";

export default function ExportContractsButton({
  supplier,
  orgID = 0,
  disabled,
}) {
  const { api } = useAuth();
  const [busy, setBusy] = useState(false);
  const pending = useRef(false);

  async function download() {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    try {
      const params = new URLSearchParams({
        dobavljac: supplier,
        id_ugo_org: orgID,
      });
      const blob = await api(`/ugo_evid/otvoreni/excel?${params}`, {
        responseType: "blob",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "otvoreni-ugovori.xlsx";
      document.body.append(link);
      link.click();
      link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      toast.error(error.message);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }

  return (
    <button
      type="button"
      className="button"
      disabled={disabled || busy}
      onClick={download}
      title="Izvoz svih otvorenih ugovora koji odgovaraju aktivnom filteru"
    >
      <Icon name="file" />
      {busy ? "Priprema Excel fajla…" : "Izvezi u Excel"}
    </button>
  );
}
