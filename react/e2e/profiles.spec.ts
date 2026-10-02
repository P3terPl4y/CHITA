import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

async function fixture(page: import("@playwright/test").Page, role = "company") {
  let avatar = "", failPhoto = false, failList = false, membership = "none", identity = 3;
  let holdPhoto = false;
  let releasePhoto: (() => void) | undefined;
  const calls: unknown[] = [];
  await page.route("https://tile.openstreetmap.org/**", r => r.fulfill({ contentType: "image/svg+xml", body: '<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"/>' }));
  await page.route("**/api/**", route => {
    const url = new URL(route.request().url());
    let data: unknown = [];
    if (url.pathname === "/api/session") data = { user: { id: identity, name: identity === 3 ? "Ana" : "Nueva cuenta", email: "ana@example.test", role, phone: "", avatar_url: identity === 3 ? avatar : "" }, csrf_token: "fixture" };
    if (url.pathname === "/api/jobs") data = { items: [], total: 0 };
    if (url.pathname === "/api/profile") data = { profile: { address: "Mercado", latitude: 23, longitude: -82 } };
    if (url.pathname === "/api/halcon") data = { linked: false };
    if (url.pathname.endsWith("/rating")) data = { average: 0, count: 0, can_rate: false, completed_jobs: 0, rating: null };
    if (url.pathname === "/api/couriers/directory") {
      if (failList) return route.fulfill({ status: 503, json: { error: "No se pudo cargar el directorio" } });
      const search = url.searchParams.get("search");
      const second = url.searchParams.get("page") === "2";
      data = { items: search === "nadie" ? [] : [{ id: second ? 8 : 7, name: second ? "Luis" : "Elena", avatar_url: "", vehicle_type: "bicycle", average_rating: 4.5, rating_count: 2, membership: second ? "accepted" : membership }], total: search === "nadie" ? 0 : 31 };
    }
    if (url.pathname === "/api/network" && route.request().method() === "POST") {
      calls.push(route.request().postDataJSON()); membership = "pending"; data = { id: 4, status: "pending" };
    }
    if (url.pathname === "/api/profile/avatar") {
      if (failPhoto) return route.fulfill({ status: 503, json: { error: "No se pudo guardar la foto" } });
      avatar = route.request().postDataJSON().avatar; data = { avatar_url: avatar };
      if (holdPhoto) return new Promise<void>(resolve => { releasePhoto = () => { void route.fulfill({ json: data }).then(() => resolve()); }; });
    }
    return route.fulfill({ json: data });
  });
  return { calls, switchIdentity: () => { identity = 9; }, holdPhoto: () => { holdPhoto = true; }, releasePhoto: () => releasePhoto?.(), photoFailure: (value: boolean) => { failPhoto = value; }, listFailure: (value: boolean) => { failList = value; } };
}

test("directorio: puntuación, invitación, paginación, búsqueda y recuperación", async ({ page }) => {
  const state = await fixture(page);
  await page.goto("/");
  await page.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Directorio", exact: true }).click();
  const directory = page.getByRole("region", { name: "Directorio de repartidores" });
  await expect(directory).toContainText("4.5 / 5");
  await directory.getByRole("button", { name: "Invitar a mi red" }).click();
  await expect(directory.getByRole("button", { name: "Invitación enviada" })).toBeDisabled();
  expect(state.calls).toEqual([{ courier_id: 7 }]);
  await expect(directory.getByRole("status")).toContainText("Debe aceptarla");
  await directory.getByRole("button", { name: "Siguiente" }).click();
  await expect(directory.getByRole("heading", { name: "Luis" })).toBeVisible();
  await expect(directory.getByRole("button", { name: "En tu red" })).toBeDisabled();
  await directory.getByLabel("Buscar repartidor por nombre").fill("nadie");
  await directory.getByRole("button", { name: "Buscar", exact: true }).click();
  await expect(directory).toContainText("No encontramos repartidores");
  state.listFailure(true);
  await directory.getByLabel("Buscar repartidor por nombre").fill("");
  await directory.getByRole("button", { name: "Buscar", exact: true }).click();
  await expect(directory.getByRole("alert")).toContainText("No se pudo cargar");
  state.listFailure(false);
  await directory.getByRole("button", { name: "Volver a cargar" }).click();
  await expect(directory.getByRole("heading", { name: "Elena" })).toBeVisible();
  for (const theme of ["light", "dark"]) {
    await page.getByLabel("Apariencia").selectOption(theme);
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
  }
  await page.setViewportSize({ width: 320, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
for (const role of ["company", "courier"]) {
  test(`foto ${role}: guardar, persistir, recuperar error y quitar`, async ({ page }) => {
    const state = await fixture(page, role);
    await page.goto("/");
    const account = async () => page.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Mi cuenta", exact: true }).click();
    await account();
    const photo = page.getByRole("region", { name: "Foto de perfil", exact: true });
    const encoded = await page.evaluate(() => { const c = document.createElement("canvas"); c.width = c.height = 64; const x = c.getContext("2d")!; x.fillStyle = "red"; x.fillRect(0, 0, 64, 64); return c.toDataURL("image/png").split(",")[1]; });
    const file = { name: "perfil.png", mimeType: "image/png", buffer: Buffer.from(encoded, "base64") };
    await photo.getByLabel("Cambiar foto").setInputFiles(file);
    await expect(photo.getByRole("status")).toHaveText("Foto de perfil guardada");
    await expect(photo.locator("img")).toHaveJSProperty("naturalWidth", 128);
    await page.reload(); await account();
    await expect(photo.locator("img")).toBeVisible();
    state.photoFailure(true);
    await photo.getByRole("button", { name: "Quitar foto" }).click();
    await expect(photo.getByRole("alert")).toContainText("No se pudo guardar");
    await expect(photo.locator("img")).toBeVisible();
    state.photoFailure(false);
    await photo.getByRole("button", { name: "Quitar foto" }).click();
    await expect(photo.getByRole("status")).toHaveText("Foto de perfil eliminada");
    await expect(photo.locator("img")).toHaveCount(0);
    await photo.getByLabel("Cambiar foto").setInputFiles({ name: "bad.svg", mimeType: "image/svg+xml", buffer: Buffer.from("<svg/>") });
    await expect(photo.getByRole("alert")).toContainText("Elige una foto");
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
  });
}

test("directorio y foto reales: invitación consentida y perfil persistente", async ({ browser, baseURL }) => {
  test.skip(process.env.CHITA_REAL_DISPATCH !== "1", "requires isolated database");
  if (baseURL !== "http://127.0.0.1:3340") throw Error("isolated server 3340 required");
  const companyContext = await browser.newContext(), courierContext = await browser.newContext();
  try {
    for (const context of [companyContext, courierContext]) await context.route("https://tile.openstreetmap.org/**", r => r.fulfill({ contentType: "image/svg+xml", body: '<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"/>' }));
    const stamp = Date.now();
    for (const [context, role] of [[companyContext, "company"], [courierContext, "courier"]] as const) {
      const session = await (await context.request.get(baseURL + "/api/session")).json();
      const result = await context.request.post(baseURL + "/api/auth/register", { headers: { "X-CSRF-Token": session.csrf_token }, data: { name: `Perfil ${role} ${stamp}`, email: `perfil-${role}-${stamp}@example.test`, password: "ProfileBrowser123!", phone: "12345678", role, address: "Calle de prueba 10", latitude: 23.1, longitude: -82.3, company_name: "Empresa perfiles", vehicle_type: "bicycle" } });
      expect(result.status()).toBe(201);
    }
    const courier = await courierContext.newPage(), company = await companyContext.newPage();
    await courier.goto("/");
    await courier.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Mi cuenta", exact: true }).click();
    const encoded = await courier.evaluate(() => { const c = document.createElement("canvas"); c.width = c.height = 64; c.getContext("2d")!.fillRect(0, 0, 64, 64); return c.toDataURL("image/png").split(",")[1]; });
    await courier.getByRole("region", { name: "Foto de perfil", exact: true }).getByLabel("Cambiar foto").setInputFiles({ name: "real.png", mimeType: "image/png", buffer: Buffer.from(encoded, "base64") });
    await expect(courier.getByText("Foto de perfil guardada", { exact: true })).toBeVisible();
    await courier.reload();
    await expect(courier.locator("header img.user-avatar")).toHaveJSProperty("naturalWidth", 128);
    await company.goto("/");
    await company.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Directorio", exact: true }).click();
    await company.getByLabel("Buscar repartidor por nombre").fill(`Sin resultados ${stamp}`);
    await company.getByRole("button", { name: "Buscar", exact: true }).click();
    await expect(company.getByText("No encontramos repartidores con ese nombre.", { exact: false })).toBeVisible();
    await company.getByLabel("Buscar repartidor por nombre").fill(`Perfil courier ${stamp}`);
    await company.getByRole("button", { name: "Buscar", exact: true }).click();
    const card = company.locator(".courier-card").filter({ hasText: `Perfil courier ${stamp}` });
    await expect(card.locator("img")).toHaveJSProperty("naturalWidth", 128);
    await expect(card).toContainText("Sin calificaciones todavía");
    await card.getByRole("button", { name: "Invitar a mi red" }).click();
    await expect(card.getByRole("button", { name: "Invitación enviada" })).toBeDisabled();
    await courier.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Invitaciones", exact: true }).click();
    await courier.getByRole("button", { name: "Aceptar invitación", exact: true }).click();
    await company.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Mi red", exact: true }).click();
    await company.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Directorio", exact: true }).click();
    await expect(company.locator(".courier-card").filter({ hasText: `Perfil courier ${stamp}` }).getByRole("button", { name: "En tu red" })).toBeDisabled();
  } finally { await companyContext.close(); await courierContext.close(); }
});

test("mapas desmontados ignoran callbacks de tamaño pendientes", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.addInitScript(() => {
    const NativeObserver = window.ResizeObserver;
    const state = window as Window & { lateResizeCalls?: number };
    state.lateResizeCalls = 0;
    window.ResizeObserver = class extends NativeObserver {
      private callback: ResizeObserverCallback;
      constructor(callback: ResizeObserverCallback) { super(callback); this.callback = callback; }
      disconnect() {
        super.disconnect();
        // Deliberately deliver work queued before disconnect, after cleanup.
        queueMicrotask(() => { state.lateResizeCalls!++; this.callback([], this); });
      }
    };
  });
  await fixture(page);
  await page.goto("/");
  const nav = page.getByRole("navigation", { name: "Secciones", exact: true });
  for (let n = 0; n < 3; n++) {
    await nav.getByRole("button", { name: "Mi cuenta", exact: true }).click();
    await expect(page.getByRole("region", { name: "Elegir dirección del perfil", exact: true })).toBeVisible();
    await nav.getByRole("button", { name: "Inicio", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Tu centro de entregas" })).toBeVisible();
  }
  expect(await page.evaluate(() => (window as Window & { lateResizeCalls: number }).lateResizeCalls)).toBeGreaterThanOrEqual(3);
  expect(errors).toEqual([]);
});

test("publicar desde otra página limpia filtros y consulta la primera página", async ({ page }) => {
  await fixture(page);
  let created = false;
  const reads: number[] = [];
  await page.route("**/api/jobs**", route => {
    if (route.request().method() === "POST") { created = true; return route.fulfill({ status: 201, json: { id: 2 } }); }
    const requestedPage = Number(new URL(route.request().url()).searchParams.get("page")); reads.push(requestedPage);
    return route.fulfill({ json: { items: [{ id: created ? 2 : 1, title: created ? "Entrega publicada nueva" : "Entrega anterior", description: "", status: "published", visibility: "public", company_id: 3, assigned_courier_id: null, pickup_address: "Mercado", dropoff_address: "Centro", pickup_lat: 23, pickup_lng: -82, dropoff_lat: 23.1, dropoff_lng: -82.1, pickup_from: "2030-10-02T12:00:00Z", pickup_to: "2030-10-02T13:00:00Z", delivery_from: "2030-10-02T13:00:00Z", delivery_to: "2030-10-02T14:00:00Z", price_cents: 1200, currency: "USD" }], total: 31 } });
  });
  await page.goto("/");
  await page.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Trabajos", exact: true }).click();
  await page.getByRole("button", { name: "Siguiente", exact: true }).click();
  await expect(page.getByText("Página 2 · 31 trabajos", { exact: true })).toBeVisible();
  await page.getByLabel("Buscar en esta página").fill("anterior");
  await page.getByRole("combobox", { name: "Mostrar trabajos por estado", exact: true }).selectOption("completed");
  await page.getByRole("button", { name: "Publicar trabajo", exact: true }).click();
  await page.getByLabel("Título del trabajo").fill("Entrega publicada nueva");
  await page.getByLabel("Dirección de entrega", { exact: true }).fill("Centro");
  await page.getByLabel("Latitud de entrega").fill("23.1");
  await page.getByLabel("Longitud de entrega").fill("-82.1");
  for (const [label, value] of [["Recogida desde", "2030-10-02T12:00"], ["Recogida hasta", "2030-10-02T13:00"], ["Entrega desde", "2030-10-02T13:00"], ["Entrega hasta", "2030-10-02T14:00"]]) await page.getByLabel(label).fill(value);
  await page.getByLabel("Tarifa acordada").fill("12");
  const before = reads.length;
  await page.getByRole("button", { name: "Publicar trabajo", exact: true }).click();
  await expect(page.locator(".jobcard")).toContainText("Entrega publicada nueva");
  await expect(page.getByLabel("Buscar en esta página")).toHaveValue("");
  await expect(page.getByRole("combobox", { name: "Mostrar trabajos por estado", exact: true })).toHaveValue("all");
  await expect(page.getByText("Página 1 · 31 trabajos", { exact: true })).toBeVisible();
  expect(reads.slice(before).length).toBeGreaterThan(0);
  expect(reads.slice(before).every(n => n === 1)).toBe(true);
});

test("una foto pendiente no modifica la siguiente cuenta", async ({ page }) => {
  const state = await fixture(page);
  await page.goto("/");
  await page.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Mi cuenta", exact: true }).click();
  const encoded = await page.evaluate(() => { const c = document.createElement("canvas"); c.width = c.height = 64; c.getContext("2d")!.fillRect(0, 0, 64, 64); return c.toDataURL("image/png").split(",")[1]; });
  state.holdPhoto();
  const request = page.waitForRequest(r => r.url().endsWith("/api/profile/avatar"));
  await page.getByRole("region", { name: "Foto de perfil", exact: true }).getByLabel("Cambiar foto").setInputFiles({ name: "tardia.png", mimeType: "image/png", buffer: Buffer.from(encoded, "base64") });
  await request;
  state.switchIdentity();
  await page.evaluate(() => window.dispatchEvent(new Event("chita-session-ended")));
  await page.getByLabel("Correo electrónico", { exact: true }).fill("otra@example.test");
  await page.getByLabel("Contraseña", { exact: true }).fill("ProfileBrowser123!");
  await page.locator("form").getByRole("button", { name: "Entrar", exact: true }).click();
  await expect(page.locator("header")).toContainText("Nueva cuenta");
  const response = page.waitForResponse(r => r.url().endsWith("/api/profile/avatar"));
  state.releasePhoto(); await response;
  await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
  await expect(page.locator("header img.user-avatar")).toHaveCount(0);
});
