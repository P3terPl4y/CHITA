import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

function luminance(color: string) {
  const rgb = color.match(/rgba?\(([^)]+)\)/)?.[1];
  const values = rgb
    ? rgb.split(/[\s,\/]+/).filter(Boolean).slice(0, 3).map(Number)
    : color.match(/^#([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i)?.slice(1).map((pair) => parseInt(pair, 16)) ?? [];
  const linear = values.map((channel) => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2];
}

function contrast(foreground: string, background: string) {
  const [lighter, darker] = [luminance(foreground), luminance(background)].sort((a, b) => b - a);
  return (lighter + 0.05) / (darker + 0.05);
}

for (const role of ["company", "courier"] as const) {
  test(`identidad y estructura del panel ${role} en escritorio, móvil y ambos temas`, async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.route("**/api/**", (route) => {
      const path = new URL(route.request().url()).pathname;
      let data: unknown = [];
      if (path === "/api/session") data = {
        user: { id: 3, name: "Ana Torres", email: "ana@example.test", role, phone: "" },
        csrf_token: "design-fixture",
      };
      if (path === "/api/jobs") data = { items: [], total: 0 };
      if (path === "/api/profile") data = { profile: { address: "Centro", latitude: 23, longitude: -82 } };
      if (path === "/api/halcon") data = { linked: false };
      return route.fulfill({ contentType: "application/json", body: JSON.stringify(data) });
    });
    await page.goto("/");
    await page.getByRole("navigation", { name: "Secciones", exact: true })
      .getByRole("button", { name: "Inicio", exact: true }).click();
    await expect(page.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button")).toHaveCount(role === "company" ? 9 : 8);
    await expect(page.getByRole("heading", { name: "Tu centro de entregas" })).toBeVisible();

    for (const theme of ["light", "dark"] as const) {
      await page.getByLabel("Apariencia").selectOption(theme);
      const colors = await page.evaluate(() => {
        const shell = document.querySelector<HTMLElement>(".dashboard-shell")!;
        const nav = document.querySelector<HTMLElement>('.desktop-nav [aria-current="page"]')!;
        const primary = document.querySelector<HTMLElement>(".heading .primary, .heading > button")!;
        const firstTask = document.querySelector<HTMLElement>(".task-grid button:first-child")!;
        const desktopNav = document.querySelector<HTMLElement>(".desktop-nav")!;
        return {
          brand: getComputedStyle(shell).getPropertyValue("--brand").trim(),
          accent: getComputedStyle(shell).getPropertyValue("--accent").trim(),
          navBackground: getComputedStyle(nav).backgroundColor,
          navText: getComputedStyle(nav).color,
          desktopNavBackground: getComputedStyle(desktopNav).backgroundColor,
          inactiveNavBackground: getComputedStyle(desktopNav.querySelector("button:not([aria-current])")!).backgroundColor,
          inactiveNavText: getComputedStyle(desktopNav.querySelector("button:not([aria-current])")!).color,
          ctaBackground: getComputedStyle(primary).backgroundColor,
          ctaText: getComputedStyle(primary).color,
          ctaInk: getComputedStyle(shell).getPropertyValue("--cta-ink").trim(),
          taskBackground: getComputedStyle(firstTask).backgroundImage.match(/rgb\([^)]+\)/)?.[0] ?? "",
          taskText: getComputedStyle(firstTask).color,
          headerBackground: getComputedStyle(document.querySelector(".dashboard-shell header")!).backgroundColor,
          columns: getComputedStyle(document.querySelector(".dashboard-home")!).gridTemplateColumns,
          heroColumnEnd: getComputedStyle(document.querySelector(".start-panel")!).gridColumnEnd,
        };
      });
      expect(colors.brand).toMatch(/^#(?:087a55|59d99c)$/);
      expect(colors.accent).toMatch(/^#(?:c7f36a|d2ff75)$/);
      expect(colors.ctaInk.toLowerCase()).toBe("#d97706");
      expect(colors.taskBackground).not.toBe("");
      expect(colors.headerBackground).toBe("rgb(9, 28, 20)");
      expect(colors.desktopNavBackground).toBe("rgb(13, 35, 26)");
      expect(colors.inactiveNavBackground).not.toBe("rgb(255, 255, 255)");
      const navMetrics = await page.locator(".desktop-nav").evaluate((element) => {
        const style = getComputedStyle(element);
        const rect = element.getBoundingClientRect();
        return { height: rect.height, position: style.position, top: style.top, bottom: style.bottom, display: style.display, overflow: style.overflowY };
      });
      expect(navMetrics.height, JSON.stringify(navMetrics)).toBeGreaterThan(500);
      expect(colors.heroColumnEnd).toBe("-1");
      expect(colors.columns.split(" ")).toHaveLength(2);
      expect(contrast(colors.navText, colors.navBackground)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(colors.inactiveNavText, colors.inactiveNavBackground)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(colors.ctaText, colors.ctaBackground)).toBeGreaterThanOrEqual(4.5);
      expect(colors.ctaText).toBe("rgb(217, 119, 6)");
      expect(contrast(colors.taskText, colors.taskBackground)).toBeGreaterThanOrEqual(4.5);
      expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
      await page.screenshot({ path: `/tmp/chita-panel-${role}-${theme}.png` });
    }

    await page.setViewportSize({ width: 390, height: 844 });
    const mobile = await page.evaluate(() => ({
      columns: getComputedStyle(document.querySelector(".dashboard-home")!).gridTemplateColumns,
      taskColumns: getComputedStyle(document.querySelector(".task-grid")!).gridTemplateColumns,
      scrollWidth: document.documentElement.scrollWidth,
      width: innerWidth,
    }));
    expect(mobile.columns.split(" ")).toHaveLength(1);
    expect(mobile.taskColumns.split(" ")).toHaveLength(1);
    expect(mobile.scrollWidth).toBeLessThanOrEqual(mobile.width);
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
  });
}

test("la landing mantiene la identidad verde y CTA legibles en móvil y ambos temas", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.route("**/api/**", route => {
    const path = new URL(route.request().url()).pathname;
    const data = path === "/api/session" ? { user: null, csrf_token: "" } : [];
    return route.fulfill({ contentType: "application/json", body: JSON.stringify(data) });
  });
  await page.goto("/");
  await expect(page.getByRole("heading", { name: /Publica\. Reparte\./ })).toBeVisible();

  for (const theme of ["light", "dark"] as const) {
    await page.getByLabel("Apariencia").selectOption(theme);
    const design = await page.evaluate(() => {
      const root = getComputedStyle(document.documentElement);
      const action = document.querySelector<HTMLElement>(".landing-hero .cta-yellow")!;
      const secondaryAction = document.querySelector<HTMLElement>(".landing-hero .cta-outline")!;
      return {
        brand: root.getPropertyValue("--brand").trim(),
        accent: root.getPropertyValue("--accent").trim(),
        actionText: getComputedStyle(action).color,
        actionBackground: getComputedStyle(action).backgroundColor,
        secondaryText: getComputedStyle(secondaryAction).color,
        heroBackground: getComputedStyle(document.querySelector(".landing-hero")!).backgroundImage,
        visualBackground: getComputedStyle(document.querySelector(".landing-visual")!).backgroundImage,
      };
    });
    expect(design.brand).toMatch(/^#(?:087a55|59d99c)$/);
    expect(design.accent).toMatch(/^#(?:c7f36a|d2ff75)$/);
    expect(design.actionText).toBe("rgb(217, 119, 6)");
    expect(design.secondaryText).toBe("rgb(217, 119, 6)");
    expect(contrast(design.actionText, design.actionBackground)).toBeGreaterThanOrEqual(4.5);
    expect(design.heroBackground).toContain("rgb(");
    expect(design.visualBackground).toContain("linear-gradient");
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
    await page.screenshot({ path: `/tmp/chita-landing-${theme}.png` });
  }

  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
});
