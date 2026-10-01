import { test, expect, type Page } from "@playwright/test";
const tile =
  '<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><rect width="256" height="256" fill="#dbe8ed"/><path d="M0 128H256M128 0V256" stroke="#8b9ba3" stroke-width="3"/></svg>';
test.beforeEach(async ({ context }) => {
  await context.route("https://tile.openstreetmap.org/**", (r) =>
    r.fulfill({ contentType: "image/svg+xml", body: tile }),
  );
});
async function account(page: Page) {
  await page
    .getByRole("navigation", { name: "Secciones", exact: true })
    .getByRole("button", { name: "Mi cuenta", exact: true })
    .click();
}
async function contained(page: Page) {
  const result = await page.locator(".profile-location .map").evaluate((el) => {
    const box = el.getBoundingClientRect(),
      frame = el.parentElement!,
      boundary = frame.getBoundingClientRect();
    const panel = el.closest(".account-map-panel")!.getBoundingClientRect();
    const style = getComputedStyle(frame);
    const samples = [
      [box.left - 2, box.top + box.height / 2],
      [box.right + 2, box.top + box.height / 2],
      [box.left + box.width / 2, box.top - 2],
      [box.left + box.width / 2, box.bottom + 2],
    ];
    const escapes = samples
      .filter(([x, y]) => x >= 0 && x < innerWidth && y >= 0 && y < innerHeight)
      .some(
        ([x, y]) =>
          document.elementFromPoint(x, y)?.closest(".leaflet-container") === el,
      );
    return {
      pageFits: document.documentElement.scrollWidth <= innerWidth,
      bounded:
        box.left >= panel.left &&
        box.right <= panel.right &&
        box.bottom <= panel.bottom,
      frameFits:
        Math.abs(box.width - boundary.width) <= 5 &&
        Math.abs(box.height - boundary.height) <= 5,
      clipped: style.overflow === "hidden" && style.contain.includes("paint"),
      escapes,
    };
  });
  expect(result).toEqual({
    pageFits: true,
    bounded: true,
    frameFits: true,
    clipped: true,
    escapes: false,
  });
}
for (const role of ["company", "courier"]) {
  test(`mapa ${role} queda contenido al ampliar, desplazar y redimensionar`, async ({
    page,
  }) => {
    await page.route("**/api/**", (r) => {
      const path = new URL(r.request().url()).pathname;
      let data: unknown = [];
      if (path === "/api/session")
        data = {
          user: {
            id: 1,
            name: "Prueba mapa",
            email: "mapa@example.test",
            role,
          },
          csrf_token: "fixture",
        };
      else if (path === "/api/jobs") data = { items: [], total: 0 };
      else if (path === "/api/profile")
        data = {
          profile: {
            address: "Mi dirección",
            latitude: 23.11345,
            longitude: -82.3667,
          },
        };
      else if (path === "/api/maps/reverse")
        data = { address: "Punto seleccionado en la prueba" };
      else if (path === "/api/halcon") data = { linked: false };
      else if (path.endsWith("/rating"))
        data = {
          average: 0,
          count: 0,
          completed_jobs: 0,
          can_rate: false,
          rating: null,
        };
      return r.fulfill({
        contentType: "application/json",
        body: JSON.stringify(data),
      });
    });
    await page.setViewportSize({ width: 1366, height: 768 });
    await page.goto("/");
    await account(page);
    const map = page.getByRole("region", {
      name: "Elegir dirección del perfil",
      exact: true,
    });
    await map.scrollIntoViewIfNeeded();
    for (let i = 0; i < 3; i++) {
      await map.getByRole("button", { name: "Zoom in", exact: true }).click();
      await contained(page);
    }
    await map.focus();
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowDown");
    for (let i = 0; i < 4; i++) {
      await map.getByRole("button", { name: "Zoom out", exact: true }).click();
      await contained(page);
    }
    await page.evaluate(() => (document.documentElement.style.zoom = "1.25"));
    await contained(page);
    await page.evaluate(() => (document.documentElement.style.zoom = "1"));
    for (const width of [1920, 1440, 1024, 800, 768, 390]) {
      await page.setViewportSize({ width, height: 900 });
      await contained(page);
    }
    await page.setViewportSize({ width: 1366, height: 768 });
    await map.click({ position: { x: 170, y: 110 } });
    await expect(page.getByLabel("Dirección seleccionada")).toHaveValue(
      "Punto seleccionado en la prueba",
    );
    await expect(
      page.getByRole("button", { name: "Guardar ubicación", exact: true }),
    ).toBeEnabled();
    await contained(page);
    if (role === "company") {
      await page.evaluate(() => {
        window.scrollTo({ top: 0, behavior: "instant" });
        (document.activeElement as HTMLElement)?.blur();
      });
      await page.screenshot({
        path: `/tmp/chita-map-${process.env.CHITA_BROWSER || "chromium"}-after.png`,
        fullPage: true,
      });
    }
  });
}
for (const role of ["company", "courier"]) {
  test(`ubicación ${role} se guarda realmente y persiste al recargar`, async ({
    page,
    baseURL,
  }) => {
    test.skip(
      process.env.CHITA_REAL_DISPATCH !== "1",
      "requires isolated CHITA",
    );
    if (baseURL !== "http://127.0.0.1:3340")
      throw Error("isolated server 3340 required");
    const email = `location-${role}-${Date.now()}@browser.chita.test`;
    let session = await (await page.request.get("/api/session")).json();
    const registration = await page.request.post("/api/auth/register", {
      headers: { "X-CSRF-Token": session.csrf_token },
      data: {
        name: "Cuenta ubicación",
        email,
        password: "StrongTest123!",
        phone: "12345678",
        role,
        address: "Referencia inicial",
        latitude: 23.11345,
        longitude: -82.3667,
        company_name: "Empresa ubicación",
        vehicle_type: "bicycle",
      },
    });
    expect(registration.status()).toBe(201);
    // Only the external address provider is controlled; auth, writes and reads use real HTTP/PostgreSQL.
    await page.route("**/api/maps/reverse?*", (r) =>
      r.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ address: "Dirección persistida desde el mapa" }),
      }),
    );
    await page.setViewportSize({ width: 1366, height: 768 });
    await page.goto("/");
    await account(page);
    const map = page.getByRole("region", {
      name: "Elegir dirección del perfil",
      exact: true,
    });
    await map.getByRole("button", { name: "Zoom in", exact: true }).click();
    await map.click({ position: { x: 180, y: 120 } });
    await expect(page.getByLabel("Dirección seleccionada")).toHaveValue(
      "Dirección persistida desde el mapa",
    );
    const response = page.waitForResponse(
      (r) =>
        r.url().endsWith("/api/profile/location") &&
        r.request().method() === "PUT",
    );
    await page
      .getByRole("button", { name: "Guardar ubicación", exact: true })
      .click();
    const write = await response;
    expect(write.status()).toBe(204);
    const sent = write.request().postDataJSON();
    expect(sent.latitude).not.toBe(23.11345);
    expect(sent.longitude).not.toBe(-82.3667);
    const persisted = await (await page.request.get("/api/profile")).json();
    expect(persisted.profile.address).toBe(sent.address);
    expect(persisted.profile.latitude).toBeCloseTo(sent.latitude, 7);
    expect(persisted.profile.longitude).toBeCloseTo(sent.longitude, 7);
    await page.reload();
    await account(page);
    await expect(page.getByLabel("Dirección seleccionada")).toHaveValue(
      sent.address,
    );
    await expect(page.locator('input[name="latitude"]')).toHaveValue(
      String(persisted.profile.latitude),
    );
    await expect(page.locator('input[name="longitude"]')).toHaveValue(
      String(persisted.profile.longitude),
    );
    session = await (await page.request.get("/api/session")).json();
    const invalid = await page.request.put("/api/profile/location", {
      headers: { "X-CSRF-Token": session.csrf_token },
      data: { ...sent, user_id: 999 },
    });
    expect(invalid.status()).toBe(422);
    const noCSRF = await page.request.put("/api/profile/location", {
      data: sent,
    });
    expect(noCSRF.status()).toBe(403);
    const loggedOut = page.waitForResponse(
      (r) =>
        r.url().endsWith("/api/auth/logout") && r.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Salir", exact: true }).click();
    expect((await loggedOut).status()).toBe(204);
    await expect(
      page.getByRole("button", { name: "Salir", exact: true }),
    ).toHaveCount(0);
    const noAuth = await page.request.put("/api/profile/location", {
      headers: {
        "X-CSRF-Token": (await (await page.request.get("/api/session")).json())
          .csrf_token,
      },
      data: sent,
    });
    expect(noAuth.status()).toBe(401);
  });
}
