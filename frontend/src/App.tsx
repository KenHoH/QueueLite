import { Link, Navigate, Route, Routes } from 'react-router-dom';
import { Card, PageContainer } from './components/foundation';
import { CustomerHeader } from './components/CustomerHeader';
import { RouteEffects } from './components/RouteEffects';
import { AccountRoute, BusinessRoute } from './components/RouteAccess';
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
import { ForbiddenPage, UnauthorizedPage } from './pages/AccessPage';
import StaffCounters from './pages/StaffCounters';
import QueueQRLanding from './pages/QueueQRLanding';
import BusinessMembers from './pages/BusinessMembers';
import ShareQueue from './pages/ShareQueue';
export default function App() {
  return <><RouteEffects /><CustomerHeader /><Routes>
    <Route path="/" element={<Home />} />
    <Route path="/business/:businessId" element={<BusinessDetail />} />
    <Route path="/qr/:businessId" element={<QueueQRLanding />} />
    <Route path="/business/:businessId/join" element={<JoinQueue />} />
    <Route path="/queue/:queueId" element={<QueueStatus />} />
    <Route path="/my-queues" element={<AccountRoute guestTitle="Sign in to view your queues." guestDescription="My Queues is available for customers with an account. Guest tickets can be opened in the browser where you joined."><MyQueues /></AccountRoute>} />
    <Route path="/profile" element={<AccountRoute><Profile /></AccountRoute>} />
    <Route path="/plans" element={<AccountRoute><Plans /></AccountRoute>} />
    <Route path="/business/create" element={<AccountRoute><CreateBusiness /></AccountRoute>} />
    <Route path="/business/manage" element={<AccountRoute><ManageBusinesses /></AccountRoute>} />
    <Route path="/business/:businessId/settings" element={<BusinessRoute roles={['owner', 'admin']}><BusinessSettings /></BusinessRoute>} />
    <Route path="/business/:businessId/dashboard" element={<BusinessRoute><Dashboard /></BusinessRoute>} />
    <Route path="/business/:businessId/counters" element={<BusinessRoute><CounterManagement /></BusinessRoute>} />
    <Route path="/business/:businessId/members" element={<BusinessRoute roles={['owner', 'admin']}><BusinessMembers /></BusinessRoute>} />
    <Route path="/business/:businessId/plans" element={<BusinessRoute roles={['owner', 'admin']}><Plans /></BusinessRoute>} />
    <Route path="/business/:businessId/share" element={<BusinessRoute roles={['owner', 'admin']}><ShareQueue /></BusinessRoute>} />
    <Route path="/counter/:counterId" element={<AccountRoute><CounterWorkspace /></AccountRoute>} />
    <Route path="/dashboard" element={<AccountRoute><Navigate to="/business/manage" replace /></AccountRoute>} />
    <Route path="/counters" element={<AccountRoute><StaffCounters /></AccountRoute>} />
    <Route path="/unauthorized" element={<UnauthorizedPage />} />
    <Route path="/forbidden" element={<ForbiddenPage />} />
    <Route path="/login" element={<AuthPage key="login" />} />
    <Route path="/register" element={<AuthPage key="register" registering />} />
    <Route path="*" element={<PageContainer><Card><h1>Page not found</h1><Link to="/">Return to QueueLite</Link></Card></PageContainer>} />
  </Routes></>;
}
