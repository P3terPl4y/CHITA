// A stationary watch may not emit another position. Refresh an old fix before
// a heartbeat rather than repeatedly presenting old coordinates as current.
export async function freshPosition(
  cached: GeolocationPosition | null,
  geo: Pick<Geolocation, "getCurrentPosition">,
  now = Date.now(),
): Promise<GeolocationPosition> {
  if (cached && now - cached.timestamp >= 0 && now - cached.timestamp <= 30_000)
    return cached;
  const position = await new Promise<GeolocationPosition>((resolve, reject) =>
    geo.getCurrentPosition(
      resolve,
      () =>
        reject(
          new Error(
            "No se pudo actualizar el GPS. Revisa los permisos y la señal.",
          ),
        ),
      { enableHighAccuracy: true, maximumAge: 10000, timeout: 12000 },
    ),
  );
  if (Date.now() - position.timestamp > 60_000)
    throw new Error("El dispositivo todavía no ofrece una posición reciente");
  return position;
}
