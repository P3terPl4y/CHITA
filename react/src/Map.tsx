import { useEffect, useRef, useState } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import type { Job, Position } from "./api";
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

// Picking a point changes only the coordinate inputs. The typed street address
// remains explicit, because a map click cannot verify a postal address.
export function LocationPicker({
  prefix,
  center,
}: {
  prefix: string;
  center: [number, number];
}) {
  const el = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open || !el.current) return;
    const form = el.current.closest("form");
    if (!form) return;
    const lat = form.elements.namedItem(prefix + "_lat") as HTMLInputElement,
      lng = form.elements.namedItem(prefix + "_lng") as HTMLInputElement;
    const initial: [number, number] =
      lat.value && lng.value ? [Number(lat.value), Number(lng.value)] : center;
    const m = L.map(el.current, { scrollWheelZoom: false }).setView(
      initial,
      13,
    );
    L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution:
        '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>',
      maxZoom: 19,
    }).addTo(m);
    let marker: L.CircleMarker | null = null;
    m.on("click", (e: L.LeafletMouseEvent) => {
      lat.value = e.latlng.lat.toFixed(7);
      lng.value = e.latlng.lng.toFixed(7);
      if (marker) marker.setLatLng(e.latlng);
      else
        marker = L.circleMarker(e.latlng, {
          radius: 9,
          color: "#16795e",
          fillOpacity: 1,
        }).addTo(m);
    });
    return () => {
      m.remove();
    };
  }, [open, prefix, center[0], center[1]]);
  return (
    <>
      <button
        type="button"
        className="quiet"
        onClick={() => setOpen((x) => !x)}
      >
        {open ? "Cerrar mapa" : "Elegir coordenadas en el mapa"}
      </button>
      {open && (
        <>
          <p className="muted">
            Pulsa el punto exacto y comprueba que coincida con la dirección.
            También puedes escribir las coordenadas.
          </p>
          <div
            ref={el}
            className="map"
            aria-label="Seleccionar punto de ubicación"
          />
        </>
      )}
    </>
  );
}
