import { Link, Route, Routes } from 'react-router-dom';
import { Card, PageContainer } from './components/foundation';
import { CustomerHeader } from './components/CustomerHeader';
import { AuthPage } from './pages/AuthPage';
import Dashboard from './pages/Dashboard';
import CounterManagement from './pages/CounterManagement';
import CounterWorkspace from './pages/CounterWorkspace';
import Home from './pages/Home';
import BusinessDetail from './pages/BusinessDetail';
import JoinQueue from './pages/JoinQueue';
import QueueStatus from './pages/QueueStatus';
import MyQueues from './pages/MyQueues';
import Profile from './pages/Profile';
import Plans from './pages/Plans';
import ManageBusinesses from './pages/ManageBusinesses';
import { CreateBusiness, BusinessSettings } from './pages/BusinessSetup';
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
    <Route path="/business/:businessId/dashboard" element={<Dashboard />} />
    <Route path="/business/:businessId/counters" element={<CounterManagement />} />
    <Route path="/business/:businessId/plans" element={<Plans />} />
    <Route path="/counter/:counterId" element={<CounterWorkspace />} />
    <Route path="/dashboard" element={<ManageBusinesses />} />
    <Route path="/counters" element={<ManageBusinesses />} />
    <Route path="/login" element={<AuthPage key="login" />} />
    <Route path="/register" element={<AuthPage key="register" registering />} />
    <Route path="*" element={<PageContainer><Card><h1>Page not found</h1><Link to="/">Return to QueueLite</Link></Card></PageContainer>} />
  </Routes></>;
}
