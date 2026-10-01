import { describe, it, expect, vi } from "vitest";
import { freshPosition } from "./gps";
const fix = (timestamp: number) =>
  ({
    timestamp,
    coords: { latitude: 23, longitude: -82, accuracy: 5 },
  }) as GeolocationPosition;
describe("GPS en primer plano", () => {
  it("reutiliza una posición reciente", async () => {
    const p = fix(Date.now()),
      getCurrentPosition = vi.fn();
    expect(await freshPosition(p, { getCurrentPosition })).toBe(p);
    expect(getCurrentPosition).not.toHaveBeenCalled();
  });
  it("renueva una posición antigua aunque no haya movimiento", async () => {
    const fresh = fix(Date.now()),
      getCurrentPosition = vi.fn((ok: PositionCallback) => ok(fresh));
    expect(
      await freshPosition(fix(Date.now() - 65000), { getCurrentPosition }),
    ).toBe(fresh);
    expect(getCurrentPosition).toHaveBeenCalledOnce();
  });
  it("obtiene una posición cuando todavía no existe una", async () => {
    const fresh = fix(Date.now()),
      getCurrentPosition = vi.fn((ok: PositionCallback) => ok(fresh));
    expect(await freshPosition(null, { getCurrentPosition })).toBe(fresh);
  });
  it("explica un fallo de permisos o de señal", async () => {
    const getCurrentPosition = vi.fn(
      (_ok: PositionCallback, error?: PositionErrorCallback | null) =>
        error?.({ code: 1 } as GeolocationPositionError),
    );
    await expect(freshPosition(null, { getCurrentPosition })).rejects.toThrow(
      "Revisa los permisos",
    );
  });
  it("rechaza una respuesta del dispositivo que sigue siendo antigua", async () => {
    const getCurrentPosition = vi.fn((ok: PositionCallback) =>
      ok(fix(Date.now() - 90000)),
    );
    await expect(freshPosition(null, { getCurrentPosition })).rejects.toThrow(
      "posición reciente",
    );
  });
});
