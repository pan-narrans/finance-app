import { useState, useEffect } from 'react';
import { fetchWithAuth } from '../utils/api';
import WebApp from '@twa-dev/sdk';
import { useNavigate } from 'react-router-dom';

export function AddTransaction() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [accounts, setAccounts] = useState<string[]>([]);
  const [formData, setFormData] = useState({
    date: new Date().toISOString().split('T')[0],
    description: '',
    amount: '',
    source: '',
    target: '',
    currency: 'EUR',
  });

  useEffect(() => {
    fetchWithAuth('/api/accounts')
      .then((data) => {
        setAccounts(data?.accounts || []);
        setLoading(false);
      })
      .catch((err) => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    WebApp.HapticFeedback.impactOccurred('medium');

    try {
      await fetchWithAuth('/api/transaction', {
        method: 'POST',
        body: JSON.stringify({
          ...formData,
          amount: parseFloat(formData.amount),
        }),
      });
      WebApp.showAlert('Transaction added successfully!');
      navigate('/');
    } catch (err) {
      WebApp.showAlert('Failed to add transaction. Please check inputs.');
    }
  };

  if (loading) return <div>Loading accounts...</div>;

  return (
    <div className="page">
      <h2>Add Transaction</h2>
      <form onSubmit={handleSubmit} className="transaction-form">
        <div className="form-group">
          <label>Date</label>
          <input
            type="date"
            value={formData.date}
            onChange={(e) => setFormData({ ...formData, date: e.target.value })}
            required
          />
        </div>

        <div className="form-group">
          <label>Description</label>
          <input
            type="text"
            placeholder="What for?"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            required
          />
        </div>

        <div className="form-group">
          <label>Amount</label>
          <div className="amount-input">
            <input
              type="number"
              step="0.01"
              placeholder="0.00"
              value={formData.amount}
              onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
              required
            />
            <input
              type="text"
              className="currency-input"
              value={formData.currency}
              onChange={(e) => setFormData({ ...formData, currency: e.target.value })}
              required
            />
          </div>
        </div>

        <div className="form-group">
          <label>Source (From)</label>
          <input
            list="accounts-list"
            placeholder="Select source account"
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
            required
          />
        </div>

        <div className="form-group">
          <label>Target (To)</label>
          <input
            list="accounts-list"
            placeholder="Select target account"
            value={formData.target}
            onChange={(e) => setFormData({ ...formData, target: e.target.value })}
            required
          />
        </div>

        <datalist id="accounts-list">
          {accounts.map((acc) => (
            <option key={acc} value={acc} />
          ))}
        </datalist>

        <button type="submit" className="submit-button">
          Save Transaction
        </button>
      </form>
    </div>
  );
}
