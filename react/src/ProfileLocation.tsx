import { useEffect, useState, type FormEvent } from "react";
import { api } from "./api";
import { LocationPicker } from "./Map";
type Profile = { address: string; latitude: number; longitude: number };
export function ProfileLocation({
  profile,
  saved,
}: {
  profile: Profile;
  saved: (profile: Profile) => void;
}) {
  const [point, setPoint] = useState({
    latitude: profile.latitude,
    longitude: profile.longitude,
  });
  const [address, setAddress] = useState(profile.address);
  const [busy, setBusy] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [mapFeedback, setMapFeedback] = useState("");
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    if (dirty || busy) return;
    setPoint({ latitude: profile.latitude, longitude: profile.longitude });
    setAddress(profile.address);
  }, [profile.address, profile.latitude, profile.longitude, dirty, busy]);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy || resolving || !dirty) return;
    const next = {
      address: address.trim(),
      latitude: Number(point.latitude),
      longitude: Number(point.longitude),
    };
    if (
      next.address.length < 3 ||
      next.address.length > 255 ||
      !Number.isFinite(next.latitude) ||
      !Number.isFinite(next.longitude) ||
      Math.abs(next.latitude) > 90 ||
      Math.abs(next.longitude) > 180
    ) {
      setError("Revisa la dirección y el punto seleccionado en el mapa.");
      return;
    }
    setBusy(true);
    setError("");
    setMessage("");
    try {
      // Cloudflare's current edge policy rejects PUT before the request reaches
      // Fiber. POST is allowed through and remains protected by CSRF + auth.
      await api("/profile/location", "POST", next);
      saved(next);
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
            Ajusta el punto y revisa la dirección sugerida antes de guardar.
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
          onAddress={setAddress}
          onBusyChange={setResolving}
          onStatusChange={setMapFeedback}
        />
        <label>
          Dirección seleccionada
          <input
            name="address"
            value={address}
            required
            minLength={3}
            maxLength={255}
            disabled={busy}
            onChange={(event) => {
              setAddress(event.target.value);
              setMapFeedback("");
              changed();
            }}
          />
          {/no se pudo|no se encontró|proveedor|espera un momento/i.test(mapFeedback) && (
            <small className="map-address-feedback" role="status">
              No pudimos sugerir una dirección. Escribe o corrige la referencia;
              el punto seleccionado se conserva.
            </small>
          )}
        </label>
        <div className="location-details">
          <small className="muted">
            La dirección sugerida es aproximada; puedes corregirla o añadir una
            referencia.
          </small>
          <span className="location-coordinates" aria-label="Coordenadas seleccionadas">
            {point.latitude.toFixed(5)}, {point.longitude.toFixed(5)}
          </span>
        </div>
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
                  setAddress(profile.address);
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
              {busy ? "Guardando ubicación…" : "Guardar ubicación"}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}
