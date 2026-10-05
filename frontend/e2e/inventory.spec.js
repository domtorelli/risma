import { expect, test } from "@playwright/test";

import { unique } from "./helpers";

test("prodotto: creazione, carico/scarico e avviso sotto soglia", async ({ page }) => {
  const name = `E2E Risma ${unique()}`;

  await page.goto("/inventario");
  await page.getByRole("button", { name: "Nuovo prodotto" }).click();
  const dialog = page.getByRole("dialog");

  await dialog.getByLabel("Tipo").selectOption({ label: "Carta" });
  await dialog.getByLabel("Nome").fill(name);
  await dialog.getByLabel("Prezzo unitario (€)").fill("4,90");
  await dialog.getByLabel("Soglia minima").fill("20");
  await dialog.getByLabel("Quantità iniziale").fill("25");
  await dialog.getByRole("button", { name: "Crea prodotto" }).click();
  await expect(dialog.getByText("prezzo non valido")).toBeVisible();

  await dialog.getByLabel("Prezzo unitario (€)").fill("4.90");
  await dialog.getByRole("button", { name: "Crea prodotto" }).click();
  await expect(page.getByText(`Prodotto ${name} creato.`)).toBeVisible();

  await page.getByLabel("Cerca per nome").fill(name);
  const row = page.getByRole("row", { name: new RegExp(name) });
  await expect(row).toContainText("4,90 €");
  await expect(row).toContainText("25");

  // Scarico che porta sotto soglia (25 - 10 = 15 < 20).
  await row.getByRole("button", { name: /Carico\/Scarico/ }).click();
  await dialog.getByText("Scarico", { exact: true }).click();
  await dialog.getByLabel("Quantità").fill("10");
  await dialog.getByLabel("Nota").fill("Vendita al banco");
  await dialog.getByRole("button", { name: "Registra movimento" }).click();
  await expect(page.getByText(/gli amministratori sono stati avvisati/)).toBeVisible();
  await expect(row.getByText("sotto soglia")).toBeVisible();

  // Scarico oltre la giacenza: rifiutato dal backend.
  await row.getByRole("button", { name: /Carico\/Scarico/ }).click();
  await dialog.getByText("Scarico", { exact: true }).click();
  await dialog.getByLabel("Quantità").fill("500");
  await dialog.getByRole("button", { name: "Registra movimento" }).click();
  await expect(dialog.getByText("quantità disponibile insufficiente")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);

  // Nome duplicato (maiuscole diverse): errore del backend sul campo.
  await page.getByRole("button", { name: "Nuovo prodotto" }).click();
  await dialog.getByLabel("Tipo").selectOption({ label: "Carta" });
  await dialog.getByLabel("Nome").fill(name.toUpperCase());
  await dialog.getByLabel("Prezzo unitario (€)").fill("1");
  await dialog.getByRole("button", { name: "Crea prodotto" }).click();
  await expect(dialog.getByText("esiste già un prodotto con questo nome")).toBeVisible();
  await dialog.getByRole("button", { name: "Annulla" }).click();

  // Eliminazione.
  await row.getByRole("button", { name: /Elimina/ }).click();
  await dialog.getByRole("button", { name: "Elimina" }).click();
  await expect(page.getByText(`Prodotto ${name} eliminato.`)).toBeVisible();
  await expect(row).toHaveCount(0);
});

test("dashboard: indicatori, grafico e ultimi movimenti", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page.getByText("Valore dell'inventario")).toBeVisible();
  await expect(page.locator("canvas")).toHaveCount(1);
  await expect(page.getByRole("table", { name: "Numero di prodotti per tipo" })).toBeAttached();
  await expect(page.getByRole("heading", { name: "Ultimi movimenti di magazzino" })).toBeVisible();
});
