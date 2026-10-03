import { afterEach, describe, it, expect, vi } from "vitest";
import { api, APIError, cents, active } from "./api";
describe("tarifas", () => {
  it.each([
    ["0.01", 1],
    ["12.34", 1234],
    ["12.3", 1230],
    ["12", 1200],
  ])("convierte %s sin redondeo flotante", (s, n) => expect(cents(s)).toBe(n));
  it.each(["0", "-1", "1.234", "1e3", "NaN", "10,5", "10000001"])(
    "rechaza %s",
    (s) => expect(() => cents(s)).toThrow(),
  );
});
it("limita seguimiento al trabajo activo", () => {
  for (const s of ["accepted", "picked_up", "arrived"])
    expect(active(s)).toBe(true);
  for (const s of ["published", "delivery_reported", "completed", "cancelled"])
    expect(active(s)).toBe(false);
});

describe("feedback de errores de API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    [401, "Tu sesión no es válida o ha caducado. Vuelve a entrar."],
    [403, "No tienes permiso para esta acción. Actualiza la página y vuelve a intentarlo."],
    [404, "No encontramos este elemento. Puede que se haya retirado o cambiado."],
    [409, "Este elemento cambió mientras lo consultabas. Actualiza la lista antes de continuar."],
    [422, "Revisa los datos indicados y vuelve a intentarlo."],
    [429, "Has realizado muchas solicitudes seguidas. Espera un momento y vuelve a intentarlo."],
    [503, "CHITA tuvo un problema temporal. Inténtalo de nuevo en unos minutos."],
  ])("ofrece una instrucción útil cuando HTTP responde %i sin mensaje", async (status, message) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("", { status })));
    await expect(api("/test")).rejects.toMatchObject({ status, message });
  });

  it("conserva el mensaje seguro que devuelve la API", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: "La entrega ya fue aceptada por otra persona." }),
      { status: 409, headers: { "Content-Type": "application/json" } },
    )));
    await expect(api("/test")).rejects.toMatchObject({
      name: "APIError",
      message: "La entrega ya fue aceptada por otra persona.",
      status: 409,
    } satisfies Partial<APIError>);
  });

  it("explica cómo recuperarse si falla la conexión", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(api("/test")).rejects.toMatchObject({
      message: "No pudimos conectar con CHITA. Comprueba tu conexión e inténtalo de nuevo.",
      status: 0,
    });
  });
});
