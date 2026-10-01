import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, cents, labels, type Job } from "./api";
import { CompanyChoice } from "./CompanyChoice";
import { PasswordField } from "./PasswordField";

type Account = {
  id: number;
  user_id: number;
  name: string;
  email: string;
  phone: string;
  address: string;
  latitude: number;
  longitude: number;
  company_name: string;
  vehicle_type: string;
  enabled: boolean;
  archived: boolean;
  version: number;
};
type AdminJob = Job & { archived: boolean; version: number };
type Audit = {
  id: number;
  created_at: string;
  actor_user_id: number;
  entity: string;
  entity_id: number;
  action: string;
  reason: string;
};
type RecordRow = Account | AdminJob | Audit;
type Page = { items: RecordRow[]; total: number; page: number };
const titles: Record<string, string> = {
  overview: "Control de la plataforma",
  jobs: "Trabajos",
  companies: "Empresas",
  couriers: "Repartidores",
  audits: "Historial de administración",
  security: "Seguridad de tu cuenta",
};
const actions: Record<string, string> = {
  archive: "Archivar",
  restore: "Restaurar",
  cancel: "Cancelar trabajo",
  create: "Crear",
  update: "Editar",
  password: "Contraseña",
};
const localTime = (value?: string, hours = 1) => {
  const date = value ? new Date(value) : new Date(Date.now() + hours * 3600000);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
};
const jobRow = (r: RecordRow): r is AdminJob => "title" in r;
const recordName = (r: RecordRow) =>
  jobRow(r) ? r.title : "name" in r ? r.company_name || r.name : "";
export function AdminPanel({
  tab,
  select,
  endSession,
}: {
  tab: string;
  select: (s: string) => void;
  endSession: () => void;
}) {
  const section = titles[tab] ? tab : "overview";
  const [data, setData] = useState<Page>({ items: [], total: 0, page: 1 });
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [state, setState] = useState("all");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [revision, setRevision] = useState(0);
  const [editor, setEditor] = useState<Account | AdminJob | "new" | null>(null);
  const [detail, setDetail] = useState<Account | AdminJob | null>(null);
  const [change, setChange] = useState<{
    row: Account | AdminJob;
    action: string;
  } | null>(null);
  const dialog = useRef<HTMLDialogElement>(null);
  const editorHeading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    setPage(1);
    setState("all");
    setSearch("");
    setQuery("");
    setEditor(null);
    setDetail(null);
    setChange(null);
    setError("");
    setNotice("");
  }, [section]);
  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);
  useEffect(() => {
    const abort = new AbortController();
    let live = true;
    setLoading(true);
    const request =
      section === "overview"
        ? api<Record<string, number>>(
            "/admin/overview",
            "GET",
            undefined,
            abort.signal,
          ).then((x) => {
            if (live) setCounts(x);
          })
        : section === "security"
          ? Promise.resolve()
          : api<Page>(
              `/admin/${section}?page=${page}&state=${state}&search=${encodeURIComponent(query)}`,
              "GET",
              undefined,
              abort.signal,
            ).then((x) => {
              if (live) {
                setData(x);
                if (page > 1 && !x.items.length)
                  setPage(Math.max(1, Math.ceil(x.total / 20)));
              }
            });
    request
      .catch((e) => {
        if (live) setError(e.message);
      })
      .finally(() => {
        if (live) setLoading(false);
      });
    return () => {
      live = false;
      abort.abort();
    };
  }, [section, page, query, state, revision]);
  useEffect(() => {
    if (editor) editorHeading.current?.focus();
  }, [editor]);
  useEffect(() => {
    if (change) {
      dialog.current?.showModal();
      const old = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = old;
      };
    } else dialog.current?.close();
  }, [change]);
  async function run(action: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await action();
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "No se pudo completar la operación",
      );
    } finally {
      setBusy(false);
    }
  }
  function input(
    name: string,
    label: string,
    value: string | number = "",
    type = "text",
    required = true,
  ) {
    return (
      <label key={name}>
        {label}
        <input
          name={name}
          type={type}
          required={required}
          defaultValue={value}
          maxLength={type === "text" ? 255 : undefined}
          step={type === "number" ? "any" : undefined}
        />
      </label>
    );
  }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const f = new FormData(event.currentTarget);
    const get = (n: string) => String(f.get(n) || "");
    await run(async () => {
      const version = editor && editor !== "new" ? editor.version : 0;
      let body: Record<string, unknown> = { version, reason: get("reason") };
      if (section === "jobs") {
        body = {
          ...body,
          company_id: Number(get("company_id")),
          title: get("title"),
          description: get("description"),
          visibility: get("visibility"),
          pickup_address: get("pickup_address"),
          dropoff_address: get("dropoff_address"),
          pickup_lat: Number(get("pickup_lat")),
          pickup_lng: Number(get("pickup_lng")),
          dropoff_lat: Number(get("dropoff_lat")),
          dropoff_lng: Number(get("dropoff_lng")),
          price_cents: cents(get("price")),
          currency: get("currency"),
        };
        for (const field of [
          "pickup_from",
          "pickup_to",
          "delivery_from",
          "delivery_to",
        ])
          body[field] = new Date(get(field)).toISOString();
      } else
        body = {
          ...body,
          name: get("name"),
          email: get("email"),
          phone: get("phone"),
          password: get("password"),
          address: get("address"),
          latitude: Number(get("latitude")),
          longitude: Number(get("longitude")),
          company_name: get("company_name"),
          vehicle_type: get("vehicle_type"),
          enabled: f.has("enabled"),
        };
      const url =
        `/admin/${section}` +
        (editor && editor !== "new" ? `/${editor.id}` : "");
      await api(url, editor === "new" ? "POST" : "PUT", body);
      setEditor(null);
      setDetail(null);
      setRevision((x) => x + 1);
      setNotice("Cambios guardados y registrados en el historial.");
    });
  }
  function openEdit(row: Account | AdminJob) {
    setDetail(null);
    setEditor(row);
    setError("");
    setNotice("");
  }
  const account =
    editor && editor !== "new" && !jobRow(editor) ? editor : undefined;
  const job = editor && editor !== "new" && jobRow(editor) ? editor : undefined;
  const managed = ["jobs", "companies", "couriers"].includes(section);
  return (
    <div className="admin-panel">
      <section className="heading">
        <div>
          <p className="eyebrow">ADMINISTRACIÓN CHITA</p>
          <h1>{titles[section]}</h1>
          <p className="muted">
            Gestiona registros con permisos exclusivos y trazabilidad de los
            cambios.
          </p>
        </div>
        {managed && !editor && (
          <button
            className="primary"
            onClick={() => {
              setEditor("new");
              setDetail(null);
              setError("");
            }}
          >
            Crear{" "}
            {section === "jobs"
              ? "trabajo"
              : section === "companies"
                ? "empresa"
                : "repartidor"}
          </button>
        )}
      </section>
      {error && (
        <div className="error" role="alert">
          {error}
          <button
            type="button"
            className="quiet"
            onClick={() => setRevision((x) => x + 1)}
          >
            Actualizar datos
          </button>
        </div>
      )}
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      {section === "overview" && (
        <>
          <div className="admin-stats">
            {[
              ["companies", "Empresas"],
              ["couriers", "Repartidores"],
              ["jobs", "Trabajos"],
              ["pending_confirmation", "Por confirmar"],
            ].map(([key, label]) => (
              <button
                className="panel admin-stat"
                key={key}
                onClick={() =>
                  select(key === "pending_confirmation" ? "jobs" : key)
                }
              >
                <span>{label}</span>
                <strong>{loading ? "…" : (counts[key] ?? 0)}</strong>
                <span className="muted">Abrir gestión →</span>
              </button>
            ))}
          </div>
          <section className="panel">
            <h2>Qué puedes gestionar</h2>
            <p>
              Crea cuentas de empresas y repartidores, actualiza sus datos y
              administra publicaciones públicas o exclusivas de una red.
            </p>
            <p>
              Los trabajos aceptados mantienen su tarifa y recorrido. La empresa
              sigue siendo quien confirma una entrega. No se pueden desactivar
              cuentas con entregas pendientes.
            </p>
            <p>
              Archivar retira un registro de uso sin borrar su historial.
              Restaurar una cuenta la mantiene desactivada hasta que la actives
              desde su formulario. Las invitaciones revocadas requieren un nuevo
              consentimiento.
            </p>
          </section>
        </>
      )}
      {editor && (
        <section className="panel">
          <h2 ref={editorHeading} tabIndex={-1}>
            {editor === "new"
              ? "Nuevo registro"
              : `Editar ${recordName(editor)}`}
          </h2>
          <p className="muted">
            Los campos marcados por el navegador son obligatorios. Las
            coordenadas deben corresponder a la dirección.
          </p>
          <form key={editor === "new" ? "new" : editor.id} onSubmit={save}>
            {section === "jobs" ? (
              <>
                <div className="fields">
                  <CompanyChoice
                    id={job?.company_id}
                    name={job?.company?.name}
                  />
                  {input("title", "Título", job?.title)}
                  <label>
                    Visibilidad
                    <select
                      name="visibility"
                      defaultValue={job?.visibility ?? "public"}
                    >
                      <option value="public">
                        Público para todos los repartidores
                      </option>
                      <option value="network">Sólo la red de la empresa</option>
                    </select>
                  </label>
                </div>
                <p className="muted">
                  Selecciona una empresa activa. Los trabajos aceptados
                  conservan su tarifa, sus horarios y sus direcciones.
                </p>
                <label>
                  Descripción
                  <textarea
                    name="description"
                    maxLength={3000}
                    defaultValue={job?.description}
                  />
                </label>
                <div className="admin-form-grid">
                  {input(
                    "pickup_address",
                    "Dirección de recogida",
                    job?.pickup_address,
                  )}
                  {input(
                    "dropoff_address",
                    "Dirección de entrega",
                    job?.dropoff_address,
                  )}
                  {input(
                    "pickup_lat",
                    "Latitud de recogida",
                    job?.pickup_lat ?? "",
                    "number",
                  )}
                  {input(
                    "pickup_lng",
                    "Longitud de recogida",
                    job?.pickup_lng ?? "",
                    "number",
                  )}
                  {input(
                    "dropoff_lat",
                    "Latitud de entrega",
                    job?.dropoff_lat ?? "",
                    "number",
                  )}
                  {input(
                    "dropoff_lng",
                    "Longitud de entrega",
                    job?.dropoff_lng ?? "",
                    "number",
                  )}
                  {input(
                    "pickup_from",
                    "Recogida desde",
                    localTime(job?.pickup_from),
                    "datetime-local",
                  )}
                  {input(
                    "pickup_to",
                    "Recogida hasta",
                    localTime(job?.pickup_to, 2),
                    "datetime-local",
                  )}
                  {input(
                    "delivery_from",
                    "Entrega desde",
                    localTime(job?.delivery_from, 2),
                    "datetime-local",
                  )}
                  {input(
                    "delivery_to",
                    "Entrega hasta",
                    localTime(job?.delivery_to, 3),
                    "datetime-local",
                  )}
                  {input(
                    "price",
                    "Tarifa",
                    job ? (job.price_cents / 100).toFixed(2) : "",
                    "text",
                  )}
                  <label>
                    Moneda
                    <select
                      name="currency"
                      defaultValue={job?.currency ?? "USD"}
                    >
                      {["USD", "CUP", "EUR"].map((c) => (
                        <option key={c}>{c}</option>
                      ))}
                    </select>
                  </label>
                </div>
              </>
            ) : (
              <>
                <div className="admin-form-grid">
                  {input("name", "Nombre del usuario", account?.name)}
                  {input("email", "Correo", account?.email, "email")}
                  {input("phone", "Teléfono", account?.phone, "tel")}
                  {section === "companies" ? (
                    input(
                      "company_name",
                      "Nombre de empresa",
                      account?.company_name,
                    )
                  ) : (
                    <label>
                      Transporte
                      <select
                        name="vehicle_type"
                        defaultValue={account?.vehicle_type ?? "bicycle"}
                      >
                        {[
                          ["foot", "A pie"],
                          ["bicycle", "Bicicleta"],
                          ["motorbike", "Moto"],
                          ["car", "Auto"],
                          ["van", "Furgoneta"],
                        ].map(([k, v]) => (
                          <option value={k} key={k}>
                            {v}
                          </option>
                        ))}
                      </select>
                    </label>
                  )}
                  {input("address", "Dirección", account?.address)}
                  {input(
                    "latitude",
                    "Latitud",
                    account?.latitude ?? "",
                    "number",
                  )}
                  {input(
                    "longitude",
                    "Longitud",
                    account?.longitude ?? "",
                    "number",
                  )}
                </div>
                <PasswordField
                  name="password"
                  label={
                    editor === "new"
                      ? "Contraseña inicial"
                      : "Nueva contraseña (opcional)"
                  }
                  autoComplete="new-password"
                  required={editor === "new"}
                />
                {editor !== "new" && (
                  <p className="muted">
                    Déjala vacía para conservarla. Restablecerla cierra las
                    sesiones y desvincula HALCON; el repartidor podrá volver a
                    vincularlo.
                  </p>
                )}
                <label className="admin-checkbox">
                  <input
                    type="checkbox"
                    name="enabled"
                    defaultChecked={account?.enabled ?? true}
                  />{" "}
                  Cuenta activa
                </label>
              </>
            )}
            <label>
              Motivo del cambio
              <textarea
                name="reason"
                required
                minLength={5}
                maxLength={1000}
                placeholder="Explica brevemente por qué realizas este cambio"
              />
            </label>
            <div className="actions">
              <button className="primary" disabled={busy}>
                {busy ? "Guardando…" : "Guardar cambios"}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => setEditor(null)}
              >
                Volver a la lista
              </button>
            </div>
          </form>
        </section>
      )}
      {detail && (
        <section className="panel">
          <h2>{recordName(detail)}</h2>
          <dl className="admin-detail">
            {Object.entries(detail)
              .filter(
                ([k, v]) =>
                  v != null &&
                  typeof v !== "object" &&
                  !["version", "id", "archived"].includes(k),
              )
              .map(([key, value]) => (
                <div key={key}>
                  <dt>
                    {(
                      {
                        name: "Nombre",
                        email: "Correo",
                        phone: "Teléfono",
                        address: "Dirección",
                        latitude: "Latitud",
                        longitude: "Longitud",
                        company_name: "Empresa",
                        vehicle_type: "Transporte",
                        enabled: "Activa",
                        user_id: "ID de usuario",
                        title: "Título",
                        description: "Descripción",
                        status: "Estado",
                        company_id: "ID de empresa",
                        assigned_courier_id: "ID de repartidor",
                        pickup_address: "Recogida",
                        dropoff_address: "Entrega",
                        pickup_from: "Recogida desde",
                        pickup_to: "Recogida hasta",
                        delivery_from: "Entrega desde",
                        delivery_to: "Entrega hasta",
                        visibility: "Visibilidad",
                        price_cents: "Tarifa en centavos",
                        currency: "Moneda",
                        cancellation_reason: "Motivo de cancelación",
                      } as Record<string, string>
                    )[key] ?? key}
                  </dt>
                  <dd>
                    {typeof value === "boolean"
                      ? value
                        ? "Sí"
                        : "No"
                      : String(value)}
                  </dd>
                </div>
              ))}
          </dl>
          <button onClick={() => setDetail(null)}>Cerrar detalles</button>
        </section>
      )}
      {managed && !editor && (
        <>
          <div className="admin-filters">
            <label>
              Buscar por nombre o correo
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                maxLength={100}
                placeholder={
                  section === "jobs" ? "Título o referencia" : "Nombre o correo"
                }
              />
            </label>
            <label>
              Estado
              <select
                value={state}
                onChange={(e) => {
                  setState(e.target.value);
                  setPage(1);
                }}
              >
                {(section === "jobs"
                  ? [
                      ["all", "Todos"],
                      ["published", "Disponibles"],
                      ["accepted", "Aceptados"],
                      ["picked_up", "Recogidos"],
                      ["arrived", "En destino"],
                      ["delivery_reported", "Por confirmar"],
                      ["completed", "Completados"],
                      ["cancelled", "Cancelados"],
                      ["archived", "Archivados"],
                    ]
                  : [
                      ["all", "Todas las cuentas"],
                      ["active", "Activas"],
                      ["disabled", "Desactivadas"],
                      ["archived", "Archivadas"],
                    ]
                ).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <p className="muted" aria-live="polite">
            {loading
              ? "Cargando registros…"
              : `${data.total} registros encontrados`}
          </p>
          {!loading && data.items.length === 0 && (
            <section className="panel">
              <h2>No hay registros</h2>
              <p>Prueba otra búsqueda o crea el primer registro.</p>
            </section>
          )}
          <div className="admin-records" aria-busy={loading}>
            {!loading &&
              data.items.map((row) => {
                if (!("version" in row)) return null;
                const name = recordName(row);
                return (
                  <article className="panel admin-record" key={row.id}>
                    <div>
                      <p className="eyebrow">ID {row.id}</p>
                      <h2>{name}</h2>
                      <p>
                        {jobRow(row)
                          ? `${labels[row.status] ?? row.status} · ${row.company?.name ?? `Empresa #${row.company_id}`} · ${(row.price_cents / 100).toFixed(2)} ${row.currency}`
                          : `${row.email} · ${row.enabled ? "Activa" : "Desactivada"}`}
                      </p>
                      <p className="muted">
                        {jobRow(row)
                          ? `${row.pickup_address} → ${row.dropoff_address}`
                          : row.address}
                        {row.archived ? " · Archivado" : ""}
                      </p>
                    </div>
                    <div className="actions">
                      <button
                        onClick={() =>
                          run(async () =>
                            setDetail(
                              await api<Account | AdminJob>(
                                `/admin/${section}/${row.id}`,
                              ),
                            ),
                          )
                        }
                        disabled={busy}
                      >
                        Ver detalles
                      </button>
                      {!row.archived &&
                        (!jobRow(row) ||
                          (row.status === "published" &&
                            row.assigned_courier_id == null)) && (
                          <button onClick={() => openEdit(row)}>Editar</button>
                        )}
                      {row.archived ? (
                        <button
                          onClick={() => setChange({ row, action: "restore" })}
                        >
                          Restaurar
                        </button>
                      ) : (
                        <>
                          {jobRow(row) &&
                            ["published", "accepted"].includes(row.status) && (
                              <button
                                onClick={() =>
                                  setChange({ row, action: "cancel" })
                                }
                              >
                                Cancelar trabajo
                              </button>
                            )}
                          {(!jobRow(row) ||
                            ["published", "completed", "cancelled"].includes(
                              row.status,
                            )) && (
                            <button
                              className="danger"
                              onClick={() =>
                                setChange({ row, action: "archive" })
                              }
                            >
                              Archivar
                            </button>
                          )}
                        </>
                      )}
                    </div>
                  </article>
                );
              })}
          </div>
        </>
      )}
      {section === "audits" && (
        <section className="panel">
          <p className="muted">
            Historial de acciones administrativas. No guarda contraseñas ni
            credenciales de HALCON.
          </p>
          {loading ? (
            <p>Cargando historial…</p>
          ) : (
            data.items.map(
              (row) =>
                "action" in row && (
                  <article className="admin-audit" key={row.id}>
                    <p>
                      <strong>{actions[row.action] ?? row.action}</strong> ·{" "}
                      {titles[row.entity] ?? row.entity} #{row.entity_id}
                    </p>
                    <p>{row.reason}</p>
                    <p className="muted">
                      Administrador #{row.actor_user_id} ·{" "}
                      {new Date(row.created_at).toLocaleString("es")}
                    </p>
                  </article>
                ),
            )
          )}
        </section>
      )}
      {(managed || section === "audits") && !editor && (
        <nav className="pagination" aria-label="Páginas de administración">
          <button
            disabled={loading || page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Anterior
          </button>
          <span>
            Página {page} de {Math.max(1, Math.ceil(data.total / 20))}
          </span>
          <button
            disabled={loading || page * 20 >= data.total}
            onClick={() => setPage((p) => p + 1)}
          >
            Siguiente
          </button>
        </nav>
      )}
      {section === "security" && (
        <section className="panel">
          <h2>Cambiar tu contraseña</h2>
          <p>
            Cierra todas tus sesiones administrativas después de guardar.
            Tendrás que entrar con tu nueva contraseña.
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void run(async () => {
                if (f.get("password") !== f.get("confirmation"))
                  throw new Error("Las contraseñas nuevas no coinciden");
                await api("/admin/password", "POST", {
                  current_password: f.get("current_password"),
                  password: f.get("password"),
                });
                endSession();
              });
            }}
          >
            <PasswordField
              name="current_password"
              label="Contraseña actual"
              autoComplete="current-password"
            />
            <PasswordField
              name="password"
              label="Nueva contraseña"
              autoComplete="new-password"
            />
            <PasswordField
              name="confirmation"
              label="Confirma la nueva contraseña"
              autoComplete="new-password"
            />
            <button className="primary" disabled={busy}>
              {busy ? "Guardando…" : "Cambiar contraseña"}
            </button>
          </form>
        </section>
      )}
      <dialog
        ref={dialog}
        className="admin-dialog"
        aria-labelledby="admin-change-title"
        onCancel={(e) => {
          if (busy) e.preventDefault();
        }}
        onClose={() => {
          if (!busy) setChange(null);
        }}
      >
        {change && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void run(async () => {
                const action = change.action;
                await api(
                  `/admin/${section}/${change.row.id}` +
                    (action === "archive" ? "" : `/${action}`),
                  action === "archive" ? "DELETE" : "POST",
                  { version: change.row.version, reason: f.get("reason") },
                );
                setChange(null);
                setDetail(null);
                setRevision((x) => x + 1);
                setNotice("Acción completada y registrada.");
              });
            }}
          >
            <h2 id="admin-change-title">
              {actions[change.action]}: {recordName(change.row)}
            </h2>
            <p>
              {change.action === "restore"
                ? "El registro volverá a estar disponible. Las cuentas se restauran desactivadas; actívalas desde Editar."
                : change.action === "archive"
                  ? "El registro dejará de estar disponible. Se conservará el historial y podrás restaurarlo."
                  : "Se notificará a la empresa y al repartidor. Sólo se puede cancelar antes de recoger el producto."}
            </p>
            <label>
              Motivo
              <textarea name="reason" minLength={5} maxLength={1000} required />
            </label>
            {error && (
              <p className="error" role="alert">
                {error}
              </p>
            )}
            <div className="actions">
              <button disabled={busy} className="primary">
                {busy ? "Procesando…" : `Confirmar: ${actions[change.action]}`}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => setChange(null)}
              >
                Volver
              </button>
            </div>
          </form>
        )}
      </dialog>
    </div>
  );
}
