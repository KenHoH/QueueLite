import { Link, Route, Routes } from 'react-router-dom';
import { Card, PageContainer } from './components/foundation';
import { CustomerHeader } from './components/CustomerHeader';
import { AuthPage } from './pages/AuthPage';
import { plannedRoutes } from './routes';
import Home from './pages/Home';
import BusinessDetail from './pages/BusinessDetail';
import JoinQueue from './pages/JoinQueue';
import QueueStatus from './pages/QueueStatus';
import MyQueues from './pages/MyQueues';
import Profile from './pages/Profile';
import Plans from './pages/Plans';
import ManageBusinesses from './pages/ManageBusinesses';
import { CreateBusiness, BusinessSettings } from './pages/BusinessSetup';
function StagePlaceholder({ title }: { title: string }) {
  return <PageContainer><Card><p className="ql-meta">QueueLite</p><h1>{title}</h1><p className="ql-muted">{title === 'Join queue' ? 'Joining queues is coming in Stage 3. No queue has been created.' : 'This page is reserved for a later development stage.'}</p><Link to="/">Return home</Link></Card></PageContainer>;
}
export default function App() {
  return <><CustomerHeader /><Routes>
    <Route path="/" element={<Home />} />
    <Route path="/business/:businessId" element={<BusinessDetail />} />
    <Route path="/business/:businessId/join" element={<JoinQueue />} />
    <Route path="/queue/:queueId" element={<QueueStatus />} />
    <Route path="/my-queues" element={<MyQueues />} />
    <Route path="/profile" element={<Profile />} />
    <Route path="/plans" element={<Plans />} />
    <Route path="/business/create" element={<CreateBusiness />} />
    <Route path="/business/manage" element={<ManageBusinesses />} />
    <Route path="/business/:businessId/settings" element={<BusinessSettings />} />
    <Route path="/login" element={<AuthPage key="login" />} />
    <Route path="/register" element={<AuthPage key="register" registering />} />
    {plannedRoutes.filter(({ path }) => ['/dashboard', '/counters', '/counter/:counterId'].includes(path)).map(({ path, title }) => <Route key={path} path={path} element={<StagePlaceholder title={title} />} />)}
    <Route path="*" element={<PageContainer><Card><h1>Page not found</h1><Link to="/">Return to QueueLite</Link></Card></PageContainer>} />
  </Routes></>;
}
