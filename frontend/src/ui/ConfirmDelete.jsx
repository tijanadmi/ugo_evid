export default function ConfirmDelete({ resourceName, onConfirm, disabled, onCloseModal, blocked = false }) {
  return <section className="confirm-delete" aria-label="Brisanje lica">
    <h3>Obriši {resourceName}</h3>
    <p>Da li ste sigurni da želite trajno da obrišete {resourceName}? Ova radnja se ne može poništiti.</p>
    <p>Lice povezano sa ugovorima možete deaktivirati kroz izmenu.</p>
    <div className="contact-actions"><button type="button" className="button" disabled={disabled} onClick={onCloseModal}>Odustani</button><button type="button" className="button danger" disabled={disabled || blocked} onClick={onConfirm}>{disabled ? 'Brisanje…' : 'Potvrdi brisanje'}</button></div>
  </section>;
}
