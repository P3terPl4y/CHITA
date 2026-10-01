import { test, expect, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
const tileImage = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg==",
  "base64",
);
test.beforeEach(async ({ context }) => {
  await context.route("https://tile.openstreetmap.org/**", (route) =>
    route.fulfill({ status: 200, contentType: "image/png", body: tileImage }),
  );
});
async function register(p: Page, role: string, email: string) {
  await p.goto("/registro");
  await p.getByLabel("Quiero usar CHITA como").selectOption(role);
  await p
    .getByLabel("Tu nombre")
    .fill(role === "company" ? "Empresa Prueba" : "Repartidor Prueba");
  await p.getByLabel("Teléfono").fill("12345678");
  if (role === "company")
    await p.getByLabel("Nombre de la empresa").fill("Comercio Prueba");
  await p.getByLabel("Correo electrónico", { exact: true }).fill(email);
  await p.getByLabel("Contraseña", { exact: true }).fill("StrongTest123!");
  await p.getByLabel("Dirección de referencia").fill("Calle central 10");
  await p.getByLabel("Latitud", { exact: true }).fill("23.11345");
  await p.getByLabel("Longitud", { exact: true }).fill("-82.3667");
  await p
    .getByRole("button", { name: "Crear cuenta", exact: true })
    .last()
    .click();
  await expect(p.getByRole("button", { name: "Salir" })).toBeVisible();
}
async function fits(p: Page) {
  expect(
    await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
  ).toBe(true);
}
test("login y registro son accesibles y adaptables", async ({ page }) => {
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Coordina la siguiente entrega." }),
    ).toBeVisible();
    await fits(page);
    await page
      .getByRole("link", { name: "Crear cuenta", exact: true })
      .first()
      .click();
    await fits(page);
  }
  const result = await new AxeBuilder({ page }).analyze();
  expect(result.violations.map((v) => ({ id: v.id, help: v.help }))).toEqual(
    [],
  );
});
test("empresa invita, publica y confirma; repartidor acepta y entrega", async ({
  browser,
}) => {
  const ctx1 = await browser.newContext(),
    ctx2 = await browser.newContext();
  await ctx1.route("https://tile.openstreetmap.org/**", (route) =>
    route.fulfill({ status: 200, contentType: "image/png", body: tileImage }),
  );
  await ctx2.route("https://tile.openstreetmap.org/**", (route) =>
    route.fulfill({ status: 200, contentType: "image/png", body: tileImage }),
  );
  const company = await ctx1.newPage(),
    courier = await ctx2.newPage();
  const stamp = Date.now();
  const ce = "business-" + stamp + "@example.test",
    de = "courier-" + stamp + "@example.test";
  await register(company, "company", ce);
  await register(courier, "courier", de);
  await company.getByRole("button", { name: "Mi red", exact: true }).click();
  await company.getByLabel("Correo del repartidor registrado").fill(de);
  await company.getByRole("button", { name: "Invitar", exact: true }).click();
  await expect(
    company.getByText("Invitación enviada", { exact: true }),
  ).toBeVisible();
  await courier
    .getByRole("button", { name: "Invitaciones", exact: true })
    .click();
  await courier.reload();
  await courier
    .getByRole("button", { name: "Invitaciones", exact: true })
    .click();
  await courier
    .getByRole("button", { name: "Aceptar invitación", exact: true })
    .click();
  await expect(
    courier.getByText("Ahora puedes ver los trabajos de esta empresa"),
  ).toBeVisible();
  await company
    .getByRole("button", { name: "Publicar trabajo", exact: true })
    .click();
  await company.getByLabel("Título del trabajo").fill("Paquete de prueba E2E");
  await company
    .getByLabel("Descripción", { exact: true })
    .fill("Entrega de una caja");
  await company
    .getByLabel("Dirección de entrega", { exact: true })
    .fill("Calle destino 20");
  await company.getByLabel("Latitud de entrega").fill("23.12");
  await company.getByLabel("Longitud de entrega").fill("-82.37");
  const local = (hours: number) => {
    const d = new Date(Date.now() + hours * 3600000);
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
    return d.toISOString().slice(0, 16);
  };
  await company.getByLabel("Recogida desde").fill(local(1));
  await company.getByLabel("Recogida hasta").fill(local(2));
  await company.getByLabel("Entrega desde").fill(local(2));
  await company.getByLabel("Entrega hasta").fill(local(3));
  await company.route("**/api/maps/reverse?**", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ address: "Calle destino elegida en el mapa" }),
    }),
  );
  await company
    .locator("fieldset")
    .last()
    .locator(".map")
    .click({ position: { x: 100, y: 100 } });
  await expect(company.getByLabel("Latitud de entrega")).not.toHaveValue(
    "23.12",
  );
  await expect(
    company.getByLabel("Dirección de entrega", { exact: true }),
  ).toHaveValue("Calle destino elegida en el mapa");
  await company
    .locator("fieldset")
    .last()
    .getByRole("button", { name: "Cerrar mapa" })
    .click();
  await company.getByLabel("Tarifa acordada").fill("12.34");
  await company
    .getByRole("button", { name: "Publicar trabajo", exact: true })
    .last()
    .click();
  await expect(
    company.getByText("Trabajo publicado para tu red"),
  ).toBeVisible();
  await courier.reload();
  await courier
    .getByRole("button")
    .filter({ hasText: "Paquete de prueba E2E" })
    .click();
  await courier
    .getByRole("button", { name: "Aceptar trabajo", exact: true })
    .click();
  await courier
    .getByRole("button", { name: "Confirmar recogida", exact: true })
    .click();
  await courier
    .getByRole("button", { name: "Estoy en el punto de entrega", exact: true })
    .click();
  await courier.getByLabel("Nota de entrega (opcional)").fill("Recibido");
  await courier
    .getByRole("button", { name: "Notificar entrega a la empresa" })
    .click();
  await expect(courier.locator(".detail .badge")).toHaveText(
    "Pendiente de confirmación",
  );
  await company.reload();
  await company
    .getByRole("button")
    .filter({ hasText: "Paquete de prueba E2E" })
    .click();
  await company
    .getByRole("button", { name: "Confirmar entrega completada" })
    .click();
  await expect(company.locator(".detail .badge")).toHaveText("Completado");
  for (const width of [320, 390, 768, 1440]) {
    await company.setViewportSize({ width, height: 900 });
    await fits(company);
    await courier.setViewportSize({ width, height: 900 });
    await fits(courier);
  }
  const result = await new AxeBuilder({ page: company }).analyze();
  expect(result.violations.map((v) => ({ id: v.id, help: v.help }))).toEqual(
    [],
  );
  await company.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() => company.locator(".leaflet-tile-loaded").count())
    .toBeGreaterThan(0);
  await company.screenshot({
    path: "test-results/company-mobile.png",
    fullPage: true,
  });
  await company.getByRole("button", { name: "Salir" }).click();
  await expect(
    company.getByRole("heading", { name: "Coordina la siguiente entrega." }),
  ).toBeVisible();
  await ctx1.close();
  await ctx2.close();
});

test("trabajo público permite aceptar sin invitación", async ({ browser }) => {
  const c1 = await browser.newContext(),
    c2 = await browser.newContext();
  for (const context of [c1, c2])
    await context.route("https://tile.openstreetmap.org/**", (route) =>
      route.fulfill({ status: 200, contentType: "image/png", body: tileImage }),
    );
  try {
    const company = await c1.newPage(),
      courier = await c2.newPage();
    const stamp = Date.now();
    await register(company, "company", `public-company-${stamp}@example.test`);
    await register(courier, "courier", `public-courier-${stamp}@example.test`);
    await company
      .getByRole("button", { name: "Publicar trabajo", exact: true })
      .click();
    await company.getByLabel("Visibilidad del trabajo").selectOption("public");
    const title = `Trabajo público ${stamp}`;
    await company.getByLabel("Título del trabajo").fill(title);
    await company
      .getByLabel("Dirección de entrega", { exact: true })
      .fill("Destino público 10");
    await company.getByLabel("Latitud de entrega").fill("23.12");
    await company.getByLabel("Longitud de entrega").fill("-82.37");
    const local = (h: number) => {
      const d = new Date(Date.now() + h * 3600000);
      d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
      return d.toISOString().slice(0, 16);
    };
    await company.getByLabel("Recogida desde").fill(local(1));
    await company.getByLabel("Recogida hasta").fill(local(2));
    await company.getByLabel("Entrega desde").fill(local(2));
    await company.getByLabel("Entrega hasta").fill(local(3));
    await company.getByLabel("Tarifa acordada").fill("10.50");
    await company
      .getByRole("button", { name: "Publicar trabajo", exact: true })
      .last()
      .click();
    await expect(
      company
        .getByRole("status")
        .filter({ hasText: "Trabajo público publicado" }),
    ).toBeVisible();
    await courier.reload();
    const card = courier.locator(".jobcard").filter({ hasText: title });
    await expect(card).toContainText("Público");
    await expect(card).toContainText("km hasta recogida");
    await card.click();
    await courier
      .getByRole("button", { name: "Aceptar trabajo", exact: true })
      .click();
    await expect(courier.locator(".detail .badge")).toHaveText("Aceptado");
    await courier
      .getByRole("button", { name: "Vincular con HALCON", exact: true })
      .first()
      .click();
    await expect(courier.getByLabel("Correo de HALCON")).toBeVisible();
  } finally {
    await c1.close();
    await c2.close();
  }
});
