import { useEffect, useId, useRef, useState } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { api, type Job, type Position } from "./api";
type GeocodeResult = { address: string; latitude: number; longitude: number };
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
  const tilesLayer = useRef<L.TileLayer | null>(null);
  const [tilesFailed, setTilesFailed] = useState(false);
  useEffect(() => {
    if (!el.current) return;
    const m = L.map(el.current, { scrollWheelZoom: false, zoomControl: false }).setView(
      [job.pickup_lat, job.pickup_lng],
      13,
    );
    map.current = m;
    addZoomControl(m);
    const baseTiles = tiles(m);
    tilesLayer.current = baseTiles;
    baseTiles.on("tileerror", () => setTilesFailed(true));
    const points: [number, number][] = [
      [job.pickup_lat, job.pickup_lng],
      [job.dropoff_lat, job.dropoff_lng],
    ];
    points.forEach((p, i) =>
      L.marker(p, {
        icon: pointIcon(i ? "dropoff" : "pickup"),
        title: i ? "Punto de entrega" : "Punto de recogida",
        alt: i ? "Punto de entrega" : "Punto de recogida",
        keyboard: false,
      })
        .addTo(m)
        .bindTooltip(i ? "Entrega" : "Recogida"),
    );
    m.fitBounds(L.latLngBounds(points).pad(0.25), { maxZoom: 15 });
    let live = true;
    const resize = new ResizeObserver(() => { if (live) m.invalidateSize({ pan: false }); });
    resize.observe(el.current);
    return () => {
      live = false;
      resize.disconnect();
      m.remove();
      map.current = null;
      marker.current = null;
      tilesLayer.current = null;
      setTilesFailed(false);
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
    <div className="job-map">
      <div className="map-frame">
        <div
          ref={el}
          className="map chita-map"
          role="region"
          aria-label="Mapa de recogida, entrega y última posición disponible"
        />
      </div>
      <MapLegend points={[
        { latitude: job.pickup_lat, longitude: job.pickup_lng, label: "Recogida", kind: "pickup" },
        { latitude: job.dropoff_lat, longitude: job.dropoff_lng, label: "Entrega", kind: "dropoff" },
      ]} />
      {tilesFailed && (
        <p className="map-status" role="status">
          No se cargaron algunas calles. Puedes seguir consultando los puntos.
          <button type="button" onClick={() => { setTilesFailed(false); tilesLayer.current?.redraw(); }}>
            Reintentar mapa
          </button>
        </p>
      )}
    </div>
  );
}

export type MapPoint = {
  latitude: number;
  longitude: number;
  label: string;
  id?: number;
  kind?: "origin" | "pickup" | "dropoff" | "courier" | "account";
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
  return L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
    attribution:
      '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>',
    maxZoom: 19,
    className: "chita-map-tiles",
  }).addTo(map);
}
function addZoomControl(map: L.Map) {
  L.control.zoom({
    position: "topright",
    zoomInTitle: "Acercar el mapa",
    zoomOutTitle: "Alejar el mapa",
  }).addTo(map);
}
function tooltip(label: string) {
  const span = document.createElement("span");
  span.textContent = label;
  return span;
}
function markerKind(point: MapPoint, index: number): NonNullable<MapPoint["kind"]> {
  if (point.kind) return point.kind;
  if (/repartidor/i.test(point.label)) return "courier";
  return index === 0 ? "origin" : "dropoff";
}
function kindLabel(kind: NonNullable<MapPoint["kind"]>) {
  return kind === "pickup" ? "recogida"
    : kind === "dropoff" ? "entrega"
      : kind === "courier" ? "repartidor"
        : kind === "account" ? "empresa" : "origen";
}
function pointIcon(kind: NonNullable<MapPoint["kind"]>, selected = false) {
  return L.divIcon({
    className: `chita-point-icon chita-point-icon--${kind}${selected ? " is-selected" : ""}`,
    html: '<span aria-hidden="true"></span>',
    iconSize: [42, 48],
    iconAnchor: [21, 42],
  });
}
export function PointMap({
  points,
  label = "Mapa de direcciones",
  select,
  selectedId = null,
  viewKey = "default",
}: {
  points: MapPoint[];
  label?: string;
  select?: (id: number) => void;
  selectedId?: number | null;
  viewKey?: string | number;
}) {
  const el = useRef<HTMLDivElement>(null);
  const map = useRef<L.Map | null>(null);
  const markers = useRef<L.LayerGroup | null>(null);
  const tilesLayer = useRef<L.TileLayer | null>(null);
  const initialViewKey = useRef<string | number | null>(null);
  const [tilesFailed, setTilesFailed] = useState(false);
  const onSelect = useRef(select);
  onSelect.current = select;
  const key = JSON.stringify(points);
  useEffect(() => {
    if (!el.current) return;
    const m = L.map(el.current, { scrollWheelZoom: false, zoomControl: false }).setView(
      [23.1134, -82.3667],
      12,
    );
    map.current = m;
    addZoomControl(m);
    const baseTiles = tiles(m);
    tilesLayer.current = baseTiles;
    baseTiles.on("tileerror", () => setTilesFailed(true));
    markers.current = L.layerGroup().addTo(m);
    let live = true;
    const resize = new ResizeObserver(() => { if (live) m.invalidateSize({ pan: false }); });
    resize.observe(el.current);
    return () => {
      live = false;
      resize.disconnect();
      m.remove();
      map.current = null;
      markers.current = null;
      tilesLayer.current = null;
      initialViewKey.current = null;
      setTilesFailed(false);
    };
  }, []);
  useEffect(() => {
    const m = map.current;
    const layer = markers.current;
    if (!m || !layer) return;
    const items = (JSON.parse(key) as MapPoint[]).filter((p) => valid(p.latitude, p.longitude));
    layer.clearLayers();
    items.forEach((p, i) => {
      const kind = markerKind(p, i);
      const accessibleLabel = p.id == null
        ? `${kindLabel(kind)}: ${p.label}`
        : `Seleccionar repartidor: ${p.label}`;
      const item = L.marker([p.latitude, p.longitude], {
        icon: pointIcon(kind, p.id != null && p.id === selectedId),
        title: accessibleLabel,
        alt: accessibleLabel,
        keyboard: p.id != null,
        riseOnHover: true,
        zIndexOffset: p.id != null && p.id === selectedId ? 1000 : 0,
      }).addTo(layer).bindTooltip(tooltip(p.label), { direction: "top", offset: [0, -18] });
      if (p.id != null) {
        item.getElement()?.setAttribute("aria-label", accessibleLabel);
        item.on("click", () => onSelect.current?.(p.id!));
        item.getElement()?.addEventListener("keydown", (event: KeyboardEvent) => {
          if (event.key !== "Enter" && event.key !== " ") return;
          event.preventDefault();
          event.stopPropagation();
          onSelect.current?.(p.id!);
        });
      }
    });
    if (items.length && initialViewKey.current !== viewKey) {
      if (items.length > 1) {
        m.fitBounds(
          L.latLngBounds(items.map((p) => [p.latitude, p.longitude] as [number, number])).pad(0.2),
          { maxZoom: 15, animate: false },
        );
      } else {
        m.setView([items[0].latitude, items[0].longitude], 14, { animate: false });
      }
      initialViewKey.current = viewKey;
    }
  }, [key, selectedId, viewKey]);
  return (
    <div className="point-map">
      <div className="map-frame">
        <div ref={el} className="map chita-map" role="region" aria-label={label} />
      </div>
      {tilesFailed && (
        <p className="map-status" role="status">
          No se cargaron algunas calles; las ubicaciones siguen visibles.
          <button type="button" onClick={() => { setTilesFailed(false); tilesLayer.current?.redraw(); }}>
            Reintentar mapa
          </button>
        </p>
      )}
      <MapLegend points={points} />
    </div>
  );
}

function MapLegend({ points }: { points: MapPoint[] }) {
  const items = new globalThis.Map<string, { kind: NonNullable<MapPoint["kind"]>; label: string }>();
  points.filter((p) => valid(p.latitude, p.longitude)).forEach((p, index) => {
    const kind = markerKind(p, index);
    const label = kindLabel(kind).replace(/^./, (char) => char.toUpperCase());
    items.set(kind, { kind, label });
  });
  if (!items.size) return null;
  return (
    <ul className="map-legend" aria-label="Leyenda del mapa">
      {[...items.values()].map(({ kind, label }) => (
        <li key={kind}><span className={`map-legend-mark map-legend-mark--${kind}`} aria-hidden="true" />{label}</li>
      ))}
    </ul>
  );
}

export function LocationPicker({
  prefix,
  center,
  latitudeName = prefix ? prefix + "_lat" : "latitude",
  longitudeName = prefix ? prefix + "_lng" : "longitude",
  addressName = prefix ? prefix + "_address" : "address",
  compact = false,
  disabled = false,
  onPick,
  onAddress,
  onBusyChange,
  onStatusChange,
}: {
  prefix?: string;
  center: [number, number];
  latitudeName?: string;
  longitudeName?: string;
  addressName?: string;
  compact?: boolean;
  disabled?: boolean;
  onPick?: (point: { latitude: number; longitude: number }) => void;
  onAddress?: (address: string) => void;
  onBusyChange?: (busy: boolean) => void;
  onStatusChange?: (message: string) => void;
}) {
  const el = useRef<HTMLDivElement>(null);
  const map = useRef<L.Map | null>(null);
  const tilesLayer = useRef<L.TileLayer | null>(null);
  const choose = useRef<((lat: number, lng: number, address?: string) => void) | null>(null);
  const searchAbort = useRef<AbortController | null>(null);
  const searchGeneration = useRef(0);
  const searchID = useId();
  const callbacks = useRef({ disabled, onPick, onAddress, onBusyChange, onStatusChange });
  callbacks.current = { disabled, onPick, onAddress, onBusyChange, onStatusChange };
  const [open, setOpen] = useState(true);
  const [message, setMessage] = useState(
    "Toca el mapa o usa el GPS para colocar el pin; puedes arrastrarlo para ajustar.",
  );
  const [busy, setBusy] = useState(false);
  const [locating, setLocating] = useState(false);
  const [tilesFailed, setTilesFailed] = useState(false);
  const [hasPoint, setHasPoint] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<GeocodeResult[]>([]);
  const [searchError, setSearchError] = useState("");
  function updateMessage(value: string) {
    setMessage(value);
    callbacks.current.onStatusChange?.(value);
  }
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
    const m = L.map(el.current, { scrollWheelZoom: false, zoomControl: false }).setView(
      initial,
      14,
    );
    map.current = m;
    addZoomControl(m);
    const baseTiles = tiles(m);
    tilesLayer.current = baseTiles;
    baseTiles.on("tileerror", () => setTilesFailed(true));
    let marker: L.Marker | null = null;
    let abort: AbortController | null = null;
    let generation = 0;
    let live = true;
    function sync() {
      if (!live) return;
      if (
        lat.value === "" ||
        lng.value === "" ||
        !valid(Number(lat.value), Number(lng.value))
      ) {
        setHasPoint(false);
        return;
      }
      setHasPoint(true);
      const point: L.LatLngTuple = [Number(lat.value), Number(lng.value)];
      if (marker) marker.setLatLng(point);
      else
        marker = L.marker(point, {
          draggable: true,
          autoPan: true,
          keyboard: true,
          title: "Pin de ubicación; arrástralo para ajustar el punto",
          alt: "Ubicación seleccionada",
          icon: L.divIcon({
            className: "chita-location-pin",
            html: '<span aria-hidden="true"></span>',
            iconSize: [36, 46],
            iconAnchor: [18, 44],
          }),
        })
          .addTo(m)
          .bindTooltip(tooltip("Ubicación seleccionada · arrastra para ajustar"));
      marker.off("dragend");
      marker.on("dragend", () => {
        const selected = marker!.getLatLng();
        void pick(selected.lat, selected.lng);
      });
      m.panTo(point, { animate: false });
    }
    function event(input: HTMLInputElement) {
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.dispatchEvent(new Event("change", { bubbles: true }));
    }
    async function pick(a: number, b: number, suggestedAddress?: string) {
      if (!valid(a, b) || callbacks.current.disabled) return;
      callbacks.current.onPick?.({
        latitude: Number(a.toFixed(7)),
        longitude: Number(b.toFixed(7)),
      });
      const current = ++generation;
      abort?.abort();
      abort = new AbortController();
      lat.value = a.toFixed(7);
      lng.value = b.toFixed(7);
      if (marker) marker.setLatLng([a, b]);
      else sync();
      address.value = suggestedAddress ?? "";
      event(lat);
      event(lng);
      event(address);
      callbacks.current.onAddress?.(suggestedAddress ?? "");
      if (suggestedAddress) {
        setBusy(false);
        callbacks.current.onBusyChange?.(false);
        updateMessage("Coincidencia seleccionada. Comprueba el pin y la dirección antes de guardar.");
        return;
      }
      setBusy(true);
      callbacks.current.onBusyChange?.(true);
      updateMessage("Buscando la dirección aproximada…");
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
          callbacks.current.onAddress?.(r.address);
        }
        updateMessage(
          "Punto elegido. Revisa la dirección y añade detalles si hace falta.",
        );
      } catch (e) {
        if (live && current === generation)
          updateMessage(
            e instanceof Error
              ? e.message
              : "No se encontró la dirección. Escríbela manualmente.",
          );
      } finally {
        if (live && current === generation) {
          setBusy(false);
          callbacks.current.onBusyChange?.(false);
        }
      }
    }
    choose.current = (a, b, address) => {
      void pick(a, b, address);
    };
    sync();
    m.on(
      "click",
      (e: L.LeafletMouseEvent) => void pick(e.latlng.lat, e.latlng.lng),
    );
    lat.addEventListener("input", sync);
    lng.addEventListener("input", sync);
    const resize = new ResizeObserver(() => { if (live) m.invalidateSize({ pan: false }); });
    resize.observe(el.current);
    return () => {
      live = false;
      generation++;
      abort?.abort();
      searchAbort.current?.abort();
      callbacks.current.onBusyChange?.(false);
      resize.disconnect();
      lat.removeEventListener("input", sync);
      lng.removeEventListener("input", sync);
      choose.current = null;
      map.current = null;
      tilesLayer.current = null;
      setTilesFailed(false);
      m.remove();
    };
  }, [open, latitudeName, longitudeName, addressName]);
  async function searchPlaces() {
    const query = searchQuery.trim();
    if (query.length < 3) {
      setSearchError("Escribe al menos 3 caracteres para buscar.");
      setSearchResults([]);
      return;
    }
    const version = ++searchGeneration.current;
    searchAbort.current?.abort();
    const controller = new AbortController();
    searchAbort.current = controller;
    setSearching(true);
    setSearchError("");
    setSearchResults([]);
    try {
      const results = await api<GeocodeResult[]>(
        "/maps/search?q=" + encodeURIComponent(query),
        "GET",
        undefined,
        controller.signal,
      );
      if (version !== searchGeneration.current) return;
      setSearchResults(results);
      if (!results.length)
        setSearchError("No encontramos lugares con ese texto. Puedes marcar el punto manualmente.");
    } catch (error) {
      if (controller.signal.aborted || version !== searchGeneration.current) return;
      setSearchError(error instanceof Error ? error.message : "No se pudo buscar. Puedes marcarlo en el mapa.");
    } finally {
      if (version === searchGeneration.current) setSearching(false);
    }
  }
  function gps() {
    if (locating || disabled) return;
    if (!navigator.geolocation) {
      updateMessage("Este navegador no ofrece GPS. Elige un punto en el mapa.");
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
        updateMessage(
          "No se pudo obtener el GPS. Puedes seleccionar el punto en el mapa.",
        );
      },
      { enableHighAccuracy: true, timeout: 12000, maximumAge: 15000 },
    );
  }
  return (
    <div
      className={
        "location-picker" + (compact ? " location-picker--compact" : "")
      }
    >
      <p className="map-instructions">
        <strong>Elige el punto de la dirección.</strong> Toca el mapa o arrastra
        el pin. Puedes afinarlo con el teclado y confirmar el centro visible.
      </p>
      {open && (
        <div className="map-search" role="search" aria-label="Buscar dirección o lugar">
          <label htmlFor={searchID}>Buscar una calle o lugar</label>
          <div className="map-search-row">
            <input
              id={searchID}
              type="search"
              autoComplete="off"
              value={searchQuery}
              onChange={(event) => {
                searchGeneration.current++;
                searchAbort.current?.abort();
                setSearching(false);
                setSearchQuery(event.target.value);
                setSearchError("");
                setSearchResults([]);
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  void searchPlaces();
                }
              }}
            />
            <button className="map-action-secondary" type="button" disabled={searching || busy || disabled} onClick={() => void searchPlaces()}>
              {searching ? "Buscando…" : "Buscar"}
            </button>
          </div>
          <small className="muted">Enviaremos sólo el texto de búsqueda al proveedor; no incluyas nombres ni datos personales.</small>
          {searchError && <p className="map-search-feedback" role="status">{searchError}</p>}
          {searchResults.length > 0 && (
            <ul className="map-search-results" id={`${searchID}-results`} aria-label="Resultados de búsqueda">
              {searchResults.map((result, index) => (
                <li key={`${result.latitude}:${result.longitude}:${index}`}>
                  <button type="button" onClick={() => {
                    setSearchResults([]);
                    setSearchQuery(result.address);
                    setSearchError("");
                    choose.current?.(result.latitude, result.longitude, result.address);
                  }}>
                    <span>{result.address}</span>
                    <small>{result.latitude.toFixed(5)}, {result.longitude.toFixed(5)}</small>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
      <div className="map-actions">
        {!compact && (
          <button
            type="button"
            className="map-action-secondary"
            aria-expanded={open}
            onClick={() => {
              if (open) {
                searchGeneration.current++;
                searchAbort.current?.abort();
                setSearching(false);
                setSearchResults([]);
              }
              setOpen((x) => !x);
              setBusy(false);
            }}
          >
            {open ? "Cerrar mapa" : "Elegir ubicación en el mapa"}
          </button>
        )}
        {open && (
          <>
            <button className="map-action-primary" type="button" disabled={locating || disabled} onClick={gps}>
              {locating ? "Buscando GPS…" : "Usar mi ubicación"}
            </button>
            <button
              type="button"
              className="map-action-secondary"
              disabled={disabled}
              onClick={() => {
                const p = map.current?.getCenter();
                if (p) choose.current?.(p.lat, p.lng);
              }}
            >
              Colocar pin en el centro
            </button>
          </>
        )}
      </div>
      {open && (
        <>
          <p className="muted" role="status" aria-live="polite">
            {message}
          </p>
          <div className="map-frame">
            <div
              ref={el}
              className={"map chita-map" + (disabled ? " map-saving" : "")}
              role="region"
              aria-label={
                prefix
                  ? `Elegir dirección de ${prefix === "pickup" ? "recogida" : "entrega"}`
                  : "Elegir dirección del perfil"
              }
              aria-busy={busy}
            />
          </div>
          {hasPoint && (
            <div className="map-selection-summary" aria-live="polite">
              <span className="map-selection-mark" aria-hidden="true" />
              <span>Pin seleccionado</span>
              <span className="map-selection-help">Arrástralo para afinar el punto</span>
            </div>
          )}
          {tilesFailed && (
            <p className="map-status" role="status">
              No cargaron algunas calles. El pin sigue disponible; puedes marcar el punto igualmente.
              <button type="button" onClick={() => { setTilesFailed(false); tilesLayer.current?.redraw(); }}>
                Reintentar mapa
              </button>
            </p>
          )}
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
