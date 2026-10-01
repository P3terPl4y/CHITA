import { useEffect, useRef, useState } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { api, type Job, type Position } from "./api";
export function Map({
  job,
  position,
}: {
  job: Job;
  position: Position | null;
}) {
  const el = useRef<HTMLDivElement>(null);
  const map = useRef<L.Map | null>(null);
  const marker = useRef<L.CircleMarker | null>(null);
  useEffect(() => {
    if (!el.current) return;
    const m = L.map(el.current, { scrollWheelZoom: false }).setView(
      [job.pickup_lat, job.pickup_lng],
      13,
    );
    map.current = m;
    L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution:
        '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>',
      maxZoom: 19,
    }).addTo(m);
    const points: [number, number][] = [
      [job.pickup_lat, job.pickup_lng],
      [job.dropoff_lat, job.dropoff_lng],
    ];
    points.forEach((p, i) =>
      L.circleMarker(p, {
        radius: 9,
        color: i ? "#df0029" : "#171717",
        fillOpacity: 1,
      })
        .addTo(m)
        .bindTooltip(i ? "Entrega" : "Recogida"),
    );
    m.fitBounds(L.latLngBounds(points).pad(0.25), { maxZoom: 15 });
    return () => {
      m.remove();
      map.current = null;
      marker.current = null;
    };
  }, [
    job.id,
    job.pickup_lat,
    job.pickup_lng,
    job.dropoff_lat,
    job.dropoff_lng,
  ]);
  useEffect(() => {
    if (!map.current) return;
    if (position?.latitude != null && position.longitude != null) {
      if (!marker.current)
        marker.current = L.circleMarker(
          [position.latitude, position.longitude],
          { radius: 9, color: "#171717", fillColor: "#ffdc00", fillOpacity: 1 },
        )
          .addTo(map.current!)
          .bindTooltip("Última posición del repartidor");
      else marker.current.setLatLng([position.latitude, position.longitude]);
    } else {
      marker.current?.remove();
      marker.current = null;
    }
  }, [position]);
  return (
    <div
      ref={el}
      className="map"
      aria-label="Mapa de recogida, entrega y última posición disponible"
    />
  );
}

export type MapPoint = {
  latitude: number;
  longitude: number;
  label: string;
  id?: number;
};
function valid(lat: number, lng: number) {
  return (
    Number.isFinite(lat) &&
    Number.isFinite(lng) &&
    Math.abs(lat) <= 90 &&
    Math.abs(lng) <= 180
  );
}
function tiles(map: L.Map) {
  L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
    attribution:
      '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>',
    maxZoom: 19,
  }).addTo(map);
}
function tooltip(label: string) {
  const span = document.createElement("span");
  span.textContent = label;
  return span;
}
export function PointMap({
  points,
  label = "Mapa de direcciones",
  select,
}: {
  points: MapPoint[];
  label?: string;
  select?: (id: number) => void;
}) {
  const el = useRef<HTMLDivElement>(null);
  const onSelect = useRef(select);
  onSelect.current = select;
  const key = JSON.stringify(points);
  useEffect(() => {
    if (!el.current) return;
    const values: MapPoint[] = JSON.parse(key);
    const items = values.filter((p) => valid(p.latitude, p.longitude));
    if (!items.length) return;
    const m = L.map(el.current, { scrollWheelZoom: false }).setView(
      [items[0].latitude, items[0].longitude],
      14,
    );
    tiles(m);
    items.forEach((p, i) => {
      const marker = L.circleMarker([p.latitude, p.longitude], {
        radius: 9,
        color: i ? "#d60024" : "#111111",
        fillColor: i ? "#ffdc00" : "#ffffff",
        fillOpacity: 1,
      })
        .addTo(m)
        .bindTooltip(tooltip(p.label));
      if (p.id != null) marker.on("click", () => onSelect.current?.(p.id!));
    });
    if (items.length > 1)
      m.fitBounds(
        L.latLngBounds(
          items.map((p) => [p.latitude, p.longitude] as [number, number]),
        ).pad(0.2),
        { maxZoom: 15 },
      );
    const resize = new ResizeObserver(() => m.invalidateSize());
    resize.observe(el.current);
    return () => {
      resize.disconnect();
      m.remove();
    };
  }, [key]);
  return <div ref={el} className="map" role="region" aria-label={label} />;
}

export function LocationPicker({
  prefix,
  center,
  latitudeName = prefix ? prefix + "_lat" : "latitude",
  longitudeName = prefix ? prefix + "_lng" : "longitude",
  addressName = prefix ? prefix + "_address" : "address",
}: {
  prefix?: string;
  center: [number, number];
  latitudeName?: string;
  longitudeName?: string;
  addressName?: string;
}) {
  const el = useRef<HTMLDivElement>(null);
  const map = useRef<L.Map | null>(null);
  const choose = useRef<((lat: number, lng: number) => void) | null>(null);
  const [open, setOpen] = useState(true);
  const [message, setMessage] = useState(
    "Selecciona un punto en el mapa. Puedes ajustar después la dirección.",
  );
  const [busy, setBusy] = useState(false);
  const [locating, setLocating] = useState(false);
  useEffect(() => {
    if (!open || !el.current) return;
    const form = el.current.closest("form");
    if (!form) return;
    const lat = form.elements.namedItem(latitudeName) as HTMLInputElement,
      lng = form.elements.namedItem(longitudeName) as HTMLInputElement,
      address = form.elements.namedItem(addressName) as HTMLInputElement;
    if (!lat || !lng || !address) return;
    const initial: [number, number] =
      lat.value !== "" &&
      lng.value !== "" &&
      valid(Number(lat.value), Number(lng.value))
        ? [Number(lat.value), Number(lng.value)]
        : valid(...center)
          ? center
          : [23.1134, -82.3667];
    const m = L.map(el.current, { scrollWheelZoom: false }).setView(
      initial,
      14,
    );
    map.current = m;
    tiles(m);
    let marker: L.CircleMarker | null = null;
    let abort: AbortController | null = null;
    let generation = 0;
    let live = true;
    function sync() {
      if (
        lat.value === "" ||
        lng.value === "" ||
        !valid(Number(lat.value), Number(lng.value))
      )
        return;
      const point: L.LatLngTuple = [Number(lat.value), Number(lng.value)];
      if (marker) marker.setLatLng(point);
      else
        marker = L.circleMarker(point, {
          radius: 9,
          color: "#d60024",
          fillColor: "#ffdc00",
          fillOpacity: 1,
        })
          .addTo(m)
          .bindTooltip(tooltip("Ubicación seleccionada"));
      m.panTo(point);
    }
    function event(input: HTMLInputElement) {
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.dispatchEvent(new Event("change", { bubbles: true }));
    }
    async function pick(a: number, b: number) {
      if (!valid(a, b)) return;
      const current = ++generation;
      abort?.abort();
      abort = new AbortController();
      lat.value = a.toFixed(7);
      lng.value = b.toFixed(7);
      address.value = "";
      event(lat);
      event(lng);
      event(address);
      sync();
      setBusy(true);
      setMessage("Buscando la dirección de este punto…");
      try {
        const r = await api<{ address: string }>(
          `/maps/reverse?lat=${a}&lng=${b}`,
          "GET",
          undefined,
          abort.signal,
        );
        if (!live || current !== generation) return;
        if (address.value === "") {
          address.value = r.address;
          event(address);
        }
        setMessage(
          "Punto elegido. Revisa la dirección y añade detalles si hace falta.",
        );
      } catch (e) {
        if (live && current === generation)
          setMessage(
            e instanceof Error
              ? e.message
              : "No se encontró la dirección. Escríbela manualmente.",
          );
      } finally {
        if (live && current === generation) setBusy(false);
      }
    }
    choose.current = (a, b) => {
      void pick(a, b);
    };
    sync();
    m.on(
      "click",
      (e: L.LeafletMouseEvent) => void pick(e.latlng.lat, e.latlng.lng),
    );
    lat.addEventListener("input", sync);
    lng.addEventListener("input", sync);
    const resize = new ResizeObserver(() => m.invalidateSize());
    resize.observe(el.current);
    return () => {
      live = false;
      generation++;
      abort?.abort();
      resize.disconnect();
      lat.removeEventListener("input", sync);
      lng.removeEventListener("input", sync);
      choose.current = null;
      map.current = null;
      m.remove();
    };
  }, [open, latitudeName, longitudeName, addressName]);
  function gps() {
    if (locating) return;
    if (!navigator.geolocation) {
      setMessage("Este navegador no ofrece GPS. Elige un punto en el mapa.");
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (p) => {
        setLocating(false);
        choose.current?.(p.coords.latitude, p.coords.longitude);
      },
      () => {
        setLocating(false);
        setMessage(
          "No se pudo obtener el GPS. Puedes seleccionar el punto en el mapa.",
        );
      },
      { enableHighAccuracy: true, timeout: 12000, maximumAge: 15000 },
    );
  }
  return (
    <div className="location-picker">
      <div className="actions">
        <button
          type="button"
          className="quiet"
          aria-expanded={open}
          onClick={() => {
            setOpen((x) => !x);
            setBusy(false);
          }}
        >
          {open ? "Cerrar mapa" : "Elegir ubicación en el mapa"}
        </button>
        {open && (
          <>
            <button type="button" disabled={locating} onClick={gps}>
              {locating ? "Obteniendo GPS…" : "Usar mi ubicación"}
            </button>
            <button
              type="button"
              onClick={() => {
                const p = map.current?.getCenter();
                if (p) choose.current?.(p.lat, p.lng);
              }}
            >
              Seleccionar centro del mapa
            </button>
          </>
        )}
      </div>
      {open && (
        <>
          <p className="muted" role="status" aria-live="polite">
            {message}
          </p>
          <div
            ref={el}
            className="map"
            role="region"
            aria-label={
              prefix
                ? `Elegir dirección de ${prefix === "pickup" ? "recogida" : "entrega"}`
                : "Elegir dirección del perfil"
            }
            aria-busy={busy}
          />
          <small className="muted">
            Dirección aproximada con datos de OpenStreetMap. Al elegir un punto,
            sus coordenadas se consultan al proveedor de direcciones; no se
            envía tu nombre.
          </small>
        </>
      )}
    </div>
  );
}
