import type { Page, Locator } from '@playwright/test';

/**
 * Page object for the /login route.
 * The SvelteKit login form lives under src/routes/login. We bind to fields
 * by their accessible labels/placeholders rather than CSS selectors so the
 * test survives styling changes.
 */
export class LoginPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly errorMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.getByLabel(/username/i).or(page.getByPlaceholder(/username/i));
    this.passwordInput = page.getByLabel(/password/i).or(page.getByPlaceholder(/password/i));
    this.submitButton = page.getByRole('button', { name: /sign in|log in|login/i });
    this.errorMessage = page.getByRole('alert');
  }

  async goto() {
    await this.page.goto('/login');
  }

  async loginAs(username: string, password: string) {
    await this.usernameInput.fill(username);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  /** Convenience: login as the default admin credentials wired into the API. */
  async loginAsAdmin() {
    await this.loginAs('admin', 'asdqwe123');
  }
}
