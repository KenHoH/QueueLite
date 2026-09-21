import { useEffect } from "react";
import { Route, Routes, useLocation } from "react-router-dom";
import { CustomerLayout, BusinessLayout } from "./components/Layout";
import { EmptyState } from "./components/ui";
import Home from "./pages/Home";
import BusinessDetail from "./pages/BusinessDetail";
import JoinQueue from "./pages/JoinQueue";
import QueueStatus from "./pages/QueueStatus";
import Dashboard from "./pages/Dashboard";
import CounterWorkspace from "./pages/CounterWorkspace";
export default function App() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);
  return (
    <Routes>
      <Route element={<CustomerLayout />}>
        <Route index element={<Home />} />
        <Route path="business/:id" element={<BusinessDetail />} />
        <Route path="business/:id/join" element={<JoinQueue />} />
        <Route path="queue/:id" element={<QueueStatus />} />
        <Route
          path="*"
          element={
            <EmptyState
              title="This page has stepped out"
              description="Let's get you back to your neighborhood."
            />
          }
        />
      </Route>
      <Route element={<BusinessLayout />}>
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="counter/:id" element={<CounterWorkspace />} />
      </Route>
    </Routes>
  );
}
