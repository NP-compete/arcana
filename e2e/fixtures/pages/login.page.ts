import { Page, expect } from "@playwright/test";

const ROLE_LABELS: Record<string, string> = {
  admin: "Administrator",
  developer: "Developer",
  "data-engineer": "Data Engineer",
  sre: "SRE",
  auditor: "Auditor",
  user: "User",
};

export class LoginPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto("/");
    await this.page.waitForLoadState("networkidle");
  }

  async signInAs(role: string) {
    const label = ROLE_LABELS[role] || role;
    await this.page.getByLabel(`Sign in as ${label}`).click();
    await this.page.waitForLoadState("networkidle");
  }

  async isVisible() {
    const btn = this.page.getByLabel("Sign in as Administrator");
    return btn.isVisible({ timeout: 3000 }).catch(() => false);
  }

  async getRoleCardCount() {
    const cards = this.page.locator('[aria-label^="Sign in as"]');
    return cards.count();
  }
}
