import DocumentList from "./DocumentList";

export default function ContractDocumentList({
  data,
  busy,
  content,
  code = "UGOVOR",
}) {
  const contractTypes = new Set(
    data?.types?.filter((type) => type.sifra === code).map((type) => type.id),
  );
  const items =
    data?.items?.filter((doc) => contractTypes.has(doc.id_tip_doc)) || [];

  if (!items.length)
    return (
      <p>
        {code === "UGOVOR"
          ? "Nema dodatih dokumenata tipa Ugovor."
          : "Nema dodatih rešenja za operativno praćenje ugovora."}
      </p>
    );

  return (
    <DocumentList
      items={items}
      busy={busy}
      canWrite={false}
      content={content}
    />
  );
}
