import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  timeout: 60000,
  // A separate worker/browser keeps theme emulation independent of the
  // preceding map zoom, full-page captures and tracking scenarios in Firefox.
  projects: [
    { name: "app", testIgnore: "**/theme.spec.ts" },
    { name: "theme", testMatch: "**/theme.spec.ts" },
  ],
  use: {
    baseURL: process.env.CHITA_BROWSER_URL || "http://127.0.0.1:3340",
    headless: true,
    browserName:
      process.env.CHITA_BROWSER === "firefox" ? "firefox" : "chromium",
    launchOptions:
      process.env.CHITA_BROWSER === "firefox"
        ? {}
        : {
            executablePath:
              process.env.CHITA_CHROME_PATH ||
              "/home/peter/.cache/puppeteer/chrome/linux-146.0.7680.31/chrome-linux64/chrome",
            args: ["--no-sandbox"],
          },
  },
});
