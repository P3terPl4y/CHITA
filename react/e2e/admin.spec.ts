import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test("panel administrativo accesible en móvil y escritorio", async ({
  page,
}) => {
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    expect(["PUT", "DELETE"]).not.toContain(route.request().method());
    const user = {
      id: 99,
      name: "Administración",
      email: "admin@example.test",
      phone: "",
      role: "admin",
    };
    const content =
      path === "/api/session"
        ? { user, csrf_token: "fixture" }
        : path === "/api/admin/overview"
          ? { companies: 12, couriers: 30, jobs: 42, pending_confirmation: 3 }
          : path === "/api/admin/audits"
            ? {
                items: [
                  {
                    id: 1,
                    created_at: new Date().toISOString(),
                    actor_user_id: 99,
                    entity: "companies",
                    entity_id: 1,
                    action: "create",
                    reason: "Alta de empresa verificada",
                  },
                ],
                total: 1,
                page: 1,
              }
            : {
                items: [
                  {
                    id: 1,
                    user_id: 2,
                    name: "Ana Pérez",
                    email: "ana@example.test",
                    phone: "12345678",
                    address: "Avenida Central 15",
                    latitude: 23.1,
                    longitude: -82.3,
                    company_name: path.includes("companies")
                      ? "Mercado Central"
                      : "",
                    vehicle_type: "bicycle",
                    enabled: true,
                    archived: false,
                    version: 1,
                  },
                ],
                total: 1,
                page: 1,
              };
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(content),
    });
  });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Control de la plataforma" }),
  ).toBeVisible();
  for (const theme of ["light", "dark"]) {
    await page.getByLabel("Apariencia").selectOption(theme);
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    }
    const result = await new AxeBuilder({ page }).analyze();
    expect(
      result.violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => n.target),
      })),
    ).toEqual([]);
  }
  await page.setViewportSize({ width: 390, height: 900 });
  await page.getByRole("button", { name: "Menú", exact: true }).click();
  await page
    .getByRole("navigation", { name: "Secciones móviles" })
    .getByRole("button", { name: "Empresas", exact: true })
    .click();
  await expect(
    page.getByRole("dialog", { name: "Tu plataforma" }),
  ).not.toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Mercado Central" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Editar", exact: true }).click();
  await expect(page.getByLabel("Nombre del usuario")).toHaveValue("Ana Pérez");
  for (const theme of ["light", "dark"]) {
    await page.getByLabel("Apariencia").selectOption(theme);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    expect(
      (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => n.target),
      })),
    ).toEqual([]);
  }
  await page.getByRole("button", { name: "Volver a la lista" }).click();
  await page.getByRole("button", { name: "Archivar", exact: true }).click();
  await expect(
    page.getByRole("dialog", { name: /Archivar: Mercado Central/ }),
  ).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
  ).toEqual([]);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: /Archivar: Mercado Central/ }),
  ).not.toBeVisible();
});

test("CRUD de empresa, repartidor y trabajo desde el navegador", async ({
  page,
}) => {
  test.skip(
    process.env.CHITA_ADMIN_BROWSER !== "1",
    "requires isolated database and local admin fixture",
  );
  await page.route("**/api/maps/reverse?*", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ address: "Dirección desde el mapa" }),
    }),
  );
  const stamp = Date.now();
  await page.goto("/entrar");
  await page
    .getByLabel("Correo electrónico", { exact: true })
    .fill("admin-browser@chita.test");
  await page.getByLabel("Contraseña", { exact: true }).fill("BrowserAdmin123!");
  await page
    .getByRole("button", { name: "Entrar", exact: true })
    .last()
    .click();
  await expect(
    page.getByRole("heading", { name: "Control de la plataforma" }),
  ).toBeVisible();
  async function nav(label: string) {
    await page
      .getByRole("navigation", { name: "Secciones", exact: true })
      .getByRole("button", { name: label, exact: true })
      .click();
  }
  for (const entity of ["empresa", "repartidor"]) {
    await nav(entity === "empresa" ? "Empresas" : "Repartidores");
    await page.getByRole("button", { name: `Crear ${entity}` }).click();
    await page
      .getByLabel("Nombre del usuario")
      .fill(`Prueba ${entity} ${stamp}`);
    await page
      .getByLabel("Correo", { exact: true })
      .fill(`${entity}-${stamp}@browser.chita.test`);
    await page.getByLabel("Teléfono", { exact: true }).fill("12345678");
    if (entity === "empresa")
      await page
        .getByLabel("Nombre de empresa")
        .fill(`Empresa Browser ${stamp}`);
    await page
      .getByLabel("Dirección", { exact: true })
      .fill("Calle validación 10");
    await page.getByLabel("Latitud", { exact: true }).fill("23.1");
    await page.getByLabel("Longitud", { exact: true }).fill("-82.3");
    await page
      .getByRole("button", { name: "Colocar pin en el centro", exact: true })
      .click();
    await expect(page.getByLabel("Dirección", { exact: true })).toHaveValue(
      "Dirección desde el mapa",
    );
    await page
      .getByLabel("Contraseña inicial", { exact: true })
      .fill("UserBrowser123!");
    await page
      .getByLabel("Motivo del cambio")
      .fill("Creación desde prueba de navegador");
    await page.getByRole("button", { name: "Guardar cambios" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: "Cambios guardados" }),
    ).toBeVisible();
    await page
      .getByLabel("Buscar por nombre o correo")
      .fill(`${entity}-${stamp}`);
    const row = page
      .locator(".admin-record")
      .filter({ hasText: `${entity}-${stamp}@browser.chita.test` });
    await expect(row).toHaveCount(1);
    await row.getByRole("button", { name: "Editar", exact: true }).click();
    await page.getByLabel("Nombre del usuario").fill(`Editado ${entity}`);
    await page
      .getByLabel("Motivo del cambio")
      .fill("Edición desde prueba de navegador");
    await page.getByRole("button", { name: "Guardar cambios" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: "Cambios guardados" }),
    ).toBeVisible();
    if (entity === "repartidor") {
      await row.getByRole("button", { name: "Archivar", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: /Archivar:/ });
      await dialog
        .getByLabel("Motivo", { exact: true })
        .fill("Archivo desde prueba de navegador");
      await dialog.getByRole("button", { name: "Confirmar: Archivar" }).click();
      await expect(dialog).not.toBeVisible();
      await page
        .getByRole("combobox", { name: "Estado", exact: true })
        .selectOption("archived");
      await expect(row).toHaveCount(1);
      await row.getByRole("button", { name: "Restaurar", exact: true }).click();
      const restore = page.getByRole("dialog", { name: /Restaurar:/ });
      await restore
        .getByLabel("Motivo", { exact: true })
        .fill("Restauración desde navegador");
      await restore
        .getByRole("button", { name: "Confirmar: Restaurar" })
        .click();
      await page
        .getByRole("combobox", { name: "Estado", exact: true })
        .selectOption("disabled");
      await expect(row).toHaveCount(1);
    }
  }
  await nav("Trabajos");
  await page.getByRole("button", { name: "Crear trabajo" }).click();
  await page
    .getByLabel("Buscar empresa activa")
    .fill(`Empresa Browser ${stamp}`);
  await expect(
    page
      .getByLabel("Empresa del trabajo")
      .locator("option")
      .filter({ hasText: `Empresa Browser ${stamp}` }),
  ).toHaveCount(1);
  const value = await page
    .getByLabel("Empresa del trabajo")
    .locator("option")
    .filter({ hasText: `Empresa Browser ${stamp}` })
    .getAttribute("value");
  await page.getByLabel("Empresa del trabajo").selectOption(value!);
  await page
    .getByLabel("Título", { exact: true })
    .fill(`Entrega Browser ${stamp}`);
  await page
    .getByLabel("Dirección de recogida", { exact: true })
    .fill("Origen 1");
  await page
    .getByLabel("Dirección de entrega", { exact: true })
    .fill("Destino 2");
  for (const label of ["Latitud de recogida", "Latitud de entrega"])
    await page.getByLabel(label).fill("23.1");
  for (const label of ["Longitud de recogida", "Longitud de entrega"])
    await page.getByLabel(label).fill("-82.3");
  await page
    .getByRole("button", { name: "Colocar pin en el centro", exact: true })
    .first()
    .click();
  await expect(
    page.getByLabel("Dirección de recogida", { exact: true }),
  ).toHaveValue("Dirección desde el mapa");
  await page.getByLabel("Tarifa", { exact: true }).fill("12.50");
  await page
    .getByLabel("Motivo del cambio")
    .fill("Publicación desde prueba de navegador");
  await page.getByRole("button", { name: "Guardar cambios" }).click();
  await expect(
    page.getByRole("status").filter({ hasText: "Cambios guardados" }),
  ).toBeVisible();
  await page
    .getByLabel("Buscar por nombre o correo")
    .fill(`Entrega Browser ${stamp}`);
  const row = page
    .locator(".admin-record")
    .filter({ hasText: `Entrega Browser ${stamp}` });
  await row.getByRole("button", { name: "Ver detalles" }).click();
  await expect(page.locator(".admin-detail")).toContainText("1250");
  await page.getByRole("button", { name: "Cerrar detalles" }).click();
  await row.getByRole("button", { name: "Cancelar trabajo" }).click();
  const dialog = page.getByRole("dialog", { name: /Cancelar trabajo:/ });
  await dialog
    .getByLabel("Motivo", { exact: true })
    .fill("Cancelación desde navegador");
  await dialog
    .getByRole("button", { name: "Confirmar: Cancelar trabajo" })
    .click();
  await expect(row).toContainText("Cancelado");
  await nav("Historial");
  await expect(page.locator(".admin-audit").first()).toContainText(
    "Cancelación desde navegador",
  );
});
