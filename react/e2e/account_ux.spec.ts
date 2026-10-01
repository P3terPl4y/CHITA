import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
for (const role of ["company", "courier"]) {
  test(`cuenta ${role}: tocar mapa y guardar, recuperar errores y descartar`, async ({
    page,
  }) => {
    let profile = {
      address: "Dirección guardada",
      latitude: 23.1,
      longitude: -82.3,
    };
    let fail = false;
    let saved = 0;
    await page.route("https://tile.openstreetmap.org/**", (r) =>
      r.fulfill({
        contentType: "image/png",
        body: Buffer.from(
          "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg==",
          "base64",
        ),
      }),
    );
    await page.route("**/api/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      let data: unknown = [];
      if (path === "/api/session")
        data = {
          user: {
            id: 3,
            name: "Cuenta de prueba",
            email: "cuenta@example.test",
            role,
            phone: "",
          },
          csrf_token: "fixture",
        };
      else if (path === "/api/jobs") data = { items: [], total: 0 };
      else if (path === "/api/profile") data = { profile };
      else if (path === "/api/halcon") data = { linked: false };
      else if (path.endsWith("/rating"))
        data = {
          average: 0,
          count: 0,
          can_rate: false,
          completed_jobs: 0,
          rating: null,
        };
      else if (path === "/api/maps/reverse") {
        await new Promise((r) => setTimeout(r, 250));
        data = { address: "Dirección del punto nuevo" };
      } else if (path === "/api/profile/location") {
        if (fail)
          return route.fulfill({
            status: 503,
            contentType: "application/json",
            body: JSON.stringify({
              error: "No se pudo guardar. Inténtalo otra vez.",
            }),
          });
        const value = route.request().postDataJSON();
        expect(Object.keys(value).sort()).toEqual([
          "address",
          "latitude",
          "longitude",
        ]);
        expect(Number.isFinite(value.latitude)).toBe(true);
        expect(Number.isFinite(value.longitude)).toBe(true);
        profile = value;
        saved++;
        return route.fulfill({ status: 204 });
      }
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify(data),
      });
    });
    await page.goto("/");
    await page
      .getByRole("navigation", { name: "Secciones", exact: true })
      .getByRole("button", { name: "Mi cuenta", exact: true })
      .click();
    const editor = page.getByRole("region", {
      name: "Editar ubicación de la cuenta",
    });
    const map = editor.getByRole("region", {
      name: "Elegir dirección del perfil",
    });
    const save = editor.getByRole("button", {
      name: "Guardar ubicación",
      exact: true,
    });
    const address = editor.getByLabel("Dirección seleccionada", {
      exact: true,
    });
    await expect(map).toBeVisible();
    await expect(editor.getByRole("spinbutton")).toHaveCount(0);
    await expect(save).toBeDisabled();
    await map.click({ position: { x: 160, y: 130 } });
    await expect(save).toBeDisabled();
    await expect(address).toHaveValue("Dirección del punto nuevo");
    await expect(save).toBeEnabled();
    await save.click();
    await expect(
      editor.getByRole("status").filter({ hasText: "Ubicación guardada" }),
    ).toBeVisible();
    expect(saved).toBe(1);
    await expect(save).toBeDisabled();
    fail = true;
    await address.fill("Referencia corregida");
    await save.click();
    await expect(editor.getByRole("alert")).toContainText("No se pudo guardar");
    await expect(address).toHaveValue("Referencia corregida");
    await expect(save).toBeEnabled();
    await editor.getByRole("button", { name: "Descartar cambios" }).click();
    await expect(address).toHaveValue("Dirección del punto nuevo");
    await expect(save).toBeDisabled();
    for (const theme of ["light", "dark"]) {
      await page.getByLabel("Apariencia").selectOption(theme);
      for (const width of [320, 390, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        ).toBe(true);
      }
      expect(
        (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
      ).toEqual([]);
    }
    if (role === "company")
      await page.screenshot({
        path: "/tmp/chita-account-dark.png",
        fullPage: true,
      });
  });
}
