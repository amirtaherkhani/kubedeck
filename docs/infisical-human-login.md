# User-run Infisical browser login (v0.151.0)

Run the local command below yourself. On macOS it opens your default browser
and also prints the exact official login URL. Complete your normal login/MFA/SSO
and select the approved organization. If opening fails, it clearly reports the
failure and keeps the printed URL usable for manual opening. Keep the terminal open. No password or token is entered
in chat, copied from developer tools, or passed as a shell argument.

```sh
/Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/bin/kuchdesk-enrollment \
  -login \
  -url https://infisical.local.dev \
  -policy /Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/infisical-control/policy.json \
  -state-dir /Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/infisical-control
```

The expected organization is `b46cad64-747c-4cec-a59a-e401cede95bd`. The controller
must be stopped for login; a competing writer produces `controller_already_running`.
Login creates/replaces only the private human session. It does not enable apply,
create an identity, grant project access, change Kubernetes auth, or restart a
service. Existing valid state is preserved when authentication fails.

Success prints sanitized JSON with:

```json
{"status":"human_authority_verified","serverAuthenticated":true,"organizationScopeVerified":true,"accessAllProjectsVerified":true,"refreshAvailable":true}
```

An `observedAt` timestamp is also present. This proves the current server accepts
the human token and permits `AccessAllProjects`, the capability used by the
self-enrollment endpoint. It does not claim an exact organization role name or
prove that dependent identity/project operations have already completed.

For a separate read-only check, run the same command with **`-status` instead of
`-login`**. Status can run alongside the controller: it does not take its writer
lock, refresh tokens, modify files or grant access. It reports `session_missing`,
`session_expired`, denied/unavailable, or verified authority. `refreshAvailable`
means a refresh credential is present, not that it has been tested. An expired
access token is not reported as authenticated simply because refresh is available.
Normal controller refresh remains separate. Dry-run reconciliation explicitly
reports `humanAuthority:"not_checked"`; a planned identity is never Admin proof.

The terminal prints only fixed callback stages:

- `callback_waiting`: listener ready; no valid callback received yet.
- `callback_preflight_received`: a preflight reached the listener; this alone does not identify its sender.
- `callback_rejected_origin`, `callback_rejected_host`, `callback_rejected_route`,
  `callback_rejected_payload` or `callback_rejected_scope`: specific validation
  failed; no credentials or incoming field values are printed.
- `callback_received`: a well-formed, correctly scoped candidate reached the server checks.
- `callback_server_validation_failed`: use the sanitized final JSON status to
  distinguish denied, expired, unavailable or incomplete MFA.
- `callback_session_saved`: validated private state was persisted.
- `callback_timeout_or_cancel`: this attempt has ended; its old URL cannot work.

If Infisical displays **Copy to clipboard**, its automatic callback failed.
The exact browser transport cause may remain unknown. The official CLI also
supports this fallback; use the following separate, user-only recovery:

1. Stop any waiting login command with Ctrl-C. Do not reopen its old URL.
2. Run the command below yourself. It does not open a browser or start a callback listener.
3. Use the official page's Copy button, then paste into the **hidden prompt** and
   press Enter. Never paste the value into chat, a shell command, or a file.

```sh
/Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/bin/kuchdesk-enrollment-fallback \
  -login-token \
  -url https://infisical.local.dev \
  -policy /Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/infisical-control/policy.json \
  -state-dir /Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/infisical-control
```

This build is installed separately so an older running command is undisturbed.
The prompt requires a terminal, hides input, allows five minutes, bounds input
size, restores terminal settings and discards queued paste bytes on exit. It
never reads the clipboard or browser storage. The decoded official payload
receives the same authentication, organization, enrollment-authority and refresh
checks as the callback before private persistence. Failed validation preserves
existing state. Check the sanitized result; the fallback page alone is not proof
that this client has usable authority. If the value has expired, or the official
page is no longer available, a fresh user-run login is required after recovery
is ready. Do not repeat login merely because the automatic callback failed.

The callback listener expires after ten minutes; Ctrl-C cancels it. Never reuse
an old port or start a second command while the first remains active.
Unsupported login/MFA flows remain in the official UI; incomplete MFA or policy
denial fails closed.

## Supported protocol and boundaries

This uses the official, version-pinned flow rather than a password grant. The
[official CLI fallback](https://github.com/Infisical/cli/blob/v0.43.140/packages/cmd/login.go)
accepts a hidden base64 JSON payload; `-login-token` supports that same contract:

1. [Login page](https://github.com/Infisical/infisical/blob/v0.151.0/frontend/src/pages/auth/LoginPage/LoginPage.tsx) accepts `callback_port`.
2. [Organization selection](https://github.com/Infisical/infisical/blob/v0.151.0/frontend/src/pages/auth/SelectOrgPage/SelectOrgSection.tsx) handles SSO/MFA and posts `JTWToken`, email and an empty privateKey to `http://127.0.0.1:PORT/`.
3. The client checks human authentication and enrollment authority, then calls the official [select-organization route](https://github.com/Infisical/infisical/blob/v0.151.0/backend/src/server/routes/v3/login-router.ts) for the same organization. Its response supplies a fresh scoped access token and `jid` refresh cookie directly to the local client.
4. The returned access token is revalidated. Only then is `session.json` persisted under the approved mode-0700 directory, mode 0600, using the existing single-writer lock, atomic rename and fsync.
5. [Admin Console listing and self-enrollment](https://github.com/Infisical/infisical/blob/v0.151.0/backend/src/services/org-admin/org-admin-service.ts) require the same `AccessAllProjects` capability. Status uses the read-only listing, never the grant endpoint.

The listener binds only IPv4 loopback on an ephemeral port and accepts the exact
configured HTTPS Origin and listener Host. It requires a bounded JSON POST with
exact, unique field names, rejects extra data and nonempty private keys, validates
organization/expiry hints, and verifies the token with the trusted server before
persistence. Requests are single-use after a well-formed candidate is accepted;
callbacks with malformed or wrong-scope data cannot replace state. Redirects are
not followed. Failure output never contains upstream bodies or credentials.

The upstream protocol does **not** echo an OAuth `state` nonce. Do not describe
this as PKCE or cryptographic state validation. Cross-site browser requests are
restricted by Origin/Host/CORS/content-type checks; local privileged processes
that can forge headers or read the user's files are outside that boundary. This
is the same official callback contract, with additional strict validation.

Passwords, captcha, legacy SRP, SSO and MFA remain in Infisical's own UI. No new
password implementation or provider-specific bypass is introduced. Tests use
synthetic authenticated MFA/SSO callback tokens; live provider login remains a
user-run acceptance check. No SDK dependency was added. Current runtime refresh
continues to use the [v0.151.0 refresh route](https://github.com/Infisical/infisical/blob/v0.151.0/backend/src/server/routes/v1/auth-router.ts), which reuses the scoped refresh cookie. Modeled rotation for newer versions does not enable newer-version login.

## Verification and activation boundary

Regression coverage includes loopback binding, exact Origin/Host checks,
credentialed/PNA preflight, malformed/oversized/duplicate JSON, wrong organization,
expiry, server rejection, redirect refusal, single-use callback, cancellation,
timeout, incomplete MFA, missing refresh cookie, private persistence, preservation
of existing state, read-only status alongside the writer, and refresh persistence
after restart. Full Host race tests, vet and build are required before delivery.

No live credentials were entered during implementation. After the user's login
succeeds, continue the previously approved activation with a fresh dry run,
reviewed identity/project reconciliation, Kubernetes-auth validation and consumer
cutover. Login alone does not complete those steps.
