import { expect, test } from "@playwright/test";

import { login, passwordLink, unique } from "./helpers";

test("utente: invito, scelta della password e permessi dell'Operatore", async ({ page, browser }) => {
  const username = `e2e_${unique()}`;
  const email = `${username}@example.com`;

  // L'Admin crea l'utente e vede la scelta del ruolo.
  await page.goto("/utenti");
  await page.getByRole("button", { name: "Nuovo utente" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByLabel("Ruolo")).toBeVisible();

  await dialog.getByRole("button", { name: "Crea utente" }).click();
  await expect(dialog.getByText("campo obbligatorio").first()).toBeVisible();

  await dialog.locator("#first_name").fill("Olga");
  await dialog.getByLabel("Cognome").fill("Operatrice");
  await dialog.getByLabel("Username").fill(username);
  await dialog.getByLabel("Email").fill(email);
  await dialog.getByLabel("Data di nascita").fill("1990-05-10");
  await dialog.getByRole("button", { name: "Crea utente" }).click();
  await expect(page.getByText(`Utente ${username} creato`)).toBeVisible();

  // Il nuovo utente sceglie la password con il link dell'email di benvenuto.
  const operator = await browser.newContext({ storageState: { cookies: [], origins: [] } });
  const operatorPage = await operator.newPage();
  await operatorPage.goto(await passwordLink(email, "Benvenuto in Risma"));

  await operatorPage.getByLabel("Nuova password").fill("debole");
  await operatorPage.getByRole("button", { name: "Imposta la password" }).click();
  await expect(operatorPage.getByText("la password deve avere").first()).toBeVisible();

  await operatorPage.getByLabel("Nuova password").fill("Operatore1!");
  await operatorPage.getByLabel("Ripeti la password").fill("Operatore1!");
  await operatorPage.getByRole("button", { name: "Imposta la password" }).click();
  await expect(operatorPage.getByText("Password impostata")).toBeVisible();

  // L'Operatore accede e non può scegliere ruoli (scheda, §4.3).
  await login(operatorPage, username, "Operatore1!");
  await expect(operatorPage).toHaveURL(/\/dashboard/);
  await expect(operatorPage.getByText("Operatore", { exact: true })).toBeVisible();

  await operatorPage.getByRole("link", { name: "Utenti" }).click();
  await operatorPage.getByRole("button", { name: "Nuovo utente" }).click();
  await expect(operatorPage.getByRole("dialog").getByLabel("Ruolo")).toHaveCount(0);
  await operatorPage.keyboard.press("Escape");

  // Nessuno può disattivare sé stesso.
  await operatorPage.getByLabel("Cerca per username o email").fill(username);
  const ownRow = operatorPage.getByRole("row", { name: new RegExp(username) });
  await expect(ownRow.getByRole("button", { name: /Modifica/ })).toBeVisible();
  await expect(ownRow.getByRole("button", { name: /Disattiva/ })).toHaveCount(0);
  await operator.close();

  // L'Admin disattiva e riattiva l'utente.
  await page.getByLabel("Cerca per username o email").fill(username);
  const row = page.getByRole("row", { name: new RegExp(username) });
  await row.getByRole("button", { name: /Disattiva/ }).click();
  await dialog.getByRole("button", { name: "Disattiva" }).click();
  await expect(page.getByText(`Utente ${username} disattivato.`)).toBeVisible();

  await page.getByRole("tab", { name: "Disattivati" }).click();
  await row.getByRole("button", { name: /Riattiva/ }).click();
  await dialog.getByRole("button", { name: "Riattiva" }).click();
  await expect(page.getByText(`Utente ${username} riattivato.`)).toBeVisible();
});
