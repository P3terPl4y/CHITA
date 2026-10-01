export function RoutePreview() {
  return (
    <figure className="route-preview">
      <div className="preview-heading"><span className="preview-symbol" aria-hidden="true">↗</span><strong>Del negocio al destino.</strong><span className="preview-label">CHITA + HALCON</span></div>
      <svg className="preview-map" viewBox="0 0 600 160" aria-hidden="true" focusable="false">
        <path className="preview-street" d="M0 40H600M0 120H600M80 0V160M240 0V160M400 0V160M560 0V160" />
        <path className="preview-road" d="M0 150L170 5M370 160L550 0" />
        <path className="preview-path" d="M80 110H190Q220 110 220 80V70Q220 40 250 40H360Q390 40 390 70V90Q390 110 420 110H520" />
        <circle className="preview-origin" cx="80" cy="110" r="13" />
        <circle className="preview-destination" cx="520" cy="110" r="13" />
        <g className="preview-rider"><circle cx="310" cy="40" r="20" /><path d="M301 40H319M312 33L319 40L312 47" /></g>
      </svg>
      <figcaption><span>Recogida <span aria-hidden="true">→</span> Entrega</span><span>Ilustración del recorrido</span></figcaption>
    </figure>
  );
}
