import { Link } from 'react-router-dom';
import Icon from '../../ui/Icon';
import { value } from '../../utils/contractFormatting';
import ResponsiblePersons from './ResponsiblePersons';
import SupplierAddress from './SupplierAddress';
export default function ContractRow({ item, returnTo }) { return (<tr>
          <th scope="row"><Link className="contract-link" to={`/ugovori/detalji/${item.id_ugo_evid}`} state={{ returnTo }}>{value(item, 'br_ugovor')}</Link><small>{value(item, 'godina')}</small></th>
          <td className="subject-cell"><span>{value(item, 'predmet_ugovora')}</span><small>JN: {value(item, 'jn')}</small></td>
          <td className="supplier-cell">{value(item, 'naziv')}<SupplierAddress item={item}/></td><td><span className="unit-badge">{value(item, 'sluzba')}</span></td>
          <td className="nowrap">{value(item, 'pocetak_ug')}</td><td className="nowrap">{value(item, 'kraj_ug')}</td><td className="responsible-cell"><ResponsiblePersons item={item}/></td>
          <td className="contact-cell">{value(item, 'ime')}<small>{value(item, 'email')}</small><small>{value(item, 'telefon')}</small></td><td><Link className="icon-button" to={`/ugovori/detalji/${item.id_ugo_evid}`} state={{ returnTo }} aria-label={`Detalji ugovora ${item.br_ugovor}`}><Icon name="chevron"/></Link></td>
        </tr>); }
