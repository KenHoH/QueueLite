import { Link } from 'react-router-dom';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, LoadingSkeleton, PageContainer, PageHeader } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { canManage } from './accountForms';
import { BusinessNavigation } from '../components/BusinessNavigation';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';

export function BusinessSelector({ includeStaff = false }: { includeStaff?: boolean }) {
  const { businesses, businessesStatus, refreshBusinesses, selectedBusinessId, setSelectedBusinessId } = useAppState();
  if (businessesStatus === 'loading') return <LoadingSkeleton label="Loading your businesses" />;
  if (businessesStatus === 'error') return <Card><InlineError>We couldn’t load your businesses.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  const managers = includeStaff ? businesses : businesses.filter(canManage);
  if (!managers.length) return <Card><p className="ql-muted">{businesses.length ? 'Your account has staff access. Business settings require owner or admin access.' : 'You don’t own or manage a business yet.'}</p><Link to="/business/create">Create a business</Link></Card>;
  const value = managers.some(item => item.id === selectedBusinessId) ? selectedBusinessId! : managers[0].id;
  return <div className="ql-field"><Label htmlFor="managed-business">Select business</Label><Select value={value} onValueChange={setSelectedBusinessId}><SelectTrigger id="managed-business"><SelectValue /></SelectTrigger><SelectContent>{managers.map(item => <SelectItem key={item.id} value={item.id}>{item.name} ({item.role})</SelectItem>)}</SelectContent></Select></div>;
}
export default function ManageBusinesses() {
  const { businesses, businessesStatus, selectedBusinessId, setSelectedBusinessId } = useAppState();
  const managers = businesses;
  const selected = managers.find(item => item.id === selectedBusinessId) ?? managers[0];
  return <PageContainer className="ql-account-page"><PageHeader title="Your businesses" description="Choose a business to operate or manage." action={<Button asChild><Link to="/business/create">Create business</Link></Button>} /><AccountAccess><div className="ql-stack"><BusinessSelector includeStaff />{businessesStatus === 'ready' && selected && <Card><h2>{selected.name}</h2><p className="ql-muted">{selected.location}</p><p>Your role: {selected.role}</p><BusinessNavigation businessId={selected.id} manager={canManage(selected)} />{canManage(selected) && <><p><Link to={`/business/${selected.id}/settings`} onClick={() => setSelectedBusinessId(selected.id)}>Business settings →</Link></p><Link to={`/business/${selected.id}/plans`} onClick={() => setSelectedBusinessId(selected.id)}>View subscription</Link></>}</Card>}</div></AccountAccess></PageContainer>;
}
