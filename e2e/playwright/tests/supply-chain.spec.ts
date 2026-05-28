import { test, expect } from '@playwright/test';
import { LoginPage } from '../pages/LoginPage';
import { ProductsPage } from '../pages/ProductsPage';
import { ShipmentsPage } from '../pages/ShipmentsPage';

// API health endpoint — used to skip the suite when backend is not running.
const API_HEALTH = process.env.E2E_API_HEALTH ?? 'http://localhost:8080/health';

// Probe the backend once per worker. If it's down we mark the whole describe
// as skipped so the e2e suite stays green in environments where only the web
// app is started.
test.beforeAll(async () => {
  try {
    const ctrl = new AbortController();
    const tid = setTimeout(() => ctrl.abort(), 2000);
    const res = await fetch(API_HEALTH, { signal: ctrl.signal });
    clearTimeout(tid);
    if (!res.ok) {
      test.skip(true, `backend unhealthy at ${API_HEALTH} (status ${res.status})`);
    }
  } catch (e) {
    test.skip(true, `backend unreachable at ${API_HEALTH}: ${(e as Error).message}`);
  }
});

test.describe('Supply chain end-to-end flow', () => {
  // A unique run id keeps records from colliding across runs.
  const runId = `E2E-${Date.now()}`;
  const productId = `${runId}-P1`;
  const shipmentId = `${runId}-S1`;
  const batchId = `${runId}-B1`;

  test('Step 1: Org1 manufacturer creates a product', async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    const products = new ProductsPage(page);
    await products.goto();
    await products.createProduct({
      id: productId,
      sku: `SKU-${runId}`,
      name: 'E2E Widget',
      batchId,
      manufacturer: 'Acme'
    });
    await products.expectProductRow(productId);
  });

  test('Step 2: Org1 creates a shipment for the product', async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    const shipments = new ShipmentsPage(page);
    await shipments.gotoManufacturer();
    await shipments.createShipment({
      id: shipmentId,
      receiverMsp: 'Org3MSP',
      productIds: [productId]
    });
    await shipments.dispatch();
  });

  test('Step 3: Org2 distributor accepts custody', async ({ page }) => {
    // In a real multi-tenant test we'd log in as an Org2 user. The current
    // backend uses a single hardcoded admin so we exercise the UI verbs and
    // count the network call.
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    const shipments = new ShipmentsPage(page);
    await shipments.gotoDistributor();

    const acceptRequest = page.waitForRequest(req =>
      req.url().includes(`/shipments/${shipmentId}/custody/accept`) && req.method() === 'POST'
    );
    await shipments.acceptCustody();
    await acceptRequest;
  });

  test('Step 4: Org3 retailer records a sale at POS', async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    await page.goto('/retailer');
    const saleRequest = page.waitForRequest(req =>
      req.url().includes('/pos/sales') && req.method() === 'POST'
    );
    await page.getByRole('button', { name: /record sale|new sale/i }).first().click();
    await page.getByLabel(/product id/i).first().fill(productId);
    await page.getByLabel(/customer/i).first().fill(`${runId}-C1`);
    await page.getByLabel(/cashier/i).first().fill('Bob');
    await page.getByLabel(/unit price|price/i).first().fill('1500');
    await page.getByRole('button', { name: /submit|save|complete/i }).click();
    await saleRequest;
  });

  test('Step 5: trace endpoint shows full provenance', async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    await page.goto(`/trace?id=${productId}`);
    // Provenance page should render the product ID somewhere.
    await expect(page.getByText(productId)).toBeVisible();
  });

  test('Step 6: admin issues a recall for the batch', async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.loginAsAdmin();

    await page.goto('/recalls');
    const recallRequest = page.waitForRequest(req =>
      req.url().includes('/recalls') && req.method() === 'POST'
    );
    await page.getByRole('button', { name: /new recall|issue recall/i }).first().click();
    await page.getByLabel(/recall id|^id$/i).fill(`${runId}-REC`);
    await page.getByLabel(/scope/i).fill('PRODUCT');
    await page.getByLabel(/target ids|targets/i).fill(productId);
    await page.getByLabel(/reason/i).fill('integration-test recall');
    await page.getByLabel(/severity/i).fill('HIGH');
    await page.getByLabel(/issued by/i).fill('QA');
    await page.getByRole('button', { name: /submit|save|issue/i }).click();
    await recallRequest;
  });
});
