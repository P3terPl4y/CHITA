import { useEffect, useState, type FormEvent } from "react";
import { api } from "./api";
type Summary = {
  completed_jobs: number;
  can_rate: boolean;
  average: number;
  count: number;
  rating: { rating: number; comment: string } | null;
};
export function RatingCard({
  courierID,
  readonly = false,
  refreshKey = "",
}: {
  courierID: number;
  readonly?: boolean;
  refreshKey?: string;
}) {
  const [summary, setSummary] = useState<Summary | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState("");
  useEffect(() => {
    let live = true;
    const abort = new AbortController();
    setSummary(null);
    api<Summary>(
      `/couriers/${courierID}/rating`,
      "GET",
      undefined,
      abort.signal,
    )
      .then((x) => {
        if (live) setSummary(x);
      })
      .catch((e) => {
        if (live) setError(e.message);
      });
    return () => {
      live = false;
      abort.abort();
    };
  }, [courierID, refreshKey]);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy) return;
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const x = await api<Summary>(`/couriers/${courierID}/rating`, "POST", {
        rating: Number(data.get("rating")),
        comment: data.get("comment"),
      });
      setSummary(x);
      setNotice(
        "Calificación guardada. Cada empresa aporta una calificación al promedio.",
      );
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "No se pudo guardar la calificación",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="rating-card">
      <h3>Calificaciones del repartidor</h3>
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}
      {summary && (
        <>
          <p>
            {summary.count
              ? `${summary.average.toFixed(1)} de 5 · ${summary.count} empresas`
              : "Sin calificaciones todavía"}
          </p>
          {!readonly && !summary.can_rate ? (
            <p className="muted">
              Has confirmado {summary.completed_jobs} de las 3 entregas
              necesarias para calificar. Los trabajos sólo aceptados o
              cancelados no cuentan.
            </p>
          ) : (
            !readonly && (
              <form key={courierID} onSubmit={submit}>
                <p className="muted">
                  {summary.completed_jobs} entregas confirmadas. Puedes
                  actualizar tu calificación; no se duplica en el promedio.
                </p>
                <label>
                  Calificación
                  <select
                    name="rating"
                    defaultValue={summary.rating?.rating ?? 5}
                  >
                    {[5, 4, 3, 2, 1].map((n) => (
                      <option value={n} key={n}>
                        {n} de 5
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Comentario
                  <textarea
                    name="comment"
                    maxLength={1000}
                    defaultValue={summary.rating?.comment ?? ""}
                  />
                </label>
                <button disabled={busy}>
                  {busy
                    ? "Guardando…"
                    : summary.rating
                      ? "Actualizar calificación"
                      : "Guardar calificación"}
                </button>
              </form>
            )
          )}
        </>
      )}
      {notice && (
        <p role="status" className="notice">
          {notice}
        </p>
      )}
    </section>
  );
}
