import { useEffect, useState, useMemo } from 'react';
import WebApp from '@twa-dev/sdk';

type Mode = 'search' | 'create-parent' | 'create-child';

function App() {
  const [accounts, setAccounts] = useState<string[]>([]);
  const [roots, setRoots] = useState<string[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode>('search');
  const [selectedParent, setSelectedParent] = useState('');
  const [newSubAccount, setNewSubAccount] = useState('');

  // Get search type from URL params (source or target)
  const queryParams = new URLSearchParams(window.location.search);
  const type = queryParams.get('type') || 'target';

  useEffect(() => {
    WebApp.ready();
    WebApp.expand();
    
    fetch('/api/accounts')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch accounts');
        return res.json();
      })
      .then((data: { accounts: string[], roots: string[] }) => {
        setAccounts(data.accounts);
        setRoots(data.roots);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    // Back button behavior
    if (mode === 'search') {
      WebApp.BackButton.hide();
    } else {
      WebApp.BackButton.show();
    }

    const handleBack = () => {
      if (mode === 'create-parent') setMode('search');
      if (mode === 'create-child') setMode('create-parent');
    };

    WebApp.onEvent('backButtonClicked', handleBack);
    return () => WebApp.offEvent('backButtonClicked', handleBack);
  }, [mode]);

  useEffect(() => {
    // Main button behavior for account creation
    if (mode === 'create-child' && newSubAccount.trim()) {
      WebApp.MainButton.setText(`CREATE & SELECT ${selectedParent}:${newSubAccount}`);
      WebApp.MainButton.show();
    } else {
      WebApp.MainButton.hide();
    }

    const handleSubmit = () => handleSelect(`${selectedParent}:${newSubAccount}`);
    WebApp.onEvent('mainButtonClicked', handleSubmit);
    return () => WebApp.offEvent('mainButtonClicked', handleSubmit);
  }, [mode, newSubAccount, selectedParent]);

  const filteredAccounts = useMemo(() => {
    if (!search) return accounts;
    const s = search.toLowerCase();
    return accounts.filter((acc) => acc.toLowerCase().includes(s));
  }, [accounts, search]);

  const handleSelect = async (account: string) => {
    WebApp.HapticFeedback.impactOccurred('medium');
    
    try {
      const response = await fetch('/api/select', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          initData: WebApp.initData,
          account,
          type
        }),
      });

      if (!response.ok) throw new Error('Selection failed');
      WebApp.close();
    } catch (err) {
      WebApp.showAlert('Failed to save selection. Please try again.');
    }
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (error) return <div className="error">{error}</div>;

  if (mode === 'create-parent') {
    return (
      <div className="container">
        <div className="wizard-header">Select Top-Level Account</div>
        <div className="grid">
          {roots.map(root => (
            <div key={root} className="grid-item" onClick={() => {
              setSelectedParent(root);
              setMode('create-child');
            }}>
              {root}
            </div>
          ))}
        </div>
        {renderStyles()}
      </div>
    );
  }

  if (mode === 'create-child') {
    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const trimmed = newSubAccount.trim();
        if (trimmed && !trimmed.endsWith(':')) {
          setNewSubAccount(trimmed + ':');
        }
      }
    };

    return (
      <div className="container">
        <div className="wizard-header">New Sub-account for {selectedParent}</div>
        <div className="input-group">
          <input
            type="text"
            className="search-input"
            placeholder="e.g. Dining"
            value={newSubAccount}
            onChange={(e) => setNewSubAccount(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
          />
          <div className="preview">
            Full path: <code>{selectedParent}:{newSubAccount || '...'}</code>
          </div>
        </div>
        {renderStyles()}
      </div>
    );
  }

  return (
    <div className="container">
      <header className="header">
        <input
          type="text"
          className="search-input"
          placeholder={`Search ${type} account...`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoFocus
        />
      </header>

      <div className="list">
        {filteredAccounts.length === 0 ? (
          <div className="empty-state">
            <div className="no-results">No accounts match your search.</div>
            <button className="action-btn" onClick={() => setMode('create-parent')}>
              ✨ Create New Account
            </button>
          </div>
        ) : (
          <>
            {filteredAccounts.map((acc) => (
              <div key={acc} className="list-item" onClick={() => handleSelect(acc)}>
                <div className="account-name">{acc}</div>
                <div className="chevron">›</div>
              </div>
            ))}
            <div className="list-footer">
              <button className="text-btn" onClick={() => setMode('create-parent')}>
                + Create another account
              </button>
            </div>
          </>
        )}
      </div>

      {renderStyles()}
    </div>
  );
}

function renderStyles() {
  return (
    <style>{`
      :root {
        --primary-color: var(--tg-theme-button-color, #2481cc);
        --bg-color: var(--tg-theme-bg-color, #ffffff);
        --secondary-bg: var(--tg-theme-secondary-bg-color, #f4f4f5);
        --text-color: var(--tg-theme-text-color, #222222);
        --hint-color: var(--tg-theme-hint-color, #999999);
        --button-text: var(--tg-theme-button-text-color, #ffffff);
      }

      .container {
        display: flex;
        flex-direction: column;
        height: 100vh;
        background-color: var(--bg-color);
        color: var(--text-color);
      }

      .header, .wizard-header {
        position: sticky;
        top: 0;
        padding: 12px;
        background-color: var(--bg-color);
        border-bottom: 1px solid var(--secondary-bg);
        z-index: 10;
        font-weight: 600;
        text-align: center;
      }

      .search-input {
        width: 100%;
        padding: 12px 14px;
        border-radius: 10px;
        border: none;
        background-color: var(--secondary-bg);
        color: var(--text-color);
        font-size: 16px;
        outline: none;
        box-sizing: border-box;
      }

      .list {
        flex: 1;
        overflow-y: auto;
      }

      .list-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 14px 16px;
        border-bottom: 1px solid var(--secondary-bg);
        cursor: pointer;
      }

      .list-item:active {
        background-color: var(--secondary-bg);
      }

      .account-name {
        font-size: 16px;
        word-break: break-all;
      }

      .chevron {
        color: var(--hint-color);
        font-size: 20px;
      }

      .empty-state {
        padding: 40px 20px;
        text-align: center;
      }

      .no-results {
        color: var(--hint-color);
        margin-bottom: 20px;
      }

      .action-btn {
        background-color: var(--primary-color);
        color: var(--button-text);
        border: none;
        padding: 12px 24px;
        border-radius: 8px;
        font-size: 16px;
        font-weight: 500;
        cursor: pointer;
      }

      .list-footer {
        padding: 20px;
        text-align: center;
      }

      .text-btn {
        background: none;
        border: none;
        color: var(--primary-color);
        font-size: 15px;
        cursor: pointer;
      }

      .grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
        padding: 16px;
      }

      .grid-item {
        background-color: var(--secondary-bg);
        padding: 20px;
        border-radius: 12px;
        text-align: center;
        font-weight: 500;
        cursor: pointer;
      }

      .grid-item:active {
        opacity: 0.7;
      }

      .input-group {
        padding: 16px;
      }

      .preview {
        margin-top: 12px;
        color: var(--hint-color);
        font-size: 14px;
      }

      code {
        background-color: var(--secondary-bg);
        padding: 2px 6px;
        border-radius: 4px;
        color: var(--text-color);
      }

      .loading, .error {
        padding: 100px 20px;
        text-align: center;
      }
    `}</style>
  );
}

export default App;
