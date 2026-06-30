import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import WebApp from '@twa-dev/sdk';
import { useEffect } from 'react';

export function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    WebApp.ready();
    WebApp.expand();
  }, []);

  useEffect(() => {
    if (location.pathname !== '/') {
      WebApp.BackButton.show();
      const handleTmaBack = () => {
        navigate('/');
      };
      WebApp.BackButton.onClick(handleTmaBack);
      return () => {
        WebApp.BackButton.offClick(handleTmaBack);
        WebApp.BackButton.hide();
      };
    } else {
      WebApp.BackButton.hide();
    }
  }, [location.pathname, navigate]);

  const handleBackClick = () => {
    navigate('/');
  };

  return (
    <div className="app-container">
      <header className="app-header">
        {location.pathname !== '/' && (
          <button onClick={handleBackClick} className="back-button-ui">
            ← Back
          </button>
        )}
        <h1>Finance App</h1>
      </header>
      <main className="app-content">
        <Outlet />
      </main>
    </div>
  );
}
