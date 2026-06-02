import React, { useMemo } from "react";

interface AccountSearchProps {
  accounts: string[];
  search: string;
  setSearch: (search: string) => void;
  type: string;
  onSelect: (account: string) => void;
  onCreateNew: () => void;
}

export const AccountSearch: React.FC<AccountSearchProps> = ({
  accounts,
  search,
  setSearch,
  type,
  onSelect,
  onCreateNew,
}) => {
  const filteredAccounts = useMemo(() => {
    if (!search) {
      return accounts;
    }
    const s = search.toLowerCase();
    return accounts.filter((acc) => acc.toLowerCase().includes(s));
  }, [accounts, search]);

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
            <button className="action-btn" onClick={onCreateNew}>
              ✨ Create New Account
            </button>
          </div>
        ) : (
          <>
            {filteredAccounts.map((acc) => (
              <div
                key={acc}
                className="list-item"
                onClick={() => onSelect(acc)}
              >
                <div className="account-name">{acc}</div>
                <div className="chevron">›</div>
              </div>
            ))}
            <div className="list-footer">
              <button className="text-btn" onClick={onCreateNew}>
                + Create another account
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
