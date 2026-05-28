import type { Page, Locator } from '@playwright/test';

/**
 * Page object for the manufacturer's product view (Org1 dashboard).
 * Provides high-level "create product" and "verify product visible" verbs.
 */
export class ProductsPage {
  readonly page: Page;
  readonly newProductButton: Locator;
  readonly idInput: Locator;
  readonly skuInput: Locator;
  readonly nameInput: Locator;
  readonly batchInput: Locator;
  readonly manufacturerInput: Locator;
  readonly submitButton: Locator;
  readonly productTable: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newProductButton = page.getByRole('button', { name: /new product|create product|add product/i });
    this.idInput = page.getByLabel(/^id|product id/i);
    this.skuInput = page.getByLabel(/sku/i);
    this.nameInput = page.getByLabel(/^name|product name/i);
    this.batchInput = page.getByLabel(/batch/i);
    this.manufacturerInput = page.getByLabel(/manufacturer/i);
    this.submitButton = page.getByRole('button', { name: /save|create|submit/i });
    this.productTable = page.getByRole('table');
  }

  async goto() {
    await this.page.goto('/manufacturer');
  }

  async createProduct(data: {
    id: string;
    sku: string;
    name: string;
    batchId: string;
    manufacturer: string;
  }) {
    await this.newProductButton.click();
    await this.idInput.fill(data.id);
    await this.skuInput.fill(data.sku);
    await this.nameInput.fill(data.name);
    await this.batchInput.fill(data.batchId);
    await this.manufacturerInput.fill(data.manufacturer);
    await this.submitButton.click();
  }

  async expectProductRow(id: string) {
    await this.productTable.getByText(id).waitFor({ state: 'visible' });
  }
}
