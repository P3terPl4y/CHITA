import { useEffect, useRef, useState } from "react";
import { api, APIError, labels, type Job, type User } from "./api";
import { freshPosition } from "./gps";
import { PointMap } from "./Map";
import { RatingCard } from "./RatingCard";
export type Offer = {
  id: number;
  publication_id: number;
  courier_user_id: number;
  status: string;
  expires_at: string;
  job?: Job;
  courier?: { id: number; name: string };
};
type Nearby = {
  id: number;
  name: string;
  vehicle_type: string;
  latitude: number;
  longitude: number;
  last_seen: string;
  distance_km: number;
  average_rating: number;
  rating_count: number;
  completed_jobs: number;
  pending_offer: boolean;
};
const offerLabels: Record<string, string> = {
  pending: "Esperando respuesta",
  accepted: "Aceptada",
  declined: "Rechazada",
  expired: "Vencida",
  withdrawn: "Retirada",
};
const vehicle: Record<string, string> = {
  foot: "A pie",
  bicycle: "Bicicleta",
  motorbike: "Moto",
  car: "Auto",
  van: "Furgoneta",
};
export function CourierAvailability({
  hasActiveJob,
}: {
  hasActiveJob: boolean;
}) {
  const [enabled, setEnabled] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const generation = useRef(0);
  const sending = useRef(false);
  const consent = useRef("");
  useEffect(() => {
    if (!enabled) return;
    const version = ++generation.current;
    async function pulse() {
      if (sending.current || version !== generation.current) return;
      sending.current = true;
      try {
        if (!navigator.geolocation)
          throw new Error("Este navegador no ofrece GPS");
        const p = await freshPosition(null, navigator.geolocation);
        if (version !== generation.current) return;
        await api("/availability", "POST", {
          enabled: true,
          token: consent.current,
          latitude: p.coords.latitude,
          longitude: p.coords.longitude,
        });
        if (version === generation.current) setError("");
      } catch (e) {
        if (version === generation.current) {
          setError(
            e instanceof Error
              ? e.message
              : "No se pudo actualizar tu disponibilidad",
          );
          if (e instanceof APIError && e.status === 409) {
            consent.current = "";
            setEnabled(false);
          }
        }
      } finally {
        sending.current = false;
      }
    }
    const timer = setInterval(() => void pulse(), 30000);
    return () => {
      generation.current++;
      clearInterval(timer);
    };
  }, [enabled]);
  useEffect(() => {
    if (hasActiveJob && enabled) {
      generation.current++;
      setEnabled(false);
      void api("/availability", "POST", { enabled: false }).catch(() => {});
    }
  }, [hasActiveJob, enabled]);
  async function toggle() {
    if (busy) return;
    setBusy(true);
    setError("");
    const version = ++generation.current;
    try {
      if (enabled) {
        setEnabled(false);
        await api("/availability", "POST", { enabled: false });
      } else {
        if (!navigator.geolocation)
          throw new Error(
            "Este navegador no ofrece GPS. La disponibilidad cercana necesita una posición reciente.",
          );
        const p = await freshPosition(null, navigator.geolocation);
        if (version !== generation.current) return;
        const response = await api<{ token: string }>("/availability", "POST", {
          enabled: true,
          latitude: p.coords.latitude,
          longitude: p.coords.longitude,
        });
        consent.current = response.token;
        setEnabled(true);
      }
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "No se pudo cambiar tu disponibilidad. La última posición caduca en cinco minutos.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section
      className="availability-strip"
      aria-label="Disponibilidad para propuestas"
    >
      <div>
        <strong>
          {enabled
            ? "Visible para empresas cercanas"
            : "Disponibilidad para propuestas"}
        </strong>
        <p className="muted">
          {enabled
            ? "Tu posición GPS se actualiza mientras esta página está abierta. Al aceptar un trabajo dejas de aparecer como disponible."
            : "Actívala para que empresas cercanas vean tu nombre, transporte, calificaciones y posición GPS, y te propongan trabajos."}{" "}
          La posición deja de mostrarse a los cinco minutos sin actualizarse.
        </p>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
      </div>
      <button
        type="button"
        disabled={busy || (!enabled && hasActiveJob)}
        onClick={() => void toggle()}
      >
        {busy
          ? "Actualizando…"
          : enabled
            ? "Dejar de mostrar mi ubicación"
            : hasActiveJob
              ? "Tienes entregas pendientes"
              : "Mostrarme disponible"}
      </button>
    </section>
  );
}
export function OffersPanel({
  user,
  openJob,
  changed,
}: {
  user: User;
  openJob: (job: Job) => void;
  changed: () => Promise<void>;
}) {
  const [offers, setOffers] = useState<Offer[]>([]),
    [now, setNow] = useState(Date.now()),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true);
  const generation = useRef(0);
  async function load() {
    const version = ++generation.current;
    const rows = await api<Offer[]>("/offers");
    if (version === generation.current) {
      setOffers(rows);
      setLoading(false);
    }
  }
  useEffect(() => {
    void load().catch((e) => {
      setError(e.message);
      setLoading(false);
    });
    const timer = setInterval(
      () => void load().catch((e) => setError(e.message)),
      10000,
    );
    const clock = setInterval(() => setNow(Date.now()), 1000);
    return () => {
      generation.current++;
      clearInterval(timer);
      clearInterval(clock);
    };
  }, []);
  async function respond(offer: Offer, action: string) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/offers/${offer.id}/${action}`, "POST", {});
      await Promise.all([load(), changed()]);
      if (action === "accept") {
        const job = await api<Job>(`/jobs/${offer.publication_id}`);
        openJob(job);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "No se pudo responder");
      void load().catch(() => {});
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel">
      <h2>Propuestas de trabajo</h2>
      <p className="muted">
        La empresa reserva el trabajo por hasta dos minutos. Sólo se asigna
        cuando el repartidor acepta; al rechazar o vencer el plazo vuelve a
        estar disponible.
      </p>
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}
      {loading ? (
        <p role="status">Cargando propuestas…</p>
      ) : !offers.length ? (
        <p>Aún no tienes propuestas.</p>
      ) : (
        offers.map((o) => {
          const seconds = Math.max(
            0,
            Math.ceil((new Date(o.expires_at).getTime() - now) / 1000),
          );
          const pending = o.status === "pending" && seconds > 0;
          const status =
            o.status === "pending" && !seconds ? "expired" : o.status;
          return (
            <article className="offer-card" key={o.id}>
              <div>
                <span className="badge">{offerLabels[status] ?? status}</span>
                <h3>{o.job?.title ?? `Trabajo #${o.publication_id}`}</h3>
                <p>
                  {user.role === "company"
                    ? o.courier?.name
                    : o.job?.company?.name}
                </p>
                {o.job && (
                  <>
                    <p>
                      {o.job.pickup_address} → {o.job.dropoff_address}
                    </p>
                    <p>
                      <strong>
                        {(o.job.price_cents / 100).toFixed(2)} {o.job.currency}
                      </strong>{" "}
                      · Recogida hasta{" "}
                      {new Date(o.job.pickup_to).toLocaleString("es")}
                    </p>
                  </>
                )}
                {pending && (
                  <p
                    className="offer-clock"
                    role="timer"
                    aria-label="Tiempo restante"
                  >
                    {Math.floor(seconds / 60)}:
                    {String(seconds % 60).padStart(2, "0")} para responder
                  </p>
                )}
              </div>
              <div className="actions">
                {pending && user.role === "courier" && (
                  <>
                    <button
                      className="primary"
                      disabled={busy}
                      onClick={() => void respond(o, "accept")}
                    >
                      Aceptar propuesta
                    </button>
                    <button
                      disabled={busy}
                      onClick={() => void respond(o, "decline")}
                    >
                      Rechazar propuesta
                    </button>
                  </>
                )}
                {pending && user.role === "company" && (
                  <button
                    disabled={busy}
                    onClick={() => void respond(o, "withdraw")}
                  >
                    Retirar propuesta
                  </button>
                )}
                {o.job && ["pending", "accepted"].includes(status) && (
                  <button onClick={() => openJob(o.job!)}>Ver trabajo</button>
                )}
              </div>
            </article>
          );
        })
      )}
    </section>
  );
}
export function CompanyDiscovery({
  jobs,
  selected,
  profile,
  choose,
  changed,
}: {
  jobs: Job[];
  selected: Job | null;
  profile: { latitude: number; longitude: number; address: string } | null;
  choose: (job: Job | null) => void;
  changed: () => Promise<void>;
}) {
  const job = selected?.status === "published" && new Date(selected.pickup_to).getTime() > Date.now() ? selected : null;
  const available = jobs.filter(
    (j) =>
      j.status === "published" && new Date(j.pickup_to).getTime() > Date.now(),
  );
  if (job && !available.some((j) => j.id === job.id)) available.unshift(job);
  const [radius, setRadius] = useState("10"),
    [rows, setRows] = useState<Nearby[]>([]),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState("");
  const [selectedCourier, setSelectedCourier] = useState<number | null>(null);
  const generation = useRef(0);
  useEffect(() => {
    if (selectedCourier == null) return;
    const card = document.getElementById(`nearby-courier-${selectedCourier}`);
    if (!card) return;
    card.scrollIntoView({
      block: "nearest",
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "instant"
        : "smooth",
    });
    card.focus({ preventScroll: true });
  }, [selectedCourier]);
  const latitude = job?.pickup_lat ?? profile?.latitude,
    longitude = job?.pickup_lng ?? profile?.longitude;
  async function load() {
    if (latitude == null || longitude == null) return;
    const version = ++generation.current;
    setLoading(true);
    try {
      const found = await api<Nearby[]>(
        `/couriers/nearby?lat=${latitude}&lng=${longitude}&radius=${radius}`,
      );
      if (version === generation.current) setRows(found);
    } catch (e) {
      if (version === generation.current)
        setError(
          e instanceof Error
            ? e.message
            : "No se pudieron obtener repartidores",
        );
    } finally {
      if (version === generation.current) setLoading(false);
    }
  }
  useEffect(() => {
    setError("");
    setNotice("");
    setRows([]);
    setSelectedCourier(null);
    void load();
    const timer = setInterval(() => void load(), 15000);
    return () => {
      generation.current++;
      clearInterval(timer);
    };
  }, [latitude, longitude, radius]);
  async function propose(courier: Nearby) {
    if (!job || busy || loading) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api(`/jobs/${job.id}/offer`, "POST", { courier_id: courier.id });
      setNotice(
        `Propuesta enviada a ${courier.name}. Puede responder durante dos minutos o hasta vencer la recogida.`,
      );
      await Promise.all([load(), changed()]);
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "No se pudo enviar la propuesta",
      );
      void load();
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel">
      <h2>Repartidores cercanos</h2>
      <p>
        Consulta repartidores que han activado su disponibilidad y envíales una
        propuesta. La distancia es en línea recta desde{" "}
        {job ? "la recogida del trabajo" : "la dirección de tu empresa"}; no
        representa tiempo de viaje.
      </p>
      <div className="fields">
        <label>
          Trabajo para proponer
          <select
            value={job?.id ?? ""}
            onChange={(e) =>
              choose(
                available.find((j) => j.id === Number(e.target.value)) ?? null,
              )
            }
          >
            <option value="">Sólo consultar desde mi empresa</option>
            {available.map((j) => (
              <option key={j.id} value={j.id}>
                {j.title} · {labels[j.status]}
              </option>
            ))}
          </select>
        </label>
        <label>
          Radio de búsqueda
          <select value={radius} onChange={(e) => setRadius(e.target.value)}>
            {[2, 5, 10, 25, 50].map((r) => (
              <option value={r} key={r}>
                {r} km
              </option>
            ))}
          </select>
        </label>
        <button disabled={loading} onClick={() => void load()}>
          Actualizar cercanía
        </button>
      </div>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      {latitude != null && longitude != null && (
        <PointMap
          label="Mapa de repartidores disponibles, recogida y entrega"
          points={[
            {
              latitude,
              longitude,
              label: job?.pickup_address ?? profile?.address ?? "Origen",
              kind: job ? "pickup" : "origin",
            },
            ...(job ? [{
              latitude: job.dropoff_lat,
              longitude: job.dropoff_lng,
              label: job.dropoff_address,
              kind: "dropoff" as const,
            }] : []),
            ...rows.map((c) => ({
              latitude: c.latitude,
              longitude: c.longitude,
              label: c.name,
              id: c.id,
              kind: "courier" as const,
            })),
          ]}
          select={setSelectedCourier}
          selectedId={selectedCourier}
          viewKey={job?.id ?? `profile:${latitude}:${longitude}`}
        />
      )}
      <p className="muted" aria-live="polite">
        {loading
          ? "Buscando repartidores…"
          : `${rows.length} repartidores disponibles. Se muestran hasta 50, ordenados por cercanía.`}
      </p>
      <div className="nearby-list">
        {rows.map((c) => (
          <article
            className={
              "nearby-card " + (selectedCourier === c.id ? "selected" : "")
            }
            key={c.id}
            id={`nearby-courier-${c.id}`}
            tabIndex={-1}
          >
            <h3>{c.name}</h3>
            <p>
              {vehicle[c.vehicle_type] ?? c.vehicle_type} ·{" "}
              {c.distance_km.toFixed(1)} km
            </p>
            <p>
              {c.rating_count
                ? `${c.average_rating.toFixed(1)} de 5 · ${c.rating_count} empresas`
                : "Aún sin calificaciones"}
            </p>
            <p className="muted">
              Posición GPS actualizada{" "}
              {new Date(c.last_seen).toLocaleTimeString("es")} ·{" "}
              {c.completed_jobs} entregas confirmadas para tu empresa.
            </p>
            <button
              className="primary"
              disabled={busy || loading || !job || c.pending_offer || !!job.pending_offer}
              onClick={() => void propose(c)}
            >
              {c.pending_offer
                ? "Tiene una propuesta pendiente"
                : job?.pending_offer
                  ? "El trabajo tiene una propuesta"
                  : "Proponer este trabajo"}
            </button>
            {!job && (
              <p className="muted">
                Selecciona un trabajo para enviar una propuesta.
              </p>
            )}
            {c.completed_jobs >= 3 && <RatingCard courierID={c.id} />}
          </article>
        ))}
      </div>
    </section>
  );
}
