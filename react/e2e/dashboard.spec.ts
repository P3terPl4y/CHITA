import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
for (const role of ["company", "courier"]) {
  test(`${role}: inicio, filtros y navegación móvil`, async ({ page }) => {
    await page.route("**/api/**", route => {
      const path = new URL(route.request().url()).pathname;
      let data: unknown = [];
      if (path === "/api/session") data = { user: { id: 3, name: "Ana", email: "ana@example.test", role, phone: "" }, csrf_token: "fixture" };
      if (path === "/api/jobs") data = { items: [{ id: 1, title: "Entrega de flores", description: "", status: "published", visibility: "public", company_id: 3, assigned_courier_id: null, pickup_address: "Mercado", dropoff_address: "Centro", pickup_lat: 23, pickup_lng: -82, dropoff_lat: 23.1, dropoff_lng: -82.1, pickup_from: "2030-10-02T12:00:00Z", pickup_to: "2030-10-02T13:00:00Z", delivery_from: "2030-10-02T13:00:00Z", delivery_to: "2030-10-02T14:00:00Z", price_cents: 1200, currency: "USD" }], total: 31 };
      if (path === "/api/profile") data = { profile: { address: "Mercado", latitude: 23, longitude: -82 } };
      if (path === "/api/halcon") data = { linked: false };
      return route.fulfill({ contentType: "application/json", body: JSON.stringify(data) });
    });
    await page.goto("/");
    const nav = page.getByRole("navigation", { name: "Secciones", exact: true });
    await nav.getByRole("button", { name: "Inicio", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Tu centro de entregas" })).toBeVisible();
    await expect(page.getByText("Resumen de la página 1 · 1 de 31 trabajos.", { exact: false })).toBeVisible();
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
    await page.getByLabel("Apariencia").selectOption("dark");
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
    await page.screenshot({ path: `/tmp/chita-dashboard-desktop-${role}.png`, fullPage: true });
    if (role === "company") {
      await page.locator(".task-grid").getByRole("button", { name: /Crear una entrega/ }).click();
      await expect(page.getByRole("heading", { name: "Nueva entrega", exact: true })).toBeVisible();
      await expect(page.getByLabel("Título del trabajo")).toBeVisible();
    }
    await nav.getByRole("button", { name: "Guía de uso", exact: true }).click();
    await expect(page.locator(".guide-steps > li")).toHaveCount(5);
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
    if (role === "company") {
      await page.getByRole("button", { name: "Crear una entrega" }).click();
      await page.getByLabel("Título del trabajo").fill("Borrador conservado");
      await nav.getByRole("button", { name: "Mi red", exact: true }).click();
      await page.getByRole("button", { name: "Publicar trabajo", exact: true }).click();
      await expect(page.getByLabel("Título del trabajo")).toHaveValue("Borrador conservado");
    }
    await nav.getByRole("button", { name: "Trabajos", exact: true }).click();
    await expect(page.getByRole("region", { name: "Detalles del trabajo" })).toBeHidden();
    await page.getByLabel("Buscar en esta página").fill("inexistente");
    await expect(page.getByText("No hay coincidencias en esta página.", { exact: false })).toBeVisible();
    await page.getByRole("button", { name: "Limpiar filtros" }).click();
    await expect(page.locator(".jobcard")).toHaveCount(1);
    await page.getByRole("combobox", { name: "Mostrar trabajos por estado", exact: true }).selectOption("completed");
    await expect(page.locator(".jobcard")).toHaveCount(0);
    await page.setViewportSize({ width: 320, height: 844 });
    await page.getByRole("button", { name: "Menú", exact: true }).click();
    await page.getByRole("navigation", { name: "Secciones móviles" }).getByRole("button", { name: "Inicio", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeHidden();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: `/tmp/chita-dashboard-${role}.png`, fullPage: true });
  });
}
