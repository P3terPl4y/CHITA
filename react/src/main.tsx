import { freshPosition } from "./gps";
import { ThemePicker } from "./Theme";
import { Landing } from "./Landing";
import { PasswordField } from "./PasswordField";
import { AdminPanel } from "./AdminPanel";
import { CompanyDiscovery, CourierAvailability, OffersPanel } from "./Dispatch";
import { RatingCard } from "./RatingCard";
import { ProfileLocation } from "./ProfileLocation";
import { Avatar, ProfilePhoto } from "./ProfilePhoto";
import { CourierDirectory } from "./CourierDirectory";
import { UsageGuide } from "./UsageGuide";
import { DashboardHome } from "./DashboardHome";
import { Navigation } from "./Navigation";
import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
  type MouseEvent,
} from "react";
import { createRoot } from "react-dom/client";
import {
  api,
  session,
  cents,
  labels,
  active,
  type User,
  type Job,
  type Network,
  type Notice,
  type Position,
} from "./api";
import { Map, LocationPicker } from "./Map";
import "./style.css";
const money = (j: Job) =>
    new Intl.NumberFormat("es", {
      style: "currency",
      currency: j.currency,
    }).format(j.price_cents / 100),
  date = (s: string) =>
    new Date(s).toLocaleString("es", {
      dateStyle: "short",
      timeStyle: "short",
    });
const expired = (j: Job) =>
  j.status === "published" && new Date(j.pickup_to).getTime() <= Date.now();
const statusLabel = (j: Job) =>
  expired(j) ? "Recogida vencida" : labels[j.status] || j.status;

function Field({
  name,
  label,
  type = "text",
  required = true,
  value,
}: {
  name: string;
  label: string;
  type?: string;
  required?: boolean;
  value?: string;
}) {
  return (
    <label>
      {label}
      <input
        name={name}
        type={type}
        required={required}
        defaultValue={value}
        autoComplete={
          type === "password"
            ? "current-password"
            : type === "email"
              ? "email"
              : "off"
        }
        step={type === "number" ? "any" : undefined}
      />
    </label>
  );
}
function App() {
  const [publicPath, setPublicPath] = useState(window.location.pathname);
  const [rankingOrigin, setRankingOrigin] = useState<{
    lat: number;
    lng: number;
  } | null>(null);
  const [rankingBusy, setRankingBusy] = useState(false);
  const [user, setUser] = useState<User | null>(null),
    [ready, setReady] = useState(false),
    [online, setOnline] = useState(() => navigator.onLine),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [register, setRegister] = useState(
      window.location.pathname === "/registro",
    ),
    [role, setRole] = useState(
      new URLSearchParams(window.location.search).get("rol") === "empresa"
        ? "company"
        : "courier",
    ),
    [tab, setTab] = useState("jobs"),
    [jobs, setJobs] = useState<Job[]>([]),
    [network, setNetwork] = useState<Network[]>([]),
    [notices, setNotices] = useState<Notice[]>([]),
    [selected, setSelected] = useState<Job | null>(null),
    [linked, setLinked] = useState(false),
    [position, setPosition] = useState<Position | null>(null),
    [locError, setLocError] = useState(""),
    [sharing, setSharing] = useState(false),
    [creating, setCreating] = useState(false),
    [jobFilter, setJobFilter] = useState("all"),
    [jobSearch, setJobSearch] = useState(""),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [profile, setProfile] = useState<{
      address: string;
      latitude: number;
      longitude: number;
    } | null>(null);
  const detailPanel = useRef<HTMLElement>(null);
  const profileOrigin = useRef<{ lat: number; lng: number } | null>(null);
  const watch = useRef<number | null>(null),
    beat = useRef<ReturnType<typeof setInterval> | null>(null),
    gps = useRef<GeolocationPosition | null>(null),
    sending = useRef(false),
    generation = useRef(0),
    pollingFailure = useRef(false),
    gpsVersion = useRef(0),
    rankingVersion = useRef(0);
  function syncPublicRoute() {
    setPublicPath(window.location.pathname);
    setRegister(window.location.pathname === "/registro");
    setRole(
      new URLSearchParams(window.location.search).get("rol") === "empresa"
        ? "company"
        : "courier",
    );
  }
  useEffect(() => {
    window.addEventListener("popstate", syncPublicRoute);
    return () => window.removeEventListener("popstate", syncPublicRoute);
  }, []);
  function navigatePublic(event: MouseEvent<HTMLDivElement>) {
    if (
      user ||
      busy ||
      event.defaultPrevented ||
      event.button !== 0 ||
      event.ctrlKey ||
      event.metaKey ||
      event.shiftKey ||
      event.altKey
    )
      return;
    const anchor =
      event.target instanceof Element ? event.target.closest("a") : null;
    if (!anchor || anchor.hasAttribute("download") || anchor.target) return;
    const url = new URL(anchor.href);
    if (
      url.origin !== window.location.origin ||
      !["/", "/entrar", "/registro"].includes(url.pathname) ||
      url.hash
    )
      return;
    event.preventDefault();
    window.history.pushState(null, "", url.pathname + url.search);
    syncPublicRoute();
    setError("");
    setNotice("");
    window.scrollTo({ top: 0, behavior: "instant" });
    requestAnimationFrame(() =>
      document.getElementById("contenido")?.focus({ preventScroll: true }),
    );
  }
  async function refreshJobs(
    requestPage: number,
    version: number,
    orderVersion: number,
    includeTotal = true,
  ) {
    // Reuse the already loaded courier profile for feed ranking. Without this,
    // the API performs an extra profile lookup on every periodic feed request.
    const origin = rankingOrigin ??
      (user?.role === "courier" ? profileOrigin.current : null);
    const a = await api<{ items: Job[]; total?: number }>(
      "/jobs?page=" +
        requestPage +
        (includeTotal ? "" : "&total=false") +
        (origin
          ? `&lat=${origin.lat}&lng=${origin.lng}`
          : ""),
    );
    if (
      version !== generation.current ||
      orderVersion !== rankingVersion.current
    )
      return;
    setJobs(a.items);
    if (includeTotal && typeof a.total === "number") setTotal(a.total);
    setSelected((prev) =>
      prev ? a.items.find((j) => j.id === prev.id) || prev : null,
    );
  }
  async function refreshSideData(version: number, includeHalcon: boolean) {
    const [networkData, noticeData, halconData] = await Promise.all([
      api<Network[]>("/network"),
      api<Notice[]>("/notifications"),
      includeHalcon && user?.role === "courier"
        ? api<{ linked: boolean }>("/halcon")
        : Promise.resolve(null),
    ]);
    if (version !== generation.current) return;
    setNetwork(networkData);
    setNotices(noticeData);
    if (halconData) setLinked(halconData.linked);
  }
  async function refresh(requestPage = page) {
    const version = generation.current;
    const orderVersion = rankingVersion.current;
    await Promise.all([
      refreshJobs(requestPage, version, orderVersion),
      refreshSideData(version, true),
    ]);
  }
  useEffect(() => {
    const wentOnline = () => setOnline(true);
    const wentOffline = () => setOnline(false);
    window.addEventListener("online", wentOnline);
    window.addEventListener("offline", wentOffline);
    return () => {
      window.removeEventListener("online", wentOnline);
      window.removeEventListener("offline", wentOffline);
    };
  }, []);
  useEffect(() => {
    session()
      .then(setUser)
      .catch((e) => setError(e.message))
      .finally(() => setReady(true));
  }, []);
  useEffect(() => {
    if (!user || user.role === "admin") return;
    pollingFailure.current = false;
    const version = generation.current;
    let live = true;
    api<{ profile: typeof profile }>("/profile")
      .then((x) => {
        if (live && version === generation.current) {
          setProfile(x.profile);
          profileOrigin.current = x.profile
            ? { lat: x.profile.latitude, lng: x.profile.longitude }
            : null;
        }
      })
      .catch((e) => live && setError(e.message));
    let timer: ReturnType<typeof setTimeout>;
    let pollCount = 0;
    let pollRunning = false;
    const schedule = (delay: number) => {
      timer = setTimeout(() => void poll(), delay);
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        clearTimeout(timer);
        if (!pollRunning) schedule(0);
      }
    };
    document.addEventListener("visibilitychange", onVisibilityChange);
    const poll = async () => {
      if (!live || pollRunning) return;
      pollRunning = true;
      if (document.visibilityState === "hidden") {
        pollRunning = false;
        schedule(30000);
        return;
      }
      const orderVersion = rankingVersion.current;
      try {
        await refreshJobs(page, version, orderVersion, false);
        pollCount++;
        if (pollCount % 2 === 0)
          await refreshSideData(version, pollCount % 4 === 0);
        if (!live || version !== generation.current) return;
        if (!live || !pollingFailure.current) return;
        pollingFailure.current = false;
        setError((current) => current === "No se pudo actualizar la información. CHITA volverá a intentarlo automáticamente." ? "" : current);
        setNotice((current) => current || "Conexión recuperada; la información se actualizó.");
      } catch {
        if (live && !pollingFailure.current) {
          pollingFailure.current = true;
          setError((current) => current || "No se pudo actualizar la información. CHITA volverá a intentarlo automáticamente.");
        }
      } finally {
        pollRunning = false;
        if (live) schedule(15000 + Math.floor(Math.random() * 2000));
      }
    };
    void refresh().catch((e) => live && setError(e.message)).finally(() => {
      if (live) schedule(15000 + Math.floor(Math.random() * 2000));
    });
    return () => {
      live = false;
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [user?.id, page, rankingOrigin?.lat, rankingOrigin?.lng]);
  useEffect(() => {
    if (user?.role === "admin") setTab("overview");
    else if (user?.role === "company") setTab("home");
  }, [user?.id]);
  useEffect(() => {
    if (!user || user.role === "admin") return;
    window.scrollTo({ top: 0, behavior: "instant" });
    document.getElementById("contenido")?.focus({ preventScroll: true });
  }, [tab]);
  function rankNearby() {
    if (rankingBusy) return;
    if (!navigator.geolocation) {
      setError(
        "Puedes ordenar con la ubicación de tu perfil; este navegador no ofrece GPS.",
      );
      return;
    }
    const version = generation.current;
    setRankingBusy(true);
    navigator.geolocation.getCurrentPosition(
      (point) => {
        if (version !== generation.current) return;
        setRankingBusy(false);
        rankingVersion.current++;
        setRankingOrigin({
          lat: point.coords.latitude,
          lng: point.coords.longitude,
        });
        setPage(1);
      },
      () => {
        if (version !== generation.current) return;
        setRankingBusy(false);
        if (version === generation.current)
          setError(
            rankingOrigin
              ? "No se pudo obtener tu ubicación. El orden conserva la última ubicación GPS obtenida."
              : "No se pudo obtener tu ubicación. El orden sigue usando la ubicación de tu perfil.",
          );
      },
      { enableHighAccuracy: true, maximumAge: 30000, timeout: 12000 },
    );
  }
  async function run(fn: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Error de conexión");
      requestAnimationFrame(() => {
        const alert = document.getElementById("app-error");
        alert?.scrollIntoView({ block: "start", behavior: "instant" });
        alert?.focus({ preventScroll: true });
      });
    } finally {
      setBusy(false);
    }
  }
  function stop(notify = true) {
    gpsVersion.current++;
    if (watch.current != null) navigator.geolocation.clearWatch(watch.current);
    watch.current = null;
    if (beat.current) clearInterval(beat.current);
    beat.current = null;
    gps.current = null;
    setSharing(false);
    return notify
      ? api("/location/stop", "POST", {}).catch(() => {})
      : Promise.resolve();
  }
  useEffect(() => {
    const ended = () => {
      generation.current++;
      stop(false);
      setUser(null);
      setRankingOrigin(null);
      setRankingBusy(false);
      setSelected(null);
      setJobs([]);
      setNetwork([]);
      setNotices([]);
      setProfile(null);
      profileOrigin.current = null;
      setLinked(false);
      setTab("jobs");
      setCreating(false);
      setJobFilter("all");
      setJobSearch("");
      window.history.replaceState(null, "", "/entrar");
      setPublicPath("/entrar");
      setRegister(false);
      setError("La sesión terminó. Vuelve a entrar.");
      void session().catch(() => {});
    };
    window.addEventListener("chita-session-ended", ended);
    return () => window.removeEventListener("chita-session-ended", ended);
  }, []);
  useEffect(
    () => () => {
      if (watch.current != null)
        navigator.geolocation.clearWatch(watch.current);
      if (beat.current) clearInterval(beat.current);
    },
    [],
  );
  useEffect(() => {
    setPosition(null);
    setLocError("");
    if (!selected || !active(selected.status)) return;
    const abort = new AbortController();
    let live = true;
    async function poll() {
      try {
        const x = await api<Position>(
          "/jobs/" + selected!.id + "/location",
          "GET",
          undefined,
          abort.signal,
        );
        if (live) {
          setPosition(x);
          setLocError("");
        }
      } catch (e) {
        if (live) setLocError(e instanceof Error ? e.message : "Sin posición");
      }
    }
    void poll();
    const timer = setInterval(poll, 2500);
    return () => {
      live = false;
      abort.abort();
      clearInterval(timer);
    };
  }, [selected?.id, selected?.status]);
  useEffect(() => {
    if (sharing && (!selected || !active(selected.status))) stop();
  }, [selected?.id, selected?.status, sharing]);
  async function auth(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const r = Object.fromEntries(new FormData(e.currentTarget));
    await run(async () => {
      await session();
      await api(
        register ? "/auth/register" : "/auth/login",
        "POST",
        register
          ? {
              ...r,
              role,
              latitude: Number(r.latitude),
              longitude: Number(r.longitude),
            }
          : r,
      );
      setUser(await session());
      window.history.replaceState(null, "", "/");
      setPage(1);
      setNotice("Sesión iniciada");
    });
  }
  async function create(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const r = Object.fromEntries(new FormData(e.currentTarget));
    await run(async () => {
      const body: Record<string, unknown> = {
        ...r,
        price_cents: cents(String(r.price)),
        pickup_lat: Number(r.pickup_lat),
        pickup_lng: Number(r.pickup_lng),
        dropoff_lat: Number(r.dropoff_lat),
        dropoff_lng: Number(r.dropoff_lng),
      };
      delete body.price;
      for (const k of [
        "pickup_from",
        "pickup_to",
        "delivery_from",
        "delivery_to",
      ])
        body[k] = new Date(String(r[k])).toISOString();
      await api("/jobs", "POST", body);
      setCreating(false);
      setJobFilter("all");
      setJobSearch("");
      setPage(1);
      setTab("jobs");
      setNotice(
        body.visibility === "public"
          ? "Trabajo público publicado"
          : "Trabajo publicado para tu red",
      );
      await refresh(1);
    });
  }
  async function action(name: string, form?: HTMLFormElement) {
    if (!selected) return;
    await run(async () => {
      const note = form ? String(new FormData(form).get("note") || "") : "";
      const x = await api<Job>("/jobs/" + selected.id + "/" + name, "POST", {
        note,
      });
      if (!active(x.status)) stop();
      setSelected(x);
      setNotice("Estado actualizado");
      await refresh();
    });
  }
  function start() {
    if (!selected || !linked || !navigator.geolocation) {
      setError("Vincula HALCON y permite la ubicación");
      return;
    }
    const job = selected.id;
    const version = ++gpsVersion.current;
    setLocError("Esperando una posición del dispositivo…");
    const send = async () => {
      if (!gps.current || sending.current || version !== gpsVersion.current)
        return;
      sending.current = true;
      try {
        const p = await freshPosition(gps.current, navigator.geolocation);
        if (version !== gpsVersion.current || watch.current === null) return;
        gps.current = p;
        await api("/jobs/" + job + "/location", "POST", {
          latitude: p.coords.latitude,
          longitude: p.coords.longitude,
        });
        setLocError("");
      } catch (e) {
        setLocError(
          e instanceof Error ? e.message : "No se pudo enviar la posición",
        );
      } finally {
        sending.current = false;
      }
    };
    watch.current = navigator.geolocation.watchPosition(
      (p) => {
        gps.current = p;
        void send();
      },
      () => {
        setLocError("No se pudo obtener la posición. Revisa los permisos.");
        stop();
      },
      { enableHighAccuracy: true, maximumAge: 10000, timeout: 15000 },
    );
    beat.current = setInterval(send, 15000);
    setSharing(true);
  }
  if (!ready)
    return (
      <main className="loading" aria-live="polite">
        Preparando CHITA…
      </main>
    );
  const accountGeneration = generation.current;
  const filteredJobs = jobs.filter(j =>
    (jobFilter === "all" || (jobFilter === "active" ? active(j.status) : j.status === jobFilter)) &&
    `${j.title} ${j.pickup_address} ${j.dropoff_address}`.toLocaleLowerCase("es").includes(jobSearch.trim().toLocaleLowerCase("es"))
  );
  return (
    <div
      className={
        "app-shell" + (user && user.role !== "admin" ? " dashboard-shell" : "")
      }
      onClick={navigatePublic}
    >
      <a className="skip-link" href="#contenido">
        Saltar al contenido
      </a>
      <header className={!user ? "public-header" : ""}>
        <a className="brand" href="/" aria-label="CHITA inicio">
          <img className="platform-logo" src="/chita.svg" alt="CHITA" />
        </a>
        <span className="tagline">Entregas, paso a paso.</span>
        {user && (
          <Navigation
            role={user.role}
            tab={tab}
            unread={notices.some((n) => !n.read_at)}
            select={setTab}
          />
        )}
        {!user && (
          <div className="public-links">
            <a href="/#como-funciona">Cómo funciona</a>
            <a href="/#documentacion">Guía</a>
            <a href="/entrar">Entrar</a>
            <a className="header-cta" href="/registro">
              Crear cuenta
            </a>
          </div>
        )}
        <ThemePicker />
        {user && (
          <div className="account">
            <Avatar user={user} /><span>{user.name}</span>
            <button
              className="quiet"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  if (user.role !== "admin") await stop();
                  await api("/auth/logout", "POST", {});
                  generation.current++;
                  setProfile(null);
                  profileOrigin.current = null;
                  setTab("jobs");
                  setCreating(false);
      setJobFilter("all");
      setJobSearch("");
                  setPosition(null);
                  setUser(null);
                  setRankingOrigin(null);
                  setRankingBusy(false);
                  setSelected(null);
                  setJobs([]);
                  setNetwork([]);
                  setNotices([]);
                  setLinked(false);
                  window.history.replaceState(null, "", "/");
                  setPublicPath("/");
                  await session();
                })
              }
            >
              Salir
            </button>
          </div>
        )}
      </header>
      <main
        id="contenido"
        tabIndex={-1}
        className={
          !user && !["/entrar", "/registro"].includes(publicPath)
            ? "landing-main"
            : ""
        }
      >
        {!online && (
          <div className="connection-banner" role="status" aria-live="polite">
            <strong>Sin conexión a Internet.</strong> Comprueba la conexión; cuando vuelva, reintenta la acción que estabas realizando.
          </div>
        )}
        {error && (
          <div id="app-error" className="alert" role="alert" tabIndex={-1}>
            {error}
            <button aria-label="Cerrar error" onClick={() => setError("")}>
              ×
            </button>
          </div>
        )}
        {notice && (
          <div className="success" role="status">
            <span aria-hidden="true">✓ </span><span>{notice}</span>
            <button aria-label="Cerrar confirmación" onClick={() => setNotice("")}>×</button>
          </div>
        )}
        {!user ? (
          !["/entrar", "/registro"].includes(publicPath) ? (
            <Landing />
          ) : (
            <div className="auth-layout">
              <div className="auth-intro">
                <a href="/">← Volver a CHITA</a>
                <p className="eyebrow">TU CUENTA. TU RECORRIDO.</p>
                <h1>
                  {register
                    ? "Conecta. Coordina. Entrega."
                    : "Tu siguiente entrega empieza aquí."}
                </h1>
                <p>
                  {register
                    ? "Elige tu rol y añade los datos que tu red necesita para coordinar contigo."
                    : "Entra para ver tus trabajos, tu red y las novedades de tus entregas."}
                </p>
              </div>
              <section className="panel auth-panel" id="cuenta" tabIndex={-1}>
                <p className="auth-kicker">
                  <span aria-hidden="true">↗</span> TU SIGUIENTE PASO
                </p>
                <div className="switch">
                  <button
                    className={!register ? "chosen" : ""}
                    aria-pressed={!register}
                    disabled={busy}
                    onClick={() => {
                      window.history.replaceState(null, "", "/entrar");
                      setRegister(false);
                      setError("");
                    }}
                  >
                    Entrar
                  </button>
                  <button
                    className={register ? "chosen" : ""}
                    aria-pressed={register}
                    disabled={busy}
                    onClick={() => {
                      window.history.replaceState(null, "", "/registro");
                      setRegister(true);
                      setError("");
                    }}
                  >
                    Crear cuenta
                  </button>
                </div>
                <h2>
                  {register ? "Tu cuenta de CHITA" : "Bienvenido de nuevo"}
                </h2>
                <form onSubmit={auth} aria-busy={busy}>
                  {register && (
                    <>
                      <label>
                        Quiero usar CHITA como
                        <select
                          value={role}
                          onChange={(e) => setRole(e.target.value)}
                        >
                          <option value="courier">Repartidor</option>
                          <option value="company">Empresa</option>
                        </select>
                      </label>
                      <Field name="name" label="Tu nombre" />
                      <Field name="phone" label="Teléfono" type="tel" />
                      {role === "company" ? (
                        <Field
                          name="company_name"
                          label="Nombre de la empresa"
                        />
                      ) : (
                        <label>
                          Medio de transporte
                          <select name="vehicle_type">
                            <option value="bicycle">Bicicleta</option>
                            <option value="motorbike">Moto</option>
                            <option value="car">Coche</option>
                            <option value="van">Furgoneta</option>
                            <option value="foot">A pie</option>
                          </select>
                        </label>
                      )}
                    </>
                  )}
                  <Field name="email" label="Correo electrónico" type="email" />
                  <PasswordField register={register} />
                  {register && (
                    <>
                      <Field name="address" label="Dirección de referencia" />
                      <div className="fields">
                        <Field name="latitude" label="Latitud" type="number" />
                        <Field
                          name="longitude"
                          label="Longitud"
                          type="number"
                        />
                      </div>
                      <LocationPicker center={[23.1134, -82.3667]} />
                      <p className="muted">
                        Comprueba que las coordenadas correspondan a la
                        dirección. El rol queda definido al crear la cuenta.
                      </p>
                    </>
                  )}
                  <button className="primary" disabled={busy}>
                    {busy
                      ? "Procesando…"
                      : register
                        ? "Crear cuenta"
                        : "Entrar"}
                  </button>
                </form>
              </section>
            </div>
          )
        ) : user.role === "admin" ? (
          <AdminPanel
            tab={tab}
            select={setTab}
            endSession={() =>
              window.dispatchEvent(new Event("chita-session-ended"))
            }
          />
        ) : (
          <>
            {user.role === "courier" && (
              <div hidden={tab !== "jobs" && tab !== "offers"}>
                <CourierAvailability
                  hasActiveJob={jobs.some((j) =>
                    [
                      "accepted",
                      "picked_up",
                      "arrived",
                      "delivery_reported",
                    ].includes(j.status),
                  )}
                />
              </div>
            )}
            <section className="heading">
              <div>
                <p className="eyebrow">
                  {user.role === "company" ? "TU EMPRESA" : "TU RUTA"}
                </p>
                <h1>
                  {
                    (
                      {
                        home: "Tu centro de entregas",
                        create: "Nueva entrega",
                        guide: "Guía de uso",
                        directory: "Repartidores para tu red",
                        jobs:
                          user.role === "company"
                            ? "Organiza tus entregas"
                            : "Encuentra tu siguiente trabajo",
                        network:
                          user.role === "company"
                            ? "Tu red de repartidores"
                            : "Invitaciones a redes",
                        notifications: "Novedades de tus entregas",
                        profile: "Datos de tu cuenta",
                        nearby: "Repartidores cerca de la recogida",
                        offers: "Propuestas con tiempo de respuesta",
                        halcon: "Tu conexión con HALCON",
                      } as Record<string, string>
                    )[tab]
                  }
                </h1>
              </div>
              {user.role === "company" && (
                <button
                  className="primary"
                  onClick={() => {
                    if (tab === "create") setTab("home");
                    else { setCreating(true); setTab("create"); }
                  }}
                >
                  {tab === "create"
                    ? "Volver al inicio"
                    : "Publicar trabajo"}
                </button>
              )}
              {user.role === "courier" &&
                tab !== "halcon" &&
                tab !== "profile" && (
                  <button onClick={() => setTab("halcon")}>
                    {linked ? "Gestionar HALCON" : "Vincular con HALCON"}
                  </button>
                )}
            </section>
            {tab === "offers" && (
              <OffersPanel
                user={user}
                changed={refresh}
                openJob={(job) => {
                  setSelected(job);
                  setTab("jobs");
                }}
              />
            )}
            {tab === "nearby" && user.role === "company" && (
              <CompanyDiscovery
                jobs={jobs}
                selected={selected}
                profile={profile}
                choose={setSelected}
                changed={refresh}
              />
            )}
            {tab === "directory" && user.role === "company" && <CourierDirectory changed={refresh} />}
            {tab === "guide" && <UsageGuide company={user.role === "company"} navigate={setTab} publish={() => { setCreating(true); setTab("create"); }} />}
            {tab === "home" && <DashboardHome user={user} jobs={jobs} total={total} page={page} navigate={setTab} publish={() => { setCreating(true); setTab("create"); }} openJob={(job) => { setSelected(job); setTab("jobs"); }} />}
            {(tab === "jobs" || creating) && (
              <div hidden={tab !== "jobs" && tab !== "create"}>
                {creating && (
                  <section hidden={tab !== "create"} className="panel job-composer">
                    <p className="eyebrow">PUBLICAR · 3 PASOS EN UNA PÁGINA</p>
                    <h2>Nuevo trabajo</h2>
                    <p className="muted">
                      Elige si el trabajo es público o exclusivo de tu red. Los
                      horarios se introducen en tu zona local.
                    </p>
                    <form onSubmit={create}>
                      <h3>1. ¿Qué necesitas enviar y quién puede verlo?</h3>
                      <label>
                        Visibilidad del trabajo
                        <select name="visibility" defaultValue="network">
                          <option value="network">Exclusivo de mi red</option>
                          <option value="public">
                            Público · todos los repartidores
                          </option>
                        </select>
                      </label>
                      <p className="muted">
                        Un trabajo público muestra sus puntos, horarios y tarifa
                        a los repartidores registrados. En CHITA, el GPS del
                        repartidor sólo es visible para la empresa durante el
                        trabajo activo.
                      </p>
                      <Field name="title" label="Título del trabajo" />
                      <label>
                        Descripción
                        <textarea name="description" maxLength={3000} />
                      </label>
                      <h3>2. ¿Dónde y cuándo se recoge y se entrega?</h3>
                      <p className="muted">Elige cada punto en el mapa y comprueba su dirección. Indica un horario de inicio y fin para cada parada.</p>
                      <div className="fields">
                        {["pickup", "dropoff"].map((k) => (
                          <fieldset key={k}>
                            <legend>
                              {k === "pickup" ? "Recogida" : "Entrega"}
                            </legend>
                            <Field
                              name={k + "_address"}
                              label={
                                "Dirección de " +
                                (k === "pickup" ? "recogida" : "entrega")
                              }
                              value={
                                k === "pickup" ? profile?.address : undefined
                              }
                            />
                            <div className="fields">
                              <Field
                                name={k + "_lat"}
                                label={
                                  "Latitud de " +
                                  (k === "pickup" ? "recogida" : "entrega")
                                }
                                type="number"
                                value={
                                  k === "pickup" && profile
                                    ? String(profile.latitude)
                                    : undefined
                                }
                              />
                              <Field
                                name={k + "_lng"}
                                label={
                                  "Longitud de " +
                                  (k === "pickup" ? "recogida" : "entrega")
                                }
                                type="number"
                                value={
                                  k === "pickup" && profile
                                    ? String(profile.longitude)
                                    : undefined
                                }
                              />
                            </div>
                            <LocationPicker
                              prefix={k}
                              center={[
                                profile?.latitude ?? 0,
                                profile?.longitude ?? 0,
                              ]}
                            />
                            <Field
                              name={
                                (k === "pickup" ? "pickup" : "delivery") +
                                "_from"
                              }
                              label={
                                (k === "pickup" ? "Recogida" : "Entrega") +
                                " desde"
                              }
                              type="datetime-local"
                            />
                            <Field
                              name={
                                (k === "pickup" ? "pickup" : "delivery") + "_to"
                              }
                              label={
                                (k === "pickup" ? "Recogida" : "Entrega") +
                                " hasta"
                              }
                              type="datetime-local"
                            />
                          </fieldset>
                        ))}
                      </div>
                      <h3>3. ¿Cuánto ofreces por el trabajo?</h3>
                      <div className="fields">
                        <Field name="price" label="Tarifa acordada" />
                        <label>
                          Moneda
                          <select name="currency">
                            <option>USD</option>
                            <option>CUP</option>
                            <option>EUR</option>
                          </select>
                        </label>
                      </div>
                      <p className="muted">
                        El pago se gestiona fuera de CHITA.
                      </p>
                      <button className="primary" disabled={busy}>
                        Publicar trabajo
                      </button>
                    </form>
                  </section>
                )}
                {tab === "jobs" && <div className={"workspace " + (!selected ? "workspace-browse" : "")}>
                  <section>
                    <h2>
                      {user.role === "company"
                        ? "Mis publicaciones"
                        : "Trabajos disponibles y mis entregas"}
                    </h2>
                    {user.role === "courier" && (
                      <div className="nearby-control">
                        <p className="muted">
                          Primero tus entregas activas; después las recogidas
                          más cercanas a{" "}
                          {rankingOrigin
                            ? "tu última ubicación GPS"
                            : "la ubicación de tu perfil"}
                          . Distancia en línea recta; no es tiempo de viaje.
                        </p>
                        <button disabled={rankingBusy} onClick={rankNearby}>
                          {rankingBusy
                            ? "Buscando ubicación…"
                            : "Ordenar cerca de mí"}
                        </button>
                      </div>
                    )}
                    <div className="job-tools">
                      <label>Buscar en esta página<input type="search" value={jobSearch} onChange={e => setJobSearch(e.target.value)} placeholder="Título o dirección" /></label>
                      <label>Mostrar<select aria-label="Mostrar trabajos por estado" value={jobFilter} onChange={e => setJobFilter(e.target.value)}><option value="all">Todos los estados</option><option value="published">Disponibles</option><option value="active">En curso</option><option value="delivery_reported">Por confirmar</option><option value="completed">Completados</option><option value="cancelled">Cancelados</option></select></label>
                    </div>
                    <p className="muted">Los filtros se aplican a los trabajos de esta página. Cambia de página para revisar más resultados.</p>
                    {!jobs.length ? (
                      <div className="empty">
                        {user.role === "company"
                          ? "Publica tu primer trabajo. Puedes ofrecerlo a todos los repartidores o a tu red."
                          : "No hay trabajos disponibles. Aquí aparecerán los públicos y los exclusivos de las redes que aceptes."}
                      </div>
                    ) : (
                      <div className="joblist">
                        {filteredJobs.map((j) => (
                          <button
                            key={j.id}
                            className={
                              "jobcard " +
                              (selected?.id === j.id ? "selected" : "")
                            }
                            onClick={() => {
                              if (sharing && selected?.id !== j.id) stop();
                              setSelected(j);
                              if (
                                window.matchMedia("(max-width: 48rem)").matches
                              )
                                requestAnimationFrame(() => {
                                  detailPanel.current?.scrollIntoView({
                                    block: "start",
                                    behavior: window.matchMedia(
                                      "(prefers-reduced-motion: reduce)",
                                    ).matches
                                      ? "instant"
                                      : "smooth",
                                  });
                                  detailPanel.current?.focus({
                                    preventScroll: true,
                                  });
                                });
                            }}
                          >
                            <span className="badge">{statusLabel(j)}</span>
                            <span className="muted">
                              {j.visibility === "public"
                                ? "Público"
                                : "Exclusivo de red"}
                              {j.pickup_distance_km != null
                                ? ` · ${new Intl.NumberFormat("es", { maximumFractionDigits: 1 }).format(j.pickup_distance_km)} km hasta recogida`
                                : ""}
                            </span>
                            <strong>{j.title}</strong>
                            <span>{j.company?.name || "Mi empresa"}</span>
                            <span className="muted">
                              {j.pickup_address} → {j.dropoff_address}
                            </span>
                            <div className="cardfoot">
                              <b>{money(j)}</b>
                              <span>{date(j.pickup_from)}</span>
                            </div>
                          </button>
                        ))}
                      </div>
                    )}
                    {jobs.length > 0 && !filteredJobs.length && <div className="empty">No hay coincidencias en esta página. <button onClick={() => { setJobFilter("all"); setJobSearch(""); }}>Limpiar filtros</button></div>}
                    <div className="pagination">
                      <button
                        disabled={page === 1}
                        onClick={() => setPage((p) => p - 1)}
                      >
                        Anterior
                      </button>
                      <span>
                        Página {page} · {total} trabajos
                      </span>
                      <button
                        disabled={page * 30 >= total}
                        onClick={() => setPage((p) => p + 1)}
                      >
                        Siguiente
                      </button>
                    </div>
                  </section>
                  <section
                    hidden={!selected}
                    className="panel detail"
                    aria-label="Detalles del trabajo"
                    ref={detailPanel}
                    tabIndex={-1}
                  >
                    {selected ? (
                      <>
                        <button className="quiet" onClick={() => { if (sharing) stop(); setSelected(null); }}>← Volver a la lista</button>
                        <span className="badge">{statusLabel(selected)}</span>
                        <h2>{selected.title}</h2>
                        <p>{selected.description}</p>
                        <div className="route">
                          <div>
                            <b>Recogida</b>
                            <p>{selected.pickup_address}</p>
                            <small>
                              {date(selected.pickup_from)} —{" "}
                              {date(selected.pickup_to)}
                            </small>
                          </div>
                          <div>
                            <b>Entrega</b>
                            <p>{selected.dropoff_address}</p>
                            <small>
                              {date(selected.delivery_from)} —{" "}
                              {date(selected.delivery_to)}
                            </small>
                          </div>
                        </div>
                        <p className="price">
                          {money(selected)} <small>Pago fuera de CHITA</small>
                        </p>
                        {selected.courier && (
                          <p>
                            Repartidor: <b>{selected.courier.name}</b>
                          </p>
                        )}
                        {user.role === "company" &&
                          selected.status === "published" && (
                            <button onClick={() => setTab("nearby")}>
                              Buscar repartidores para este trabajo
                            </button>
                          )}
                        {selected.pending_offer && (
                          <p className="notice">
                            Propuesta reservada hasta{" "}
                            {date(selected.pending_offer.expires_at)}.{" "}
                            <button onClick={() => setTab("offers")}>
                              Ver propuesta
                            </button>
                          </p>
                        )}
                        {user.role === "company" &&
                          selected.assigned_courier_id && (
                            <RatingCard
                              courierID={selected.assigned_courier_id}
                              refreshKey={`${selected.id}:${selected.status}`}
                            />
                          )}
                        <Map job={selected} position={position} />
                        {active(selected.status) && (
                          <div className="tracking">
                            <p>
                              {position?.active
                                ? "Posición reciente"
                                : "Seguimiento sin posición activa"}
                              {position?.last_seen && (
                                <small>
                                  {" "}
                                  · Última señal: {date(position.last_seen)}
                                </small>
                              )}
                            </p>
                            {locError && (
                              <p className="muted" role="status">
                                {locError}
                              </p>
                            )}
                            {user.role === "courier" &&
                              selected.assigned_courier_id === user.id && (
                                <>
                                  <button
                                    onClick={() =>
                                      sharing ? void stop() : start()
                                    }
                                    disabled={!linked}
                                  >
                                    {sharing
                                      ? "Detener envío de GPS"
                                      : "Iniciar GPS de este dispositivo"}
                                  </button>
                                  <p className="muted">
                                    La empresa puede ver tu posición durante
                                    este trabajo. HALCON conserva sus permisos
                                    de destinatario y moderación. Mantén la
                                    página abierta.
                                  </p>
                                  {!linked && (
                                    <button
                                      className="quiet"
                                      onClick={() => setTab("halcon")}
                                    >
                                      Vincular HALCON
                                    </button>
                                  )}
                                </>
                              )}
                          </div>
                        )}
                        <div className="actions">
                          {user.role === "courier" &&
                            selected.status === "published" && (
                              <p className="muted">
                                Al aceptar, esta empresa podrá consultar la
                                última posición de tu HALCON vinculado mientras
                                el trabajo esté activo. El botón de GPS inicia
                                el envío desde este dispositivo.
                              </p>
                            )}
                          {user.role === "courier" &&
                            (selected.status === "published" ? (
                              <button
                                className="primary"
                                disabled={busy || expired(selected)}
                                onClick={() =>
                                  selected.pending_offer
                                    ? run(async () => {
                                        await api(
                                          `/offers/${selected.pending_offer!.id}/accept`,
                                          "POST",
                                          {},
                                        );
                                        await refresh();
                                        setSelected(
                                          await api<Job>(
                                            `/jobs/${selected.id}`,
                                          ),
                                        );
                                      })
                                    : action("accept")
                                }
                              >
                                Aceptar trabajo
                              </button>
                            ) : (
                              selected.assigned_courier_id === user.id &&
                              (
                                {
                                  accepted: (
                                    <button
                                      disabled={busy}
                                      onClick={() => action("pickup")}
                                    >
                                      Confirmar recogida
                                    </button>
                                  ),
                                  picked_up: (
                                    <button
                                      disabled={busy}
                                      onClick={() => action("arrive")}
                                    >
                                      Estoy en el punto de entrega
                                    </button>
                                  ),
                                  arrived: (
                                    <form
                                      onSubmit={(e) => {
                                        e.preventDefault();
                                        void action("report", e.currentTarget);
                                      }}
                                    >
                                      <label>
                                        Nota de entrega (opcional)
                                        <textarea
                                          name="note"
                                          maxLength={2000}
                                        />
                                      </label>
                                      <button
                                        className="primary"
                                        disabled={busy}
                                      >
                                        Notificar entrega a la empresa
                                      </button>
                                    </form>
                                  ),
                                } as Record<string, ReactNode>
                              )[selected.status]
                            ))}
                          {user.role === "company" &&
                            selected.status === "delivery_reported" && (
                              <>
                                <p>
                                  El repartidor notificó la entrega. Confírmala
                                  después de comprobarla.
                                </p>
                                <button
                                  className="primary"
                                  disabled={busy}
                                  onClick={() => action("confirm")}
                                >
                                  Confirmar entrega completada
                                </button>
                                <form
                                  onSubmit={(e) => {
                                    e.preventDefault();
                                    void action("reject", e.currentTarget);
                                  }}
                                >
                                  <Field
                                    name="note"
                                    label="Motivo para solicitar revisión"
                                  />
                                  <button disabled={busy}>
                                    Solicitar revisión
                                  </button>
                                </form>
                              </>
                            )}
                          {user.role === "company" &&
                            ["published", "accepted"].includes(
                              selected.status,
                            ) && (
                              <details>
                                <summary>Cancelar trabajo</summary>
                                <form
                                  onSubmit={(e) => {
                                    e.preventDefault();
                                    void action("cancel", e.currentTarget);
                                  }}
                                >
                                  <Field
                                    name="note"
                                    label="Motivo de cancelación"
                                  />
                                  <button disabled={busy}>
                                    Confirmar cancelación
                                  </button>
                                </form>
                              </details>
                            )}
                          {selected.cancellation_reason && (
                            <p>Cancelación: {selected.cancellation_reason}</p>
                          )}
                        </div>
                      </>
                    ) : (
                      <div className="empty">
                        Selecciona un trabajo para ver los detalles y el
                        siguiente paso.
                      </div>
                    )}
                  </section>
                </div>}
              </div>
            )}
            {tab === "network" && (
              <section className="panel">
                <h2>
                  {user.role === "company"
                    ? "Tu red de repartidores"
                    : "Tus empresas"}
                </h2>
                {user.role === "company" && (
                  <form
                    className="inline"
                    onSubmit={(e) => {
                      e.preventDefault();
                      const f = e.currentTarget;
                      run(async () => {
                        await api("/network", "POST", {
                          email: new FormData(f).get("email"),
                        });
                        f.reset();
                        setNotice("Invitación enviada");
                        await refresh();
                      });
                    }}
                  >
                    <Field
                      name="email"
                      label="Correo del repartidor registrado"
                      type="email"
                    />
                    <button className="primary" disabled={busy}>
                      Invitar
                    </button>
                  </form>
                )}
                <p className="muted">
                  La invitación requiere aceptación. Retirar a alguien no
                  cancela sus trabajos ya aceptados.
                </p>
                {network.length ? (
                  network.map((m) => (
                    <div className="row" key={m.id}>
                      <div>
                        <strong>{m.name}</strong>
                        <p>{m.email}</p>
                        <span className="badge">
                          {labels[m.status] || "En la red"}
                        </span>
                      </div>
                      {m.status !== "revoked" &&
                        (user.role === "company" ? (
                          <button
                            disabled={busy}
                            onClick={() =>
                              run(async () => {
                                await api(
                                  "/network/" + m.id + "/remove",
                                  "POST",
                                  {},
                                );
                                await refresh();
                              })
                            }
                          >
                            Retirar de la red
                          </button>
                        ) : (
                          m.status === "pending" && (
                            <button
                              className="primary"
                              disabled={busy}
                              onClick={() =>
                                run(async () => {
                                  await api(
                                    "/network/" + m.id + "/accept",
                                    "POST",
                                    {},
                                  );
                                  setNotice(
                                    "Ahora puedes ver los trabajos de esta empresa",
                                  );
                                  await refresh();
                                })
                              }
                            >
                              Aceptar invitación
                            </button>
                          )
                        ))}
                    </div>
                  ))
                ) : (
                  <div className="empty">Todavía no hay invitaciones.</div>
                )}
              </section>
            )}
            {tab === "notifications" && (
              <section className="panel">
                <h2>Avisos</h2>
                {notices.length ? (
                  notices.map((n) => (
                    <div className="row" key={n.id}>
                      <p>{n.message}</p>
                      {!n.read_at && (
                        <button
                          disabled={busy}
                          onClick={() =>
                            run(async () => {
                              await api(
                                "/notifications/" + n.id + "/read",
                                "POST",
                                {},
                              );
                              await refresh();
                            })
                          }
                        >
                          Marcar como leído
                        </button>
                      )}
                    </div>
                  ))
                ) : (
                  <div className="empty">No tienes avisos todavía.</div>
                )}
              </section>
            )}
            {(tab === "profile" ||
              (tab === "halcon" && user.role === "courier")) && (
              <section
                className={
                  tab === "profile" ? "account-layout" : "panel narrow"
                }
              >
                {tab === "profile" && (
                  <>
                    <aside className="panel account-summary">
                      <div className="account-summary-title">
                        <h2>Mi cuenta</h2>
                        <span className="account-role-tag">
                          {user.role === "company" ? "Empresa" : "Repartidor"}
                        </span>
                      </div>
                      <ProfilePhoto user={user} sessionCurrent={() => generation.current === accountGeneration} updated={avatar_url => setUser(current => current?.id === user.id ? { ...current, avatar_url } : current)} />
                      <div className="account-identity">
                        <strong>{user.name}</strong>
                        <span>{user.email}</span>
                      </div>
                      <p className="account-purpose">
                        <strong>Uso de tu ubicación</strong>
                        <span>
                        {user.role === "company"
                          ? "Tu dirección sirve como referencia para publicar recogidas y buscar repartidores cercanos."
                          : "Tu ubicación de referencia ayuda a ordenar las recogidas cercanas. No activa el envío de GPS."}
                        </span>
                      </p>
                      {user.role === "courier" && (
                        <RatingCard courierID={user.id} readonly />
                      )}
                      {user.role === "courier" && (
                        <button
                          className="account-link"
                          onClick={() => setTab("halcon")}
                        >
                          {linked ? "Gestionar HALCON" : "Vincular con HALCON"}
                        </button>
                      )}
                    </aside>
                    <div className="panel account-map-panel">
                      {profile && (
                        <ProfileLocation
                          profile={profile}
                          saved={(next) => {
                            setProfile(next);
                            profileOrigin.current = next
                              ? { lat: next.latitude, lng: next.longitude }
                              : null;
                          }}
                        />
                      )}
                    </div>
                  </>
                )}
                {user.role === "courier" && tab === "halcon" && (
                  <>
                    <h2>Seguimiento con HALCON</h2>
                    <p>
                      {linked
                        ? "Cuenta vinculada con sesión vigente"
                        : "Vincula tu cuenta personal para compartir ubicación durante un trabajo"}
                    </p>
                    <p className="muted">
                      Tu contraseña sólo inicia la sesión en HALCON. CHITA
                      guarda esa sesión cifrada durante un máximo de 25 minutos.
                      La vinculación no modifica los destinatarios ni
                      moderadores de HALCON.
                    </p>
                    <form
                      onSubmit={(e) => {
                        e.preventDefault();
                        const f = e.currentTarget;
                        run(async () => {
                          await api(
                            "/halcon",
                            "POST",
                            Object.fromEntries(new FormData(f)),
                          );
                          f.reset();
                          setLinked(true);
                          setNotice("HALCON vinculado");
                        });
                      }}
                    >
                      <Field
                        name="email"
                        label="Correo de HALCON"
                        type="email"
                      />
                      <Field
                        name="password"
                        label="Contraseña de HALCON"
                        type="password"
                      />
                      <button className="primary" disabled={busy}>
                        {linked
                          ? "Renovar sesión de HALCON"
                          : "Vincular HALCON"}
                      </button>
                    </form>
                    {linked && (
                      <button
                        className="quiet"
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            stop();
                            await api("/halcon/unlink", "POST", {});
                            setLinked(false);
                          })
                        }
                      >
                        Desvincular cuenta
                      </button>
                    )}
                    <p className="muted">
                      El navegador puede pausar el GPS en segundo plano. Esta
                      versión móvil es web adaptable; no garantiza seguimiento
                      con la pantalla bloqueada.
                    </p>
                  </>
                )}
              </section>
            )}
          </>
        )}
      </main>
      <footer>
        CHITA · Coordinación de entregas con HALCON.
        <span>Tarifas y pagos acordados entre las partes.</span>
      </footer>
    </div>
  );
}
createRoot(document.getElementById("root")!).render(<App />);
