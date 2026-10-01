import { useState, type FormEvent } from "react";
import { api } from "./api";
import { LocationPicker } from "./Map";
type Profile = { address: string; latitude: number; longitude: number };
export function ProfileLocation({
  profile,
  saved,
}: {
  profile: Profile;
  saved: () => Promise<void>;
}) {
  const [point, setPoint] = useState({
    latitude: profile.latitude,
    longitude: profile.longitude,
  });
  const [busy, setBusy] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [revision, setRevision] = useState(0);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy || resolving || !dirty) return;
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await api("/profile/location", "PUT", {
        address: data.get("address"),
        latitude: Number(data.get("latitude")),
        longitude: Number(data.get("longitude")),
      });
      await saved();
      setDirty(false);
      setMessage("Ubicación guardada");
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "No se pudo guardar. Conservamos tu selección para que puedas intentarlo otra vez.",
      );
    } finally {
      setBusy(false);
    }
  }
  function changed() {
    setDirty(true);
    setError("");
    setMessage("");
  }
  return (
    <section
      className="profile-location"
      aria-label="Editar ubicación de la cuenta"
    >
      <div className="location-heading">
        <span className="step-token" aria-hidden="true">
          ◎
        </span>
        <div>
          <h3>Tu ubicación</h3>
          <p className="muted">
            Toca el punto correcto en el mapa y guarda. También puedes usar tu
            ubicación actual.
          </p>
        </div>
      </div>
      <form key={revision} onSubmit={submit}>
        <input type="hidden" name="latitude" value={point.latitude} readOnly />
        <input
          type="hidden"
          name="longitude"
          value={point.longitude}
          readOnly
        />
        <LocationPicker
          center={[profile.latitude, profile.longitude]}
          compact
          disabled={busy}
          onPick={(selected) => {
            setPoint(selected);
            changed();
          }}
          onBusyChange={setResolving}
        />
        <label>
          Dirección seleccionada
          <input
            name="address"
            defaultValue={profile.address}
            required
            minLength={3}
            maxLength={255}
            disabled={busy}
            onChange={changed}
          />
        </label>
        <small className="muted">
          La dirección es aproximada. Corrígela o añade una referencia si hace
          falta.
        </small>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
        {message && (
          <p className="notice" role="status">
            {message}
          </p>
        )}
        <div className="location-savebar">
          <span className="muted">
            {resolving
              ? "Buscando dirección…"
              : dirty
                ? "Tienes cambios sin guardar"
                : "Ubicación guardada en tu cuenta"}
          </span>
          <div className="actions">
            {dirty && (
              <button
                type="button"
                disabled={busy}
                onClick={() => {
                  setPoint({
                    latitude: profile.latitude,
                    longitude: profile.longitude,
                  });
                  setRevision((r) => r + 1);
                  setDirty(false);
                  setError("");
                  setMessage("");
                }}
              >
                Descartar cambios
              </button>
            )}
            <button className="primary" disabled={busy || resolving || !dirty}>
              {busy ? "Guardando…" : "Guardar ubicación"}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}
