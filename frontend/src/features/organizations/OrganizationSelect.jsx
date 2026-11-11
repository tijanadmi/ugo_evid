export default function OrganizationSelect({
  organizations,
  value,
  onChange,
  all = false,
}) {
  return (
    <label className="organization-select">
      Organizaciona jedinica
      <select
        aria-label="Organizaciona jedinica"
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      >
        <option value={0}>{all ? "Sve dozvoljene OJ" : "Izaberite OJ"}</option>
        {organizations.map((org) => (
          <option key={org.id} value={org.id}>
            {org.sifra} — {org.naziv}
          </option>
        ))}
      </select>
    </label>
  );
}
