import { useState, useEffect } from 'react';
import { fetchWithAuth } from '../utils/api';

interface Posting {
  Account: string;
  Amount: number | null;
  Currency: string;
}

interface Transaction {
  Date: string;
  Description: string;
  Code: string;
  Postings: Posting[];
}

export function History() {
  const [loading, setLoading] = useState(true);
  const [transactions, setTransactions] = useState<Transaction[]>([]);

  useEffect(() => {
    fetchWithAuth('/api/history?limit=50')
      .then((data) => {
        setTransactions(data);
        setLoading(false);
      })
      .catch((err) => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  if (loading) return <div>Loading history...</div>;

  return (
    <div className="page">
      <h2>Transaction History</h2>
      <div className="history-list">
        {transactions.length === 0 ? (
          <p>No transactions found.</p>
        ) : (
          transactions.map((tx) => (
            <div key={tx.Code || Math.random()} className="history-item">
              <div className="history-item-header">
                <span className="history-date">{new Date(tx.Date).toLocaleDateString()}</span>
                <span className="history-code">{tx.Code}</span>
              </div>
              <div className="history-description">{tx.Description}</div>
              <div className="history-postings">
                {tx.Postings.map((p, i) => (
                  <div key={i} className="history-posting">
                    <span className="history-account">{p.Account}</span>
                    <span className={`history-amount ${p.Amount && p.Amount < 0 ? 'negative' : 'positive'}`}>
                      {p.Amount?.toFixed(2)} {p.Currency}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
