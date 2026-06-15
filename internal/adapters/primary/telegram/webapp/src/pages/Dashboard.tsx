import { Link } from 'react-router-dom';

export function Dashboard() {
  const menuItems = [
    { label: 'Add Transaction', path: '/add', icon: '➕' },
    { label: 'History', path: '/history', icon: '📜' },
    { label: 'Reports', path: '/reports', icon: '📊' },
    { label: 'Upload Bank Statement', path: '/upload', icon: '🏦' },
  ];

  return (
    <div className="dashboard">
      <div className="menu-grid">
        {menuItems.map((item) => (
          <Link key={item.path} to={item.path} className="menu-item">
            <span className="menu-icon">{item.icon}</span>
            <span className="menu-label">{item.label}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
