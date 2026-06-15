import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import './index.css';
import { MainLayout } from './components/layout/MainLayout';
import { Dashboard } from './pages/Dashboard';
import { AddTransaction } from './pages/AddTransaction';
import { History } from './pages/History';
import { Reports } from './pages/Reports';
import { Import } from './pages/Import';

const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      {
        path: '/',
        element: <Dashboard />,
      },
      {
        path: '/add',
        element: <AddTransaction />,
      },
      {
        path: '/history',
        element: <History />,
      },
      {
        path: '/reports',
        element: <Reports />,
      },
      {
        path: '/upload',
        element: <Import />,
      },
    ],
  },
]);

function App() {
  return <RouterProvider router={router} />;
}

export default App;
