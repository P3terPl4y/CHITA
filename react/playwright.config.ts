import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: process.env.CHITA_BROWSER_URL || "http://127.0.0.1:3340",
    headless: true,
    launchOptions: {
      executablePath:
        process.env.CHITA_CHROME_PATH ||
        "/home/peter/.cache/puppeteer/chrome/linux-146.0.7680.31/chrome-linux64/chrome",
      args: ["--no-sandbox"],
    },
  },
});
