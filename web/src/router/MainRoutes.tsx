import { useRoutes, type Location } from 'react-router-dom';
import { DashboardPage } from '@/pages/DashboardPage';
import { MonitoringCenterPage } from '@/pages/MonitoringCenterPage';
import { ModelPricesPage } from '@/pages/ModelPricesPage';

const mainRoutes = [
  { path: '/', element: <DashboardPage /> },
  { path: '/dashboard', element: <DashboardPage /> },
  { path: '/monitoring', element: <MonitoringCenterPage /> },
  { path: '/model-prices', element: <ModelPricesPage /> },
];

export function MainRoutes({ location }: { location?: Location }) {
  return useRoutes(mainRoutes, location);
}
