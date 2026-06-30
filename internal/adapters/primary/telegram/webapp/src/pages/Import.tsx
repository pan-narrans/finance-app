import { useState, useEffect } from 'react';
import { fetchWithAuth } from '../utils/api';
import WebApp from '@twa-dev/sdk';

interface Posting {
  Account: string;
  Amount?: number;
  Currency?: string;
}

interface Transaction {
  Date: string;
  Description: string;
  Code: string;
  Postings: Posting[];
}

interface ImportSummary {
  Total: number;
  Added: number;
  Updated: number;
  Failed: number;
  Pending?: Transaction[];
}

export function Import() {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [summary, setSummary] = useState<ImportSummary | null>(null);
  const [accounts, setAccounts] = useState<string[]>([]);
  const [resolvedAccounts, setResolvedAccounts] = useState<Record<string, string>>({});

  useEffect(() => {
    fetchWithAuth('/api/accounts')
      .then((data) => setAccounts(data?.accounts || []))
      .catch(console.error);
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!file) return;

    setUploading(true);
    setSummary(null);
    setResolvedAccounts({});
    WebApp.HapticFeedback.impactOccurred('medium');

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await fetch('/api/import', {
        method: 'POST',
        headers: {
          'X-TMA-Init-Data': WebApp.initData,
        },
        body: formData,
      });

      if (!response.ok) throw new Error('Upload failed');

      const data = await response.json();
      setSummary(data);
      WebApp.showAlert('Import completed!');
    } catch (err) {
      WebApp.showAlert('Import failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const handleCancel = () => {
    WebApp.HapticFeedback.notificationOccurred('warning');
    setSummary(null);
    setResolvedAccounts({});
    WebApp.showAlert('Import cancelled.');
  };

  const handleResolve = async (tx: Transaction) => {
    const target = resolvedAccounts[tx.Code];
    if (!target) {
      WebApp.showAlert('Please select a category first.');
      return;
    }

    try {
      const dateStr = new Date(tx.Date).toISOString().split('T')[0];
      const source = tx.Postings[1]?.Account || '';
      const amount = tx.Postings[0]?.Amount || 0;
      const currency = tx.Postings[0]?.Currency || 'EUR';

      await fetchWithAuth('/api/transaction', {
        method: 'POST',
        body: JSON.stringify({
          date: dateStr,
          description: tx.Description,
          amount: amount,
          source: source,
          target: target,
          currency: currency,
        }),
      });

      if (summary) {
        setSummary({
          ...summary,
          Pending: (summary.Pending || []).filter((p) => p.Code !== tx.Code),
          Added: summary.Added + 1,
        });
      }
      WebApp.HapticFeedback.notificationOccurred('success');
      WebApp.showAlert('Transaction resolved and saved!');
    } catch (err) {
      WebApp.showAlert('Failed to save transaction.');
    }
  };

  return (
    <div className="page">
      <h2>Bank Statement Upload</h2>
      <p className="hint">Upload your bank statement (Excel/CSV) to automatically import transactions.</p>
      
      <div className="upload-section">
        <label className="file-input-label">
          {file ? file.name : 'Tap to select file'}
          <input type="file" onChange={handleFileChange} style={{ display: 'none' }} />
        </label>

        <button 
          className="submit-button" 
          onClick={handleUpload} 
          disabled={!file || uploading}
        >
          {uploading ? 'Importing...' : 'Start Import'}
        </button>
      </div>

      {summary && (
        <div className="import-summary">
          <h3>Import Summary</h3>
          <div className="summary-grid">
            <div className="summary-item">
              <span className="summary-val">{summary.Total}</span>
              <span className="summary-lab">Processed</span>
            </div>
            <div className="summary-item success">
              <span className="summary-val">{summary.Added}</span>
              <span className="summary-lab">Added</span>
            </div>
            <div className="summary-item info">
              <span className="summary-val">{summary.Updated}</span>
              <span className="summary-lab">Updated</span>
            </div>
            <div className="summary-item error">
              <span className="summary-val">{summary.Failed}</span>
              <span className="summary-lab">Failed</span>
            </div>
          </div>
        </div>
      )}

      {summary && summary.Pending && summary.Pending.length > 0 && (
        <div className="pending-review-section">
          <div className="pending-review-header">
            <h3>Review Pending Transactions ({summary.Pending.length})</h3>
            <button className="cancel-import-btn" onClick={handleCancel}>
              Cancel
            </button>
          </div>
          <p className="hint">The following transactions have unknown categories. Please resolve them below:</p>
          <div className="pending-list">
            {summary.Pending.map((tx) => {
              const amount = tx.Postings[0]?.Amount || 0;
              const currency = tx.Postings[0]?.Currency || 'EUR';
              const dateStr = new Date(tx.Date).toISOString().split('T')[0];

              return (
                <div key={tx.Code} className="pending-tx-card">
                  <div className="pending-tx-header">
                    <span className="pending-tx-date">{dateStr}</span>
                    <span className="pending-tx-amount">{amount.toFixed(2)} {currency}</span>
                  </div>
                  <div className="pending-tx-desc">{tx.Description}</div>
                  <div className="pending-tx-actions">
                    <input
                      list="accounts-list"
                      placeholder="Search or select category"
                      value={resolvedAccounts[tx.Code] || ''}
                      onChange={(e) => setResolvedAccounts({ ...resolvedAccounts, [tx.Code]: e.target.value })}
                    />
                    <button className="resolve-btn" onClick={() => handleResolve(tx)}>
                      Resolve ✅
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      <datalist id="accounts-list">
        {accounts.map((acc) => (
          <option key={acc} value={acc} />
        ))}
      </datalist>
    </div>
  );
}
