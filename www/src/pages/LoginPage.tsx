import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSetAtom } from 'jotai';
import { authTokenAtom, authDeviceIdAtom } from '@/store/atoms/auth-atoms';

const API_BASE_URL = import.meta.env.VITE_API_BASE || 'http://localhost:8080';

export default function LoginPage() {
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const setAuthToken = useSetAtom(authTokenAtom);
  const setAuthDeviceId = useSetAtom(authDeviceIdAtom);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const response = await fetch(`${API_BASE_URL}/auth/local/pair`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ masterPassword: password }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        if (response.status === 401) {
          throw new Error('Invalid password');
        } else if (response.status === 403) {
          throw new Error('Access denied: This endpoint only works from localhost');
        }
        throw new Error(errorText || `HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Store token in cookie (httpOnly cannot be set from JS, so we use secure cookie settings available to JS)
      // The token will be automatically sent with requests and also stored in localStorage for JS access
      const maxAge = 30 * 24 * 60 * 60; // 30 days in seconds
      document.cookie = `gitty_auth_token=${encodeURIComponent(data.token)}; path=/; max-age=${maxAge}; SameSite=Strict`;
      document.cookie = `gitty_auth_device_id=${encodeURIComponent(data.deviceId)}; path=/; max-age=${maxAge}; SameSite=Strict`;

      // Also store in localStorage via Jotai atoms for JS access
      setAuthToken(data.token);
      setAuthDeviceId(data.deviceId);

      // Redirect to main app
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect to server');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="max-w-md w-full px-6 py-8 bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Welcome to Gitty
          </h1>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Enter your master password to sign in
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              Master Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
              placeholder="Enter your master password"
            />
          </div>

          {error && (
            <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-4">
              <div className="flex">
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800 dark:text-red-300">
                    {error}
                  </h3>
                </div>
              </div>
            </div>
          )}

          <button
            type="submit"
            disabled={loading || !password}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-indigo-500 dark:hover:bg-indigo-600"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
