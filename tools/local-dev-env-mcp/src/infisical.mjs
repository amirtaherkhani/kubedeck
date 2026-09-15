export class InfisicalStore {
  constructor({ credentials, context, fetchImpl = fetch }) {
    this.credentials = credentials;
    this.context = context;
    this.fetch = fetchImpl;
    this.accessToken = null;
    this.accessTokenExpiresAt = 0;
    this.loginPromise = null;
  }

  async token() {
    if (this.accessToken && Date.now() < this.accessTokenExpiresAt - 30_000) return this.accessToken;
    if (!this.loginPromise) {
      this.loginPromise = this.request("/api/v1/auth/universal-auth/login", {
        method: "POST",
        body: this.credentials,
        authenticated: false
      }).then(({ accessToken, expiresIn }) => {
        this.accessToken = accessToken;
        this.accessTokenExpiresAt = Date.now() + (expiresIn * 1_000);
        return accessToken;
      }).finally(() => {
        this.loginPromise = null;
      });
    }
    return this.loginPromise;
  }

  scope(extra = {}, path = this.context.path) {
    return {
      workspaceId: this.context.projectId,
      environment: this.context.environment,
      secretPath: path,
      ...extra
    };
  }

  async request(path, { method = "GET", query, body, authenticated = true, retryAuth = true } = {}) {
    const url = new URL(path, this.context.domain);
    for (const [key, value] of Object.entries(query ?? {})) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
    const headers = { "content-type": "application/json" };
    if (authenticated) headers.authorization = `Bearer ${await this.token()}`;
    const response = await this.fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(30_000)
    });
    if (response.status === 401 && authenticated && retryAuth) {
      this.accessToken = null;
      this.accessTokenExpiresAt = 0;
      return this.request(path, { method, query, body, authenticated, retryAuth: false });
    }
    if (!response.ok) throw new Error(`Infisical request failed [StatusCode=${response.status}]`);
    return response.status === 204 ? {} : response.json();
  }

  async list(path) {
    const result = await this.request("/api/v3/secrets/raw", { query: this.scope({ viewSecretValue: false }, path) });
    return result.secrets.map(({ secretKey }) => secretKey).sort();
  }

  async get(name, path) {
    const result = await this.request(`/api/v3/secrets/raw/${encodeURIComponent(name)}`, {
      query: this.scope({ type: "shared", viewSecretValue: true }, path)
    });
    return result.secret;
  }

  async set(name, value, path) {
    try {
      await this.request(`/api/v3/secrets/raw/${encodeURIComponent(name)}`, {
        query: this.scope({ type: "shared", viewSecretValue: false }, path)
      });
      await this.request(`/api/v3/secrets/raw/${encodeURIComponent(name)}`, {
        method: "PATCH",
        body: this.scope({ secretValue: value, type: "shared" }, path)
      });
      return "updated";
    } catch (error) {
      if (!String(error?.message ?? error).includes("StatusCode=404")) throw error;
      await this.request(`/api/v3/secrets/raw/${encodeURIComponent(name)}`, {
        method: "POST",
        body: this.scope({ secretValue: value, type: "shared" }, path)
      });
      return "created";
    }
  }

  async setMany(values, path) {
    const results = [];
    for (const [name, value] of Object.entries(values)) {
      results.push({ name, action: await this.set(name, value, path) });
    }
    return results;
  }

  async delete(name, path) {
    await this.request(`/api/v3/secrets/raw/${encodeURIComponent(name)}`, {
      method: "DELETE",
      body: this.scope({ type: "shared" }, path)
    });
  }
}
