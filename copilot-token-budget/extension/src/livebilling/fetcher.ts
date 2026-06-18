// extension/src/livebilling/fetcher.ts — Mirrors the Go fetcher for VS Code extension.
// Fetches org-level Copilot entitlements from GitHub's internal GraphQL API.

export interface EntitlementResponse {
  data?: {
    viewer?: {
      organization?: {
        copilotQuota: number;
      };
    };
  };
  errors?: Array<{ message: string }>;
}

export class GitHubEntitlementFetcher {
  constructor(
    private gitHubAPIURL: string,
    private token: string,
    private requestTimeoutSecs: number = 10,
    private dryRun: boolean = false
  ) {}

  async fetchEntitlements(orgSlug: string): Promise<number> {
    if (this.dryRun) {
      throw new Error('Fetcher: dry-run mode; no API call made');
    }

    if (!orgSlug || orgSlug.trim() === '') {
      throw new Error('Fetcher: orgSlug is empty');
    }

    if (!this.token || this.token.trim() === '') {
      throw new Error('Fetcher: auth token is empty');
    }

    const query = `{
      viewer {
        organization(login: "${orgSlug}") {
          copilotQuota
        }
      }
    }`;

    const payload = { query };
    const apiURL = this.gitHubAPIURL || 'https://api.github.com';
    const endpoint = `${apiURL}/graphql`;

    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(
        () => controller.abort(),
        this.requestTimeoutSecs * 1000
      );

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const body = await response.text();
        throw new Error(
          `Fetcher: GitHub API returned ${response.status}: ${body}`
        );
      }

      const result: EntitlementResponse = (await response.json()) as EntitlementResponse;

      if (result.errors && result.errors.length > 0) {
        const messages = result.errors.map((e) => e.message).join('; ');
        throw new Error(`Fetcher: GraphQL error: ${messages}`);
      }

      const quota =
        result.data?.viewer?.organization?.copilotQuota ?? 0;

      if (!quota || quota <= 0) {
        throw new Error(
          `Fetcher: org quota is ${quota} (zero or not set)`
        );
      }

      return quota;
    } catch (err) {
      if (err instanceof Error) {
        if (err.name === 'AbortError') {
          throw new Error('Fetcher: request timeout');
        }
        throw err;
      }
      throw new Error(`Fetcher: unknown error: ${err}`);
    }
  }
}
