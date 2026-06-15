import { Outlet } from 'react-router-dom';
import WebApp from '@twa-dev/sdk';
import { useEffect } from 'react';

export function MainLayout() {
  useEffect(() => {
    WebApp.ready();
    WebApp.expand();
  }, []);

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>Finance App</h1>
      </header>
      <main className="app-content">
        <Outlet />
      </main>
    </div>
  );
}
