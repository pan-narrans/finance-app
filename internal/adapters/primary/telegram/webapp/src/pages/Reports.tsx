import { useState, useEffect } from 'react';
import { fetchWithAuth } from '../utils/api';

interface ReportSection {
  Title: string;
  DateRange: string;
  Content: string;
}

export function Reports() {
  const [loading, setLoading] = useState(true);
  const [sections, setSections] = useState<ReportSection[]>([]);
  const [period, setPeriod] = useState('this month');

  useEffect(() => {
    setLoading(true);
    fetchWithAuth(`/api/reports?period=${period}`)
      .then((data) => {
        setSections(data || []);
        setLoading(false);
      })
      .catch((err) => {
        console.error(err);
        setLoading(false);
      });
  }, [period]);

  if (loading) return <div>Loading reports...</div>;

  return (
    <div className="page">
      <h2>Financial Reports</h2>
      
      <div className="period-selector">
        <button 
          className={period === 'this month' ? 'active' : ''} 
          onClick={() => setPeriod('this month')}
        >
          This Month
        </button>
        <button 
          className={period === 'last month' ? 'active' : ''} 
          onClick={() => setPeriod('last month')}
        >
          Last Month
        </button>
      </div>

      <div className="reports-container">
        {sections.length === 0 ? (
          <p>No data available for this period.</p>
        ) : (
          sections.map((section) => (
            <div key={section.Title} className="report-section">
              <h3>{section.Title}</h3>
              <div className="report-date-range">{section.DateRange}</div>
              <pre className="report-content">{section.Content}</pre>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
