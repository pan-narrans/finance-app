import { useState } from 'react';
import { fetchWithAuth } from '../utils/api';
import WebApp from '@twa-dev/sdk';

interface ImportSummary {
  Total: number;
  Added: number;
  Updated: number;
  Failed: number;
}

export function Import() {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [summary, setSummary] = useState<ImportSummary | null>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!file) return;

    setUploading(true);
    setSummary(null);
    WebApp.HapticFeedback.impactOccurred('medium');

    const formData = new FormData();
    formData.append('file', file);

    try {
      // fetchWithAuth doesn't handle FormData well as it forces JSON content-type.
      // Need manual fetch here or update helper.
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
    </div>
  );
}
