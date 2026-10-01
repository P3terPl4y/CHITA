import { describe, it, expect } from "vitest";
import { cents, active } from "./api";
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
