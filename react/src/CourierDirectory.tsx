import { useEffect, useState } from "react";
import { api } from "./api";
import { Avatar } from "./ProfilePhoto";
type Courier = { id: number; name: string; avatar_url: string; vehicle_type: string; average_rating: number; rating_count: number; membership: string };
const vehicles: Record<string, string> = { foot: "A pie", bicycle: "Bicicleta", motorbike: "Moto", car: "Auto", van: "Furgoneta" };
export function CourierDirectory({ changed }: { changed: () => Promise<void> }) {
  const [search, setSearch] = useState(""), [query, setQuery] = useState(""), [page, setPage] = useState(1), [rows, setRows] = useState<Courier[]>([]), [total, setTotal] = useState(0), [loading, setLoading] = useState(true), [busy, setBusy] = useState<number | null>(null), [error, setError] = useState(""), [notice, setNotice] = useState(""), [revision, setRevision] = useState(0);
  useEffect(() => {
    const abort = new AbortController(); setLoading(true); setError(""); setRows([]);
    api<{ items: Courier[]; total: number }>(`/couriers/directory?page=${page}&search=${encodeURIComponent(query)}`, "GET", undefined, abort.signal)
      .then(data => { if (!abort.signal.aborted) { setRows(data.items); setTotal(data.total); } })
      .catch(e => { if (!abort.signal.aborted) setError(e instanceof Error ? e.message : "No se pudo cargar el directorio"); })
      .finally(() => { if (!abort.signal.aborted) setLoading(false); });
    return () => abort.abort();
  }, [query, page, revision]);
  async function invite(courier: Courier) {
    if (busy !== null) return;
    setBusy(courier.id); setError(""); setNotice("");
    try {
      await api("/network", "POST", { courier_id: courier.id });
      setRows(old => old.map(row => row.id === courier.id ? { ...row, membership: "pending" } : row));
      setNotice(`Invitación enviada a ${courier.name}. Debe aceptarla para formar parte de tu red.`);
      try { await changed(); } catch { setError("La invitación se envió, pero no se pudo actualizar tu red. Abre Mi red para comprobarla."); }
    } catch (e) { setError(e instanceof Error ? e.message : "No se pudo enviar la invitación"); }
    finally { setBusy(null); }
  }
  return <section className="panel courier-directory" aria-label="Directorio de repartidores">
    <h2>Encuentra personas para tu red</h2><p className="muted">Consulta repartidores registrados e invítalos a afiliarse. Este directorio no indica disponibilidad ni muestra ubicaciones. Para ofrecer una entrega, usa Buscar repartidores.</p>
    <form className="directory-search" onSubmit={e => { e.preventDefault(); setPage(1); setQuery(search.trim()); setRevision(r => r+1); setNotice(""); }}><label>Buscar repartidor por nombre<input type="search" value={search} maxLength={100} onChange={e => setSearch(e.target.value)} placeholder="Nombre del repartidor" /></label><button disabled={loading}>Buscar</button></form>
    {error && <div role="alert" className="alert">{error}<button onClick={() => setRevision(r => r+1)}>Volver a cargar</button></div>}{notice && <p role="status" className="success">{notice}</p>}
    {loading ? <p role="status">Cargando repartidores…</p> : !error && <>
      <p className="muted">{total} {total === 1 ? "repartidor encontrado" : "repartidores encontrados"}</p>
      <div className="directory-grid">{rows.map(courier => <article className="courier-card" key={courier.id}><div className="courier-identity"><Avatar user={courier} /><div><h3>{courier.name}</h3><p className="muted">{vehicles[courier.vehicle_type] || "Vehículo sin especificar"}</p></div></div><p>{courier.rating_count ? <><span aria-hidden="true">★ </span><strong>{courier.average_rating.toFixed(1)} / 5</strong> · {courier.rating_count} {courier.rating_count === 1 ? "calificación" : "calificaciones"}</> : "Sin calificaciones todavía"}</p><button disabled={busy !== null || ["pending", "accepted"].includes(courier.membership)} onClick={() => void invite(courier)}>{courier.membership === "accepted" ? "En tu red" : courier.membership === "pending" ? "Invitación enviada" : busy === courier.id ? "Enviando…" : "Invitar a mi red"}</button></article>)}</div>
      {!rows.length && <div className="empty">No encontramos repartidores con ese nombre. Prueba con otro o borra la búsqueda.</div>}
      <div className="pagination"><button disabled={page === 1} onClick={() => setPage(p => p-1)}>Anterior</button><span>Página {page}</span><button disabled={page*30 >= total} onClick={() => setPage(p => p+1)}>Siguiente</button></div>
    </>}
  </section>;
}
