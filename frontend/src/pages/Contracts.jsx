import ContractTable from '../features/contracts/ContractTable';
export default function Contracts({ status }) { return <ContractTable key={status} status={status}/>; }
