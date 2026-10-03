export type User = {
  avatar_url?: string;
  id: number;
  name: string;
  email: string;
  role: "company" | "courier" | "admin";
  phone: string;
};
export type Job = {
  pending_offer?: {
    id: number;
    courier_user_id: number;
    expires_at: string;
    status: string;
  };
  visibility: "network" | "public";
  pickup_distance_km?: number | null;
  id: number;
  title: string;
  description: string;
  status: string;
  company_id: number;
  assigned_courier_id: number | null;
  company?: { id: number; name: string };
  courier?: { id: number; name: string };
  pickup_address: string;
  pickup_lat: number;
  pickup_lng: number;
  dropoff_address: string;
  dropoff_lat: number;
  dropoff_lng: number;
  pickup_from: string;
  pickup_to: string;
  delivery_from: string;
  delivery_to: string;
  price_cents: number;
  currency: string;
  cancellation_reason: string;
};
export type Network = {
  id: number;
  name: string;
  email?: string;
  status: string;
};
export type Notice = {
  id: number;
  message: string;
  read_at?: string;
  publication_id?: number;
};
export type Position = {
  latitude: number | null;
  longitude: number | null;
  active: boolean;
  last_seen: string | null;
};
export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
    this.name = "APIError";
  }
}
function fallbackError(status: number) {
  if (status === 401) return "Tu sesión no es válida o ha caducado. Vuelve a entrar.";
  if (status === 403) return "No tienes permiso para esta acción. Actualiza la página y vuelve a intentarlo.";
  if (status === 404) return "No encontramos este elemento. Puede que se haya retirado o cambiado.";
  if (status === 409) return "Este elemento cambió mientras lo consultabas. Actualiza la lista antes de continuar.";
  if (status === 422 || status === 400) return "Revisa los datos indicados y vuelve a intentarlo.";
  if (status === 429) return "Has realizado muchas solicitudes seguidas. Espera un momento y vuelve a intentarlo.";
  if (status >= 500) return "CHITA tuvo un problema temporal. Inténtalo de nuevo en unos minutos.";
  return "No se pudo completar la operación. Vuelve a intentarlo.";
}
let csrf = "";
let authenticated = false;
export async function session() {
  const x = await api<{ user: User | null; csrf_token: string }>("/session");
  csrf = x.csrf_token;
  authenticated = x.user !== null;
  return x.user;
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  let r: Response;
  try {
    r = await fetch("/api" + path, {
      method,
      credentials: "same-origin",
      signal,
      headers: {
        "Content-Type": "application/json",
        ...(method === "GET" ? {} : { "X-CSRF-Token": csrf }),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
  } catch (e) {
    if (signal?.aborted || (e instanceof DOMException && e.name === "AbortError")) throw e;
    if (e instanceof TypeError)
      throw new APIError("No pudimos conectar con CHITA. Comprueba tu conexión e inténtalo de nuevo.", 0);
    throw e;
  }
  if (!r.ok) {
    if (r.status === 401 && authenticated) {
      authenticated = false;
      window.dispatchEvent(new Event("chita-session-ended"));
    }
    const x: unknown = await r.json().catch(() => null);
    const message =
      x && typeof x === "object" && "error" in x &&
      typeof x.error === "string" && x.error.trim()
        ? x.error.trim().slice(0, 320)
        : fallbackError(r.status);
    throw new APIError(message, r.status);
  }
  return r.status === 204 ? (undefined as T) : r.json();
}
export function cents(raw: string) {
  if (!/^\d{1,8}(\.\d{1,2})?$/.test(raw))
    throw new Error("La tarifa debe tener como máximo dos decimales");
  const [a, b = ""] = raw.split(".");
  const n = Number(a) * 100 + Number(b.padEnd(2, "0"));
  if (n <= 0 || n > 1_000_000_000) throw new Error("Tarifa fuera de rango");
  return n;
}
export const labels: Record<string, string> = {
  published: "Disponible",
  accepted: "Aceptado",
  picked_up: "Recogido",
  arrived: "En el punto de entrega",
  delivery_reported: "Pendiente de confirmación",
  completed: "Completado",
  cancelled: "Cancelado",
  pending: "Invitación pendiente",
  revoked: "Retirado",
};
export const active = (s: string) =>
  ["accepted", "picked_up", "arrived"].includes(s);
