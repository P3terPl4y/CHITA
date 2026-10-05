import { test, expect } from "@playwright/test";

test("dashboard polls without overlapping requests and slows secondary data", async ({ page }) => {
  const calls = { jobs: 0, network: 0, notifications: 0, halcon: 0 };
  const totalModes: string[] = [];
  const origins: string[] = [];
  await page.clock.install({ time: new Date("2026-10-04T12:00:00Z") });
  await page.addInitScript(() => { Math.random = () => 0; });
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    let data: unknown = [];
    if (path === "/api/session") data = { user: { id: 21, name: "Repartidor", email: "poll@example.test", role: "courier", phone: "" }, csrf_token: "fixture" };
    if (path === "/api/jobs") {
      calls.jobs++;
      const requestURL = new URL(route.request().url());
      totalModes.push(requestURL.searchParams.get("total") || "default");
      origins.push(`${requestURL.searchParams.get("lat")},${requestURL.searchParams.get("lng")}`);
      data = { items: [], total: 0 };
    }
    if (path === "/api/network") calls.network++;
    if (path === "/api/notifications") calls.notifications++;
    if (path === "/api/halcon") { calls.halcon++; data = { linked: false }; }
    if (path === "/api/profile") data = { profile: { address: "Mercado", latitude: 23, longitude: -82 } };
    if (path.endsWith("/rating")) data = { average: 0, count: 0, can_rate: false, completed_jobs: 0, rating: null };
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(data) });
  });

  await page.goto("/");
  await expect.poll(() => calls.jobs).toBe(1);
  await expect.poll(() => calls.network).toBe(1);
  await expect.poll(() => calls.notifications).toBe(1);
  await expect.poll(() => calls.halcon).toBe(1);

  await page.clock.fastForward(15_000);
  await expect.poll(() => calls.jobs).toBe(2);
  expect(calls.network).toBe(1);
  expect(calls.notifications).toBe(1);
  expect(calls.halcon).toBe(1);

  await page.clock.fastForward(15_000);
  await expect.poll(() => calls.jobs).toBe(3);
  await expect.poll(() => calls.network).toBe(2);
  expect(calls.notifications).toBe(2);
  expect(calls.halcon).toBe(1);

  await page.clock.fastForward(15_000);
  await expect.poll(() => calls.jobs).toBe(4);
  expect(calls.network).toBe(2);
  expect(calls.notifications).toBe(2);
  expect(calls.halcon).toBe(1);

  await page.clock.fastForward(15_000);
  await expect.poll(() => calls.jobs).toBe(5);
  await expect.poll(() => calls.network).toBe(3);
  expect(calls.notifications).toBe(3);
  expect(calls.halcon).toBe(2);
  expect(totalModes).toEqual(["default", "false", "false", "false", "false"]);
  expect(origins.slice(1)).toEqual(["23,-82", "23,-82", "23,-82", "23,-82"]);
});
