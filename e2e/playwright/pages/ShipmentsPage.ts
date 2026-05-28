import type { Page, Locator } from '@playwright/test';

/**
 * Page object for shipment-related actions across manufacturer (Org1) and
 * distributor (Org2) views. Methods are role-agnostic — the navigation
 * verbs take care of routing to whatever org-scoped page is current.
 */
export class ShipmentsPage {
  readonly page: Page;
  readonly newShipmentButton: Locator;
  readonly idInput: Locator;
  readonly receiverInput: Locator;
  readonly productIdsInput: Locator;
  readonly submitButton: Locator;
  readonly acceptButton: Locator;
  readonly dispatchButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newShipmentButton = page.getByRole('button', { name: /new shipment|create shipment/i });
    this.idInput = page.getByLabel(/shipment id|^id$/i);
    this.receiverInput = page.getByLabel(/receiver|destination msp/i);
    this.productIdsInput = page.getByLabel(/product ids/i);
    this.submitButton = page.getByRole('button', { name: /save|create|submit/i });
    this.acceptButton = page.getByRole('button', { name: /accept custody|accept/i });
    this.dispatchButton = page.getByRole('button', { name: /dispatch/i });
  }

  async gotoManufacturer() {
    await this.page.goto('/manufacturer/shipments');
  }

  async gotoDistributor() {
    await this.page.goto('/distributor');
  }

  async createShipment(data: {
    id: string;
    receiverMsp: string;
    productIds: string[];
  }) {
    await this.newShipmentButton.click();
    await this.idInput.fill(data.id);
    await this.receiverInput.fill(data.receiverMsp);
    await this.productIdsInput.fill(data.productIds.join(','));
    await this.submitButton.click();
  }

  async dispatch() {
    await this.dispatchButton.click();
  }

  async acceptCustody() {
    await this.acceptButton.click();
  }
}
