import { expect, test as setup } from "@playwright/test";

import { admin, login } from "./helpers";

// Accede come Admin una volta e salva i cookie: gli altri test partono già
// autenticati senza consumare i tentativi di login limitati per IP.
setup("accesso come Admin", async ({ page }) => {
  expect(admin.username, "imposta E2E_ADMIN_USERNAME e E2E_ADMIN_PASSWORD").toBeTruthy();

  await login(page, admin.username, admin.password);
  await expect(page).toHaveURL(/\/dashboard/);

  await page.context().storageState({ path: "e2e/.auth/admin.json" });
});
