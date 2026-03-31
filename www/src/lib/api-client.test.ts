import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { apiClient } from './api-client';

describe('ApiClient - getTokenizedFileDiff', () => {
  const originalFetch = global.fetch;
  const originalLocalStorage = global.localStorage;
  const originalDocument = global.document;

  beforeEach(() => {
    global.fetch = vi.fn();

    // Mock localStorage
    Object.defineProperty(global, 'localStorage', {
      value: {
        getItem: vi.fn(),
        setItem: vi.fn(),
        removeItem: vi.fn(),
      },
      writable: true,
    });

    // Mock document.cookie
    Object.defineProperty(global, 'document', {
      value: {
        cookie: '',
      },
      writable: true,
    });
  });

  afterEach(() => {
    global.fetch = originalFetch;
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'document', {
      value: originalDocument,
      writable: true,
    });
  });

  it('fetches tokenized diff with correct URL and parameters', async () => {
    const mockDiff = {
      filename: 'test.ts',
      hunks: [],
      additions: 5,
      deletions: 2,
    };

    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockDiff,
    } as Response);

    const result = await apiClient.getTokenizedFileDiff('repo-id', 'src/test.ts', false);

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/repos/repo-id/diff/tokenized/src%2Ftest.ts'),
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      }),
    );
    expect(result).toEqual(mockDiff);
  });

  it('throws ApiError on network failure', async () => {
    vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

    await expect(apiClient.getTokenizedFileDiff('repo-id', 'src/test.ts')).rejects.toThrow('Network error');
  });
});
