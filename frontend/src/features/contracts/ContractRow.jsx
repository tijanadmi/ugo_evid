import Icon from '../../ui/Icon';
import { value } from '../../utils/contractFormatting';
import ResponsiblePersons from './ResponsiblePersons';
export default function ContractRow({ item, onSelect }) { return (<tr>
          <th scope="row"><button className="contract-link" onClick={() => onSelect(item)}>{value(item, 'br_ugovor')}</button><small>{value(item, 'godina')}</small></th>
          <td className="subject-cell"><span>{value(item, 'predmet_ugovora')}</span><small>JN: {value(item, 'jn')}</small></td>
          <td className="supplier-cell">{value(item, 'naziv')}</td><td><span className="unit-badge">{value(item, 'sluzba')}</span></td>
          <td className="nowrap">{value(item, 'pocetak_ug')}</td><td className="nowrap">{value(item, 'kraj_ug')}</td><td className="responsible-cell"><ResponsiblePersons item={item}/></td>
          <td className="contact-cell">{value(item, 'ime')}<small>{value(item, 'email')}</small></td><td><button className="icon-button" onClick={() => onSelect(item)} aria-label={`Detalji ugovora ${item.br_ugovor}`}><Icon name="chevron"/></button></td>
        </tr>); }
