import { expect } from "@playwright/test";

export const admin = {
  username: process.env.E2E_ADMIN_USERNAME,
  password: process.env.E2E_ADMIN_PASSWORD,
};

const MAILPIT_URL = process.env.MAILPIT_URL ?? "http://localhost:8025";

// unique genera un suffisso diverso a ogni esecuzione, così i test si
// possono ripetere senza ripulire il database.
export function unique() {
  return `${Date.now().toString(36)}${Math.floor(Math.random() * 1000)}`;
}

export async function login(page, username, password) {
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Accedi" }).click();
}

// passwordLink cerca in Mailpit l'ultima email ricevuta da recipient con
// l'oggetto indicato e ne estrae il link per impostare la password.
export async function passwordLink(recipient, subject) {
  let link;

  await expect
    .poll(
      async () => {
        const search = await fetch(`${MAILPIT_URL}/api/v1/search?query=${encodeURIComponent(`to:${recipient}`)}`);
        const message = (await search.json()).messages.find((m) => m.Subject === subject);
        if (!message) return false;

        const detail = await (await fetch(`${MAILPIT_URL}/api/v1/message/${message.ID}`)).json();
        link = detail.Text.match(/https?:\/\/\S+\/reset-password\/\S+/)?.[0];
        return Boolean(link);
      },
      { message: `email "${subject}" per ${recipient}` },
    )
    .toBe(true);

  return new URL(link).pathname;
}
