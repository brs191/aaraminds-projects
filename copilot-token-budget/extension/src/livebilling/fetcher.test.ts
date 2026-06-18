// extension/src/livebilling/fetcher.test.ts

import { GitHubEntitlementFetcher } from './fetcher';

describe('GitHubEntitlementFetcher', () => {
  let fetcher: GitHubEntitlementFetcher;

  beforeEach(() => {
    // Mock fetch globally
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should successfully fetch org entitlements', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token'
    );

    const mockResponse = {
      data: {
        viewer: {
          organization: {
            copilotQuota: 35000,
          },
        },
      },
    };

    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    const quota = await fetcher.fetchEntitlements('att-org');
    expect(quota).toBe(35000);
  });

  it('should throw error in dry-run mode', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token',
      10,
      true // dryRun
    );

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /dry-run/
    );
  });

  it('should throw error with empty orgSlug', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token'
    );

    await expect(fetcher.fetchEntitlements('')).rejects.toThrow(
      /orgSlug/
    );
  });

  it('should throw error with empty token', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      ''
    );

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /token/
    );
  });

  it('should handle GraphQL errors', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'bad-token'
    );

    const mockResponse = {
      errors: [{ message: 'Authentication failed' }],
    };

    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /GraphQL/
    );
  });

  it('should handle HTTP errors', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token'
    );

    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 401,
      text: async () => 'Unauthorized',
    });

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /401/
    );
  });

  it('should handle network errors', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token'
    );

    (global.fetch as jest.Mock).mockRejectedValueOnce(
      new Error('Network error')
    );

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /Network error/
    );
  });

  it('should handle timeout', async () => {
    fetcher = new GitHubEntitlementFetcher(
      'https://api.github.com',
      'mock-token',
      1 // 1 second timeout
    );

    const abortError = new Error('Timeout');
    abortError.name = 'AbortError';

    (global.fetch as jest.Mock).mockRejectedValueOnce(abortError);

    await expect(fetcher.fetchEntitlements('att-org')).rejects.toThrow(
      /timeout/i
    );
  });
});
