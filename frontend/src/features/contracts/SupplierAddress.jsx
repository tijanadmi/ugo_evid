export default function SupplierAddress({ item }) {
  const address = [item.adresa, item.grad].map(value => String(value ?? '').trim()).filter(Boolean).join(', ');
  return address ? <small className="supplier-address">{address}</small> : null;
}
