import { expect, test } from "@playwright/test";

import { admin } from "./helpers";

test.describe("senza sessione", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("le pagine riservate portano al login", async ({ page }) => {
    await page.goto("/utenti");
    await expect(page).toHaveURL(/\/login\?next=%2Futenti/);
  });

  test("login: errori nel browser e dal backend", async ({ page }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Accedi" }).click();
    await expect(page.getByText("inserisci lo username")).toBeVisible();
    await expect(page.getByText("inserisci la password")).toBeVisible();

    await page.getByLabel("Username").fill("utente-inesistente");
    await page.locator("#password").fill("Qualunque1!");
    await page.getByRole("button", { name: "Accedi" }).click();
    await expect(page.locator(".alert-danger")).toHaveText("credenziali non valide");
  });

  test("dopo il login si torna alla pagina richiesta", async ({ page }) => {
    await page.goto("/inventario");
    await page.getByLabel("Username").fill(admin.username);
    await page.locator("#password").fill(admin.password);
    await page.getByRole("button", { name: "Accedi" }).click();
    await expect(page).toHaveURL(/\/inventario$/);
  });

  test("next esterno ignorato (open redirect)", async ({ page }) => {
    await page.goto("/login?next=//example.com");
    await page.getByLabel("Username").fill(admin.username);
    await page.locator("#password").fill(admin.password);
    await page.getByRole("button", { name: "Accedi" }).click();
    await expect(page).toHaveURL(/localhost:3000\/dashboard$/);
  });

  test("recupero password: stessa risposta per qualsiasi email", async ({ page }) => {
    await page.goto("/recupera-password");
    await page.getByLabel("Email").fill("nessuno@example.com");
    await page.getByRole("button", { name: "Invia il link" }).click();
    await expect(page.getByText("Se l'email è associata a un account")).toBeVisible();
  });

  test("la pagina di reset non invia il Referer", async ({ page }) => {
    await page.goto("/reset-password/token-di-prova");
    await expect(page.locator('meta[name="referrer"]')).toHaveAttribute("content", "no-referrer");
  });

  test("link di reset non valido", async ({ page }) => {
    await page.goto("/reset-password/token-di-prova");
    await page.getByLabel("Nuova password").fill("Valida123!");
    await page.getByLabel("Ripeti la password").fill("Valida123!");
    await page.getByRole("button", { name: "Imposta la password" }).click();
    await expect(page.locator(".alert-danger")).toContainText("il link non è valido o è scaduto");
  });
});

test("rinnovo automatico della sessione con access token scaduto", async ({ page, context }) => {
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();

  await context.clearCookies({ name: "access_token" });
  await page.reload();

  await expect(page.getByRole("heading", { name: "Ultimi movimenti di magazzino" })).toBeVisible();
  expect((await context.cookies()).some((c) => c.name === "access_token")).toBe(true);
});

test("logout", async ({ page }) => {
  // Sessione propria: il logout revoca il refresh token, che non deve
  // essere quello condiviso dagli altri test.
  await page.context().clearCookies();
  await page.goto("/login");
  await page.getByLabel("Username").fill(admin.username);
  await page.locator("#password").fill(admin.password);
  await page.getByRole("button", { name: "Accedi" }).click();
  await expect(page).toHaveURL(/\/dashboard/);

  await page.getByRole("button", { name: "Esci" }).click();
  await expect(page).toHaveURL(/\/login$/);

  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login/);
});
