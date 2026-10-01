import { useState, type FormEvent } from "react";
import { api } from "./api";
import { LocationPicker, PointMap } from "./Map";
type Profile = { address: string; latitude: number; longitude: number };
export function ProfileLocation({
  profile,
  saved,
}: {
  profile: Profile;
  saved: () => Promise<void>;
}) {
  const [editing, setEditing] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy) return;
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    try {
      await api("/profile/location", "PUT", {
        address: data.get("address"),
        latitude: Number(data.get("latitude")),
        longitude: Number(data.get("longitude")),
      });
      await saved();
      setEditing(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "No se pudo guardar");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="profile-location">
      <PointMap
        points={[
          {
            latitude: profile.latitude,
            longitude: profile.longitude,
            label: profile.address,
          },
        ]}
        label="Dirección de tu perfil en el mapa"
      />
      {!editing ? (
        <button onClick={() => setEditing(true)}>
          Editar dirección y ubicación
        </button>
      ) : (
        <form onSubmit={submit}>
          <h3>Ubicación de referencia</h3>
          <label>
            Dirección
            <input
              name="address"
              defaultValue={profile.address}
              required
              maxLength={255}
            />
          </label>
          <div className="fields">
            <label>
              Latitud
              <input
                name="latitude"
                type="number"
                step="any"
                defaultValue={profile.latitude}
                required
                min={-90}
                max={90}
              />
            </label>
            <label>
              Longitud
              <input
                name="longitude"
                type="number"
                step="any"
                defaultValue={profile.longitude}
                required
                min={-180}
                max={180}
              />
            </label>
          </div>
          <LocationPicker center={[profile.latitude, profile.longitude]} />
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          <div className="actions">
            <button className="primary" disabled={busy}>
              {busy ? "Guardando…" : "Guardar ubicación"}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => setEditing(false)}
            >
              Volver
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
