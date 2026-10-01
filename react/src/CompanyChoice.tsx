import { useEffect, useState } from "react";
import { api } from "./api";
type Choice = { id: number; company_name: string; email: string };
export function CompanyChoice({ id, name }: { id?: number; name?: string }) {
  const [search, setSearch] = useState("");
  const [choices, setChoices] = useState<Choice[]>([]);
  const [selected, setSelected] = useState(String(id ?? ""));
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    const abort = new AbortController();
    let live = true;
    setLoading(true);
    const timer = setTimeout(() => {
      api<{ items: Choice[] }>(
        "/admin/companies?state=active&search=" + encodeURIComponent(search),
        "GET",
        undefined,
        abort.signal,
      )
        .then((x) => {
          if (live) {
            setChoices(x.items);
            setError("");
          }
        })
        .catch((e) => {
          if (live) setError(e.message);
        })
        .finally(() => {
          if (live) setLoading(false);
        });
    }, 200);
    return () => {
      live = false;
      clearTimeout(timer);
      abort.abort();
    };
  }, [search]);
  return (
    <div>
      <label>
        Buscar empresa activa
        <input
          value={search}
          maxLength={100}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Nombre o correo"
        />
      </label>
      <label>
        Empresa del trabajo
        <select
          name="company_id"
          required
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
        >
          <option value="">Selecciona una empresa</option>
          {id && !choices.some((c) => c.id === id) && (
            <option value={id}>{name ?? `Empresa #${id}`}</option>
          )}
          {choices.map((c) => (
            <option key={c.id} value={c.id}>
              {c.company_name} · {c.email}
            </option>
          ))}
        </select>
      </label>
      <p className="muted" role="status">
        {loading
          ? "Buscando empresas…"
          : "Hasta 20 coincidencias. Escribe para encontrar otras empresas."}
      </p>
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}
    </div>
  );
}
