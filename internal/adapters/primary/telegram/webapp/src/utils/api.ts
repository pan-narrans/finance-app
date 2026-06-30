import WebApp from '@twa-dev/sdk';

export async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const headers = {
    ...options.headers,
    'X-TMA-Init-Data': WebApp.initData,
    'Content-Type': 'application/json',
  };

  const response = await fetch(url, { ...options, headers });

  if (!response.ok) {
    if (response.status === 401) {
      WebApp.showAlert('Unauthorized. Please restart the app.');
    }
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const text = await response.text();
  return text ? JSON.parse(text) : {};
}
