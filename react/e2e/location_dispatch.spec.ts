import { test, expect, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
const tile = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg==",
  "base64",
);
test.beforeEach(async ({ context }) => {
  await context.route("https://tile.openstreetmap.org/**", (r) =>
    r.fulfill({ contentType: "image/png", body: tile }),
  );
});
test("selección de mapa rellena coordenadas y dirección; fallos permiten corregir", async ({
  page,
}) => {
  let fail = false;
  let calls = 0;
  await page.route("**/api/maps/reverse?**", (r) => {
    calls++;
    return r.fulfill({
      status: fail ? 503 : 200,
      contentType: "application/json",
      body: JSON.stringify(
        fail
          ? {
              error:
                "Proveedor no disponible. Escribe la dirección manualmente.",
            }
          : { address: "Calle del mapa 15, Habana, Cuba" },
      ),
    });
  });
  await page.goto("/registro");
  const map = page.getByRole("region", { name: "Elegir dirección del perfil" });
  await expect(map).toBeVisible();
  await map.click({ position: { x: 170, y: 110 } });
  await expect(page.getByLabel("Dirección de referencia")).toHaveValue(
    "Calle del mapa 15, Habana, Cuba",
  );
  await expect(page.getByLabel("Latitud", { exact: true })).not.toHaveValue("");
  await expect(page.getByLabel("Longitud", { exact: true })).not.toHaveValue(
    "",
  );
  await page
    .getByLabel("Dirección de referencia")
    .fill("Referencia corregida por usuario");
  await expect(page.getByLabel("Dirección de referencia")).toHaveValue(
    "Referencia corregida por usuario",
  );
  fail = true;
  await map.click({ position: { x: 190, y: 130 } });
  await expect(
    page.getByRole("status").filter({ hasText: "Proveedor no disponible" }),
  ).toBeVisible();
  await expect(page.getByLabel("Dirección de referencia")).toHaveValue("");
  await page
    .getByLabel("Dirección de referencia")
    .fill("Dirección manual válida");
  expect(calls).toBe(2);
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
    expect(
      (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => n.target),
      })),
    ).toEqual([]);
  }
});
test("respuesta antigua de dirección no sobrescribe el último punto", async ({
  page,
}) => {
  let count = 0;
  await page.route("**/api/maps/reverse?**", async (r) => {
    const old = ++count === 1;
    await new Promise((resolve) => setTimeout(resolve, old ? 450 : 20));
    await r
      .fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          address: old ? "Dirección antigua" : "Dirección nueva",
        }),
      })
      .catch(() => {});
  });
  await page.goto("/registro");
  const map = page.getByRole("region", { name: "Elegir dirección del perfil" });
  await map.click({ position: { x: 170, y: 120 } });
  await map.click({ position: { x: 220, y: 160 } });
  await expect(page.getByLabel("Dirección de referencia")).toHaveValue(
    "Dirección nueva",
  );
  await page.waitForTimeout(550);
  await expect(page.getByLabel("Dirección de referencia")).toHaveValue(
    "Dirección nueva",
  );
});
async function fixture(page: Page, role: "company" | "courier") {
  const now = Date.now();
  const job = {
    id: 3,
    title: "Entrega de prueba",
    description: "Paquete",
    company_id: 7,
    company: { id: 7, name: "Empresa Cercana" },
    assigned_courier_id: null,
    status: "published",
    visibility: "public",
    pickup_address: "Recogida 1",
    pickup_lat: 23.1,
    pickup_lng: -82.3,
    dropoff_address: "Entrega 2",
    dropoff_lat: 23.12,
    dropoff_lng: -82.32,
    pickup_from: new Date(now + 3600000).toISOString(),
    pickup_to: new Date(now + 7200000).toISOString(),
    delivery_from: new Date(now + 7200000).toISOString(),
    delivery_to: new Date(now + 10800000).toISOString(),
    price_cents: 1250,
    currency: "USD",
    cancellation_reason: "",
  };
  let proposed = false;
  const calls: string[] = [];
  await page.route("**/api/**", (r) => {
    const url = new URL(r.request().url());
    const path = url.pathname;
    calls.push(path + ":" + r.request().method());
    const user = {
      id: role === "company" ? 7 : 9,
      name: "Usuario de prueba",
      email: "fixture@example.test",
      phone: "12345678",
      role,
    };
    const offer = {
      id: 5,
      publication_id: 3,
      courier_user_id: 9,
      status: "pending",
      expires_at: new Date(now + 120000).toISOString(),
      job,
      courier: { id: 9, name: "Repartidor Cercano" },
    };
    let data: unknown = [];
    if (path === "/api/session") data = { user, csrf_token: "fixture" };
    else if (path === "/api/jobs") data = { items: [job], total: 1 };
    else if (path === "/api/profile")
      data = {
        profile: { address: "Mi dirección", latitude: 23.1, longitude: -82.3 },
      };
    else if (path === "/api/halcon") data = { linked: false };
    else if (path === "/api/couriers/nearby")
      data = [
        {
          id: 9,
          name: "Repartidor Cercano",
          vehicle_type: "bicycle",
          latitude: 23.11,
          longitude: -82.31,
          last_seen: new Date(now).toISOString(),
          distance_km: 1.2,
          average_rating: 4.5,
          rating_count: 2,
          completed_jobs: 2,
          pending_offer: proposed,
        },
      ];
    else if (path === "/api/jobs/3/offer") {
      proposed = true;
      data = offer;
    } else if (path === "/api/offers") data = [offer];
    else if (path === "/api/availability") data = { token: "consent-fixture" };
    else if (path.endsWith("/rating"))
      data = {
        completed_jobs: 2,
        can_rate: false,
        average: 4.5,
        count: 2,
        rating: null,
      };
    return r.fulfill({
      contentType: "application/json",
      body: JSON.stringify(data),
    });
  });
  return calls;
}
test("empresa encuentra cercanos y envía propuesta; paneles accesibles", async ({
  page,
}) => {
  const calls = await fixture(page, "company");
  await page.goto("/");
  await page
    .getByRole("navigation", { name: "Secciones", exact: true })
    .getByRole("button", { name: "Buscar repartidores", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Repartidor Cercano" }),
  ).toBeVisible();
  await page
    .getByRole("combobox", { name: "Trabajo para proponer" })
    .selectOption("3");
  await page
    .getByRole("button", { name: "Proponer este trabajo", exact: true })
    .click();
  await expect(
    page.getByRole("status").filter({ hasText: "Propuesta enviada" }),
  ).toBeVisible();
  expect(calls).toContain("/api/jobs/3/offer:POST");
  for (const theme of ["light", "dark"]) {
    await page.getByLabel("Apariencia").selectOption(theme);
    await page.setViewportSize({ width: 390, height: 900 });
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
  await page.getByRole("button", { name: "Menú", exact: true }).click();
  await page
    .getByRole("navigation", { name: "Secciones móviles" })
    .getByRole("button", { name: "Propuestas", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Propuestas de trabajo" }),
  ).toBeVisible();
  await expect(page.getByRole("timer")).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
  ).toEqual([]);
});
test("repartidor activa disponibilidad voluntaria y puede rechazar propuestas", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["geolocation"]);
  await context.setGeolocation({ latitude: 23.1, longitude: -82.3 });
  const calls = await fixture(page, "courier");
  await page.goto("/");
  await page.getByRole("button", { name: "Mostrarme disponible" }).click();
  await expect(
    page.getByRole("button", { name: "Dejar de mostrar mi ubicación" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Dejar de mostrar mi ubicación" })
    .click();
  await expect(
    page.getByRole("button", { name: "Mostrarme disponible" }),
  ).toBeVisible();
  await page
    .getByRole("navigation", { name: "Secciones", exact: true })
    .getByRole("button", { name: "Propuestas", exact: true })
    .click();
  await page.getByRole("button", { name: "Rechazar propuesta" }).click();
  expect(calls).toContain("/api/offers/5/decline:POST");
  expect(calls.filter((x) => x === "/api/availability:PUT")).toHaveLength(2);
  expect(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
  ).toEqual([]);
});

test("propuesta real y calificación sólo tras tres entregas confirmadas", async ({
  browser,
}) => {
  test.skip(
    process.env.CHITA_REAL_DISPATCH !== "1",
    "requires isolated CHITA server",
  );
  if (
    process.env.CHITA_BROWSER_URL &&
    !process.env.CHITA_BROWSER_URL.startsWith("http://127.0.0.1:3340")
  )
    throw new Error("Real dispatch tests must use isolated port 3340");
  test.setTimeout(90000);
  const companyContext = await browser.newContext(),
    courierContext = await browser.newContext({
      permissions: ["geolocation"],
      geolocation: { latitude: 23.11345, longitude: -82.3667 },
    });
  try {
    for (const context of [companyContext, courierContext]) {
      await context.route("https://tile.openstreetmap.org/**", (r) =>
        r.fulfill({ contentType: "image/png", body: tile }),
      );
      await context.route("**/api/maps/reverse?**", (r) =>
        r.fulfill({
          contentType: "application/json",
          body: JSON.stringify({ address: "Dirección elegida de prueba" }),
        }),
      );
    }
    const company = await companyContext.newPage(),
      courier = await courierContext.newPage(),
      stamp = Date.now();
    async function signup(p: Page, role: string) {
      await p.goto("/registro");
      await p.getByLabel("Quiero usar CHITA como").selectOption(role);
      await p
        .getByLabel("Tu nombre")
        .fill(role === "company" ? "Empresa Real" : "Repartidor Real");
      await p.getByLabel("Teléfono").fill("12345678");
      if (role === "company")
        await p.getByLabel("Nombre de la empresa").fill("Empresa Real");
      await p
        .getByLabel("Correo electrónico", { exact: true })
        .fill(`${role}-${stamp}@dispatch-browser.test`);
      await p
        .getByLabel("Contraseña", { exact: true })
        .fill("DispatchBrowser123!");
      await p
        .getByRole("region", { name: "Elegir dirección del perfil" })
        .click({ position: { x: 170, y: 120 } });
      await expect(p.getByLabel("Dirección de referencia")).toHaveValue(
        "Dirección elegida de prueba",
      );
      await p
        .getByRole("button", { name: "Crear cuenta", exact: true })
        .last()
        .click();
      await expect(
        p.getByRole("button", { name: "Salir", exact: true }),
      ).toBeVisible();
    }
    await signup(company, "company");
    await signup(courier, "courier");
    await courier.getByRole("button", { name: "Mostrarme disponible" }).click();
    await expect(
      courier.getByRole("button", { name: "Dejar de mostrar mi ubicación" }),
    ).toBeVisible();
    const title = `Propuesta Real ${stamp}`;
    await company
      .getByRole("button", { name: "Publicar trabajo", exact: true })
      .click();
    await company.getByLabel("Título del trabajo").fill(title);
    await company.getByLabel("Visibilidad del trabajo").selectOption("public");
    await company
      .getByRole("region", { name: "Elegir dirección de entrega" })
      .click({ position: { x: 180, y: 120 } });
    await expect(
      company.getByLabel("Dirección de entrega", { exact: true }),
    ).toHaveValue("Dirección elegida de prueba");
    const local = (h: number) => {
      const d = new Date(Date.now() + h * 3600000);
      return new Date(d.getTime() - d.getTimezoneOffset() * 60000)
        .toISOString()
        .slice(0, 16);
    };
    for (const [label, h] of [
      ["Recogida desde", 1],
      ["Recogida hasta", 2],
      ["Entrega desde", 2],
      ["Entrega hasta", 3],
    ] as const)
      await company.getByLabel(label).fill(local(h));
    await company.getByLabel("Tarifa acordada").fill("12.50");
    await company
      .getByRole("button", { name: "Publicar trabajo", exact: true })
      .last()
      .click();
    await expect(
      company
        .getByRole("status")
        .filter({ hasText: "Trabajo público publicado" }),
    ).toBeVisible();
    await company.locator(".jobcard").filter({ hasText: title }).click();
    await company
      .getByRole("button", { name: "Buscar repartidores para este trabajo" })
      .click();
    const rider = company
      .locator(".nearby-card")
      .filter({ hasText: "Repartidor Real" });
    await expect(rider).toBeVisible();
    await rider
      .getByRole("button", { name: "Proponer este trabajo", exact: true })
      .click();
    await expect(
      company.getByRole("status").filter({ hasText: "Propuesta enviada" }),
    ).toBeVisible();
    await courier
      .getByRole("navigation", { name: "Secciones", exact: true })
      .getByRole("button", { name: "Propuestas", exact: true })
      .click();
    await courier
      .locator(".offer-card")
      .filter({ hasText: title })
      .getByRole("button", { name: "Aceptar propuesta", exact: true })
      .click();
    await expect(courier.locator(".detail .badge")).toHaveText("Aceptado");
    async function finish(name: string) {
      await courier
        .getByRole("button", { name: "Confirmar recogida", exact: true })
        .click();
      await courier
        .getByRole("button", {
          name: "Estoy en el punto de entrega",
          exact: true,
        })
        .click();
      await courier
        .getByRole("button", { name: "Notificar entrega a la empresa" })
        .click();
      await expect(courier.locator(".detail .badge")).toHaveText(
        "Pendiente de confirmación",
      );
      await company.reload();
      await company.getByRole("navigation", { name: "Secciones", exact: true }).getByRole("button", { name: "Trabajos", exact: true }).click();
      await company.locator(".jobcard").filter({ hasText: name }).click();
      await company
        .getByRole("button", { name: "Confirmar entrega completada" })
        .click();
      await expect(company.locator(".detail .badge")).toHaveText("Completado");
    }
    await finish(title);
    await expect(company.locator(".rating-card")).toContainText(
      "1 de las 3 entregas",
    );
    for (let n = 2; n <= 3; n++) {
      const name = `Entrega confirmada ${stamp}-${n}`;
      const status = await company.evaluate(
        async ({ name }) => {
          const s = await fetch("/api/session").then((r) => r.json());
          const now = Date.now();
          const body = {
            title: name,
            description: "Paquete",
            visibility: "public",
            pickup_address: "Origen de prueba",
            pickup_lat: 23.11345,
            pickup_lng: -82.3667,
            dropoff_address: "Destino de prueba",
            dropoff_lat: 23.12,
            dropoff_lng: -82.37,
            pickup_from: new Date(now + 3600000).toISOString(),
            pickup_to: new Date(now + 7200000).toISOString(),
            delivery_from: new Date(now + 7200000).toISOString(),
            delivery_to: new Date(now + 10800000).toISOString(),
            price_cents: 1200,
            currency: "USD",
          };
          return (
            await fetch("/api/jobs", {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "X-CSRF-Token": s.csrf_token,
              },
              body: JSON.stringify(body),
            })
          ).status;
        },
        { name },
      );
      expect(status).toBe(201);
      await courier.reload();
      await courier.locator(".jobcard").filter({ hasText: name }).click();
      await courier
        .getByRole("button", { name: "Aceptar trabajo", exact: true })
        .click();
      await finish(name);
      if (n === 2)
        await expect(company.locator(".rating-card")).toContainText(
          "2 de las 3 entregas",
        );
    }
    await expect(
      company.getByRole("button", {
        name: "Guardar calificación",
        exact: true,
      }),
    ).toBeVisible();
    await company
      .getByRole("combobox", { name: "Calificación", exact: true })
      .selectOption("4");
    await company
      .getByLabel("Comentario", { exact: true })
      .fill("Tres entregas verificadas en navegador");
    await company
      .getByRole("button", { name: "Guardar calificación", exact: true })
      .click();
    await expect(
      company.getByRole("status").filter({ hasText: "Calificación guardada" }),
    ).toBeVisible();
    await expect(company.locator(".rating-card")).toContainText(
      "4.0 de 5 · 1 empresas",
    );
  } finally {
    await companyContext.close();
    await courierContext.close();
  }
});

test("seleccionar un trabajo en móvil enfoca sus detalles", async ({
  page,
}) => {
  await fixture(page, "company");
  await page.setViewportSize({ width: 390, height: 900 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  await page.getByRole("button", { name: "Menú", exact: true }).click();
  await page.getByRole("navigation", { name: "Secciones móviles" }).getByRole("button", { name: "Trabajos", exact: true }).click();
  await page.locator(".jobcard").first().click();
  const detail = page.getByRole("region", {
    name: "Detalles del trabajo",
    exact: true,
  });
  await expect(detail).toBeFocused();
  await expect(
    detail.getByRole("heading", { name: "Entrega de prueba", exact: true }),
  ).toBeInViewport();
});
