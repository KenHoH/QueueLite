import { Link } from 'react-router-dom';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, LoadingSkeleton, PageContainer } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { canManage } from './accountForms';

export function BusinessSelector() {
  const { businesses, businessesStatus, refreshBusinesses, selectedBusinessId, setSelectedBusinessId } = useAppState();
  if (businessesStatus === 'loading') return <LoadingSkeleton label="Loading your businesses" />;
  if (businessesStatus === 'error') return <Card><InlineError>We couldn’t load your businesses.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  const managers = businesses.filter(canManage);
  if (!managers.length) return <Card><p className="ql-muted">{businesses.length ? 'Your account has staff access. Business settings require owner or admin access.' : 'You don’t own or manage a business yet.'}</p><Link to="/business/create">Create a business</Link></Card>;
  return <div className="ql-field"><label htmlFor="managed-business">Select business</label><select id="managed-business" className="ql-input" value={managers.some(item => item.id === selectedBusinessId) ? selectedBusinessId! : managers[0].id} onChange={event => setSelectedBusinessId(event.target.value)}>{managers.map(item => <option key={item.id} value={item.id}>{item.name} ({item.role})</option>)}</select></div>;
}
export default function ManageBusinesses() {
  const { businesses, businessesStatus, selectedBusinessId, setSelectedBusinessId } = useAppState();
  const managers = businesses.filter(canManage);
  const selected = managers.find(item => item.id === selectedBusinessId) ?? managers[0];
  return <PageContainer className="ql-account-page"><h1>Manage business</h1><AccountAccess><div className="ql-stack"><BusinessSelector />{businessesStatus === 'ready' && selected && <Card><h2>{selected.name}</h2><p className="ql-muted">{selected.location}</p><p>Your role: {selected.role}</p><p><Link to={`/business/${selected.id}/settings`} onClick={() => setSelectedBusinessId(selected.id)}>Business settings →</Link></p><Link to="/plans" onClick={() => setSelectedBusinessId(selected.id)}>View subscription</Link></Card>}<Link to="/business/create">Create another business</Link></div></AccountAccess></PageContainer>;
}
