# End-to-End Tests — Playwright

Browser-driven E2E suite covering the full supply chain flow described in
[spec section 10](../../docs/superpowers/specs/2026-05-24-production-readiness-design.md):

1. Login as Org1 → create product
2. Create shipment, dispatch
3. Switch to Org2 → accept custody
4. Switch to Org3 → record sale at POS
5. Verify provenance via `/trace`
6. Issue recall and check affected products

## Quick start

```bash
cd e2e/playwright
pnpm install
pnpm exec playwright install --with-deps chromium
pnpm test
```

## Environment variables

| Variable          | Default                       | Description                                  |
| ----------------- | ----------------------------- | -------------------------------------------- |
| `E2E_BASE_URL`    | `http://localhost:3000`       | SvelteKit web app URL                        |
| `E2E_API_HEALTH`  | `http://localhost:8080/health`| API health endpoint (probed before running) |

If the API health probe fails, the suite is skipped (not failed). This keeps
the e2e job green in environments where only the web app is up.

## Running against a full stack

```bash
# In a separate terminal — bring up Fabric + API + web
make deploy-all

# Wait for /health to return 200
curl -fsS http://localhost:8080/health

# Run the suite
cd e2e/playwright && pnpm test
```

## Layout

```
e2e/playwright/
├── package.json
├── playwright.config.ts        # Base URL, reporter, retries
├── tsconfig.json
├── pages/                      # Page Object Models
│   ├── LoginPage.ts
│   ├── ProductsPage.ts
│   └── ShipmentsPage.ts
└── tests/
    └── supply-chain.spec.ts    # Full 6-step happy path
```

## Notes on multi-org login

The current backend authenticates via a single hardcoded admin account.
Tests reuse that identity across roles and rely on the `org` claim wired
into the JWT at issue time. Once multi-tenant identity is in place
(`packages/api/internal/handler/auth.go`), swap `loginAsAdmin()` for
`loginAs(org, user, pass)` in the spec.
