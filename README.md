# KubeDeck

KubeDeck is a private, distribution-independent Kubernetes launchpad for
browsing clusters, nodes, web applications, and services by category, checking
their status and uptime, and opening or copying their ingress, cluster DNS, IP,
and port information.

## Local development

Requirements:

- Node.js `>=22.13.0`

Commands:

```bash
npm install
npm run dev
npm run lint
npm test
```

The project uses vinext, React, shadcn/ui, Cloudflare Workers, D1, and Drizzle.
The D1 binding is declared as `DB` in `.openai/hosting.json`, and
`npm run db:generate` creates migrations after schema changes.

## Admin setup and login

The private deployment opens a one-time admin setup screen when no administrator
exists. Enter the administrator's first name, last name, email, password, and
password confirmation directly in the browser.

- Initial setup requires the private Sites identity header.
- Only one admin account can be created.
- Passwords are stored as salted PBKDF2-SHA-256 hashes, never as plaintext.
- A signed, HTTP-only, secure cookie protects `/dashboard`.
- Sessions expire after 12 hours.
- Use **Sign out** in the dashboard header to end the current session.

The admin record is durable in D1. Redeploying the application does not replace
the configured account.

### Optional backend environment setup

The first administrator can also be created from server-side environment
variables:

```dotenv
KUBEDECK_ADMIN_FIRST_NAME=Amir
KUBEDECK_ADMIN_LAST_NAME=Taherkhani
KUBEDECK_ADMIN_EMAIL=admin@example.com
KUBEDECK_ADMIN_PASSWORD=use-a-unique-password-with-12-or-more-characters
```

Set all four values together and store `KUBEDECK_ADMIN_PASSWORD` as a secret in
the hosting environment. KubeDeck validates the values, creates the account in
D1 on the first request, and then shows **Sign in to KubeDeck**. The plaintext
password is never written to D1 or returned to the browser.

If all four values are unset, KubeDeck shows **Create the admin account** and
keeps the private browser setup flow available. An incomplete or invalid
environment configuration is ignored without creating a partial account.
Environment values never overwrite an administrator already stored in D1.
After the account has been created successfully, the password environment
variable can be removed.

Use `.env.example` as the local template. Never commit a populated `.env` file.

## Useful commands

- `npm run dev`: start local development
- `npm run build`: create the production Worker bundle
- `npm run lint`: run the source linter
- `npm test`: build and run the D1-backed authentication tests
- `npm run db:generate`: generate Drizzle migrations
