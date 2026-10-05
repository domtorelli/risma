import { defineConfig, devices } from "@playwright/test";

// Test end-to-end dell'interfaccia. Richiedono il backend in esecuzione
// (make run), il database con le migrazioni, Mailpit e un Admin creato con
// make seed, le cui credenziali si passano in E2E_ADMIN_USERNAME e
// E2E_ADMIN_PASSWORD. Il frontend viene avviato da Playwright se non è già
// attivo.
export default defineConfig({
  testDir: "./e2e",
  // I test condividono il backend: in sequenza i dati e i limiti di
  // richieste per IP restano prevedibili.
  workers: 1,
  reporter: "list",
  use: {
    baseURL: "http://localhost:3000",
    locale: "it-IT",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "setup", testMatch: /auth\.setup\.js/ },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], storageState: "e2e/.auth/admin.json" },
      dependencies: ["setup"],
    },
  ],
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3000/login",
    reuseExistingServer: true,
    timeout: 120_000,
  },
});
