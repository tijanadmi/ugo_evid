import ConfirmAction from './ConfirmAction';

export default function ConfirmDelete({ resourceName, onConfirm, disabled, onCloseModal, blocked = false }) {
  return <ConfirmAction title={`Obriši ${resourceName}`} label="Brisanje lica" onConfirm={onConfirm} onCancel={onCloseModal}
    disabled={disabled} blocked={blocked} variation="danger" cancelLabel="Odustani" confirmLabel={disabled ? 'Brisanje…' : 'Potvrdi brisanje'}>
    <p>Da li ste sigurni da želite trajno da obrišete {resourceName}? Ova radnja se ne može poništiti.</p>
    <p>Lice povezano sa ugovorima možete deaktivirati kroz izmenu.</p>
  </ConfirmAction>;
}
