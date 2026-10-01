import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test("apariencia persistente, accesible y adaptable", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  const picker = page.getByLabel("Apariencia");
  await expect(picker).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  for (const theme of ["light", "dark"] as const) {
    await page.goto("/");
    await picker.selectOption(theme);
    await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
    await page.reload();
    await expect(picker).toHaveValue(theme);
    await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    }
    const result = await new AxeBuilder({ page }).analyze();
    expect(result.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
    await page.getByRole("link", { name: "Entrar", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Bienvenido de nuevo" })).toBeVisible();
    const login = await new AxeBuilder({ page }).analyze();
    expect(login.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
    await page.getByRole("button", { name: "Crear cuenta", exact: true }).click();
    await page.setViewportSize({ width: 320, height: 900 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const signup = await new AxeBuilder({ page }).analyze();
    expect(signup.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
    await page.getByRole("button", { name: "Entrar", exact: true }).first().click();
  }
  await picker.selectOption("system");
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("apariencia funciona cuando el almacenamiento está bloqueado", async ({ page }) => {
  await page.addInitScript(() => {
    Storage.prototype.getItem = () => { throw new Error("blocked"); };
    Storage.prototype.setItem = () => { throw new Error("blocked"); };
  });
  await page.goto("/");
  await page.getByLabel("Apariencia").selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.getByLabel("Apariencia").selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

test("landing conecta roles, acceso y enlaces directos sin perder navegación", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Coordina la siguiente entrega." })).toBeVisible();
  await page.evaluate(() => { (window as Window & { chitaNavigationMarker?: boolean }).chitaNavigationMarker = true; });
  await page.getByRole("link", { name: "Soy empresa" }).click();
  expect(await page.evaluate(() => (window as Window & { chitaNavigationMarker?: boolean }).chitaNavigationMarker)).toBe(true);
  await expect(page).toHaveURL(/\/registro\?rol=empresa$/);
  await expect(page.getByLabel("Quiero usar CHITA como")).toHaveValue("company");
  await expect(page.getByLabel("Nombre de la empresa")).toBeVisible();
  await page.goBack();
  await expect(page.getByRole("heading", { name: "Coordina la siguiente entrega." })).toBeVisible();
  await page.getByRole("link", { name: "Soy repartidor" }).click();
  await expect(page.getByLabel("Quiero usar CHITA como")).toHaveValue("courier");
  await page.getByRole("button", { name: "Entrar", exact: true }).first().click();
  await expect(page).toHaveURL(/\/entrar$/);
  await expect(page.getByLabel("Correo electrónico", { exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("heading", { name: "Bienvenido de nuevo" })).toBeVisible();
  await page.getByRole("link", { name: "Volver a CHITA" }).click();
  await expect(page.getByRole("heading", { name: "Coordina la siguiente entrega." })).toBeVisible();
  await page.emulateMedia({ reducedMotion: "reduce" });
  expect(await page.locator(".preview-path").evaluate(el => getComputedStyle(el).animationName)).toBe("none");
});

test("contraseña visible y error de acceso conservan los datos", async ({ page }) => {
  await page.route("**/api/auth/login", route => route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ error: "Credenciales incorrectas" }) }));
  await page.goto("/entrar");
  await page.getByLabel("Correo electrónico", { exact: true }).fill("ux@example.test");
  const password = page.getByLabel("Contraseña", { exact: true });
  await password.fill("PruebaVisual123!");
  await page.getByRole("button", { name: "Mostrar contraseña" }).click();
  await expect(password).toHaveAttribute("type", "text");
  await page.getByRole("button", { name: "Ocultar contraseña" }).click();
  await expect(password).toHaveAttribute("type", "password");
  await page.getByRole("button", { name: "Entrar", exact: true }).last().click();
  await expect(page.getByRole("alert")).toContainText("Credenciales incorrectas");
  await expect(password).toHaveValue("PruebaVisual123!");
  await expect(page.getByLabel("Correo electrónico", { exact: true })).toHaveValue("ux@example.test");
});

test("paneles de ambos roles son adaptables y accesibles", async ({ page }) => {
  for (const role of ["company", "courier"]) {
    await page.route("**/api/**", route => {
      const path = new URL(route.request().url()).pathname;
      const body = path === "/api/session" ? { user: { id: 1, name: "Cuenta de prueba visual", email: "ux@example.test", role, phone: "12345678" }, csrf_token: "visual-fixture" }
        : path === "/api/jobs" ? { items: [], total: 0 }
        : path === "/api/profile" ? { profile: { address: "Dirección de prueba", latitude: 23, longitude: -82 } }
        : path === "/api/halcon" ? { linked: false } : [];
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(body) });
    });
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Salir", exact: true })).toBeVisible();
    for (const theme of ["light", "dark"]) {
      await page.getByLabel("Apariencia").selectOption(theme);
      for (const width of [320, 768, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      }
      const result = await new AxeBuilder({ page }).analyze();
      expect(result.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
    }
    await page.unroute("**/api/**");
  }
});

test("menú móvil modal, HALCON y cercanía por GPS son utilizables", async ({ page, context }) => {
  await context.grantPermissions(["geolocation"]);
  await context.setGeolocation({latitude:12.25,longitude:-84.5});
  let query="";
  await page.route("**/api/**",route=>{
    const url=new URL(route.request().url());
    if(url.pathname==="/api/jobs") query=url.search;
    const body=url.pathname==="/api/session"?{user:{id:1,name:"Repartidor visual",email:"visual@example.test",role:"courier",phone:"12345678"},csrf_token:"visual"}
      :url.pathname==="/api/jobs"?{items:[],total:0}
      :url.pathname==="/api/profile"?{profile:{address:"Ubicación de perfil",latitude:23,longitude:-82}}
      :url.pathname==="/api/halcon"?{linked:false}:[];
    return route.fulfill({status:200,contentType:"application/json",body:JSON.stringify(body)});
  });
  await page.setViewportSize({width:390,height:844});
  await page.goto("/");
  const menu=page.getByRole("button",{name:"Menú",exact:true});
  await expect(menu).toBeVisible();
  await menu.click();
  const drawer=page.getByRole("dialog",{name:"Tu plataforma"});
  await expect(drawer).toBeVisible();
  const axe=await new AxeBuilder({page}).analyze();
  expect(axe.violations.map(v=>v.id)).toEqual([]);
  await drawer.getByRole("button",{name:"HALCON",exact:true}).click();
  await expect(drawer).not.toBeVisible();
  await expect(page.getByLabel("Correo de HALCON")).toBeVisible();
  await menu.click();
  await page.keyboard.press("Escape");
  await expect(drawer).not.toBeVisible();
  await expect(menu).toBeFocused();
  await menu.click();
  await drawer.getByRole("button",{name:"Trabajos",exact:true}).click();
  await page.getByRole("button",{name:"Ordenar cerca de mí",exact:true}).click();
  await expect.poll(()=>query).toContain("lat=12.25&lng=-84.5");
  await expect(page.getByText(/tu última ubicación GPS/)).toBeVisible();
  expect(await page.locator(".platform-logo").evaluate(el=>(el as HTMLImageElement).naturalWidth)).toBeGreaterThan(0);
  await menu.click();
  await page.setViewportSize({width:1440,height:900});
  await expect(drawer).not.toBeVisible();
  await expect(page.getByRole("navigation",{name:"Secciones",exact:true})).toBeVisible();
});
