import { RoutePreview } from "./RoutePreview";

export function Landing() {
  return <div className="landing">
    <section className="landing-hero" aria-labelledby="hero-title">
      <div className="intro">
        <p className="eyebrow">EMPRESAS + REPARTIDORES</p>
        <h1 id="hero-title">Publica. Reparte. <span className="hero-accent">Entrega.</span></h1>
        <p>Coordina cada entrega desde la publicación hasta la confirmación. Elige tu recorrido y empieza con los datos claros desde el primer paso.</p>
        <div className="hero-actions"><a className="cta-yellow" href="/registro?rol=empresa">Necesito repartir <span aria-hidden="true">↗</span></a><a className="cta-outline" href="/registro?rol=repartidor">Quiero repartir <span aria-hidden="true">→</span></a></div>
        <a className="hero-text-link" href="#como-funciona">Ver cómo funciona <span aria-hidden="true">↓</span></a>
      </div>
      <div className="landing-visual">
        <div className="visual-caption"><span>DE LA RECOGIDA A LA ENTREGA</span><span aria-hidden="true">↗</span></div>
        <RoutePreview />
        <div className="visual-steps" aria-label="Etapas del trabajo"><span>01 · Publica</span><span>02 · Reparte</span><span>03 · Confirma</span></div>
        <p>Ubicación mediante HALCON durante el trabajo activo.</p>
      </div>
    </section>
    <section className="landing-section pathways" id="para-quien" aria-labelledby="roles-title">
      <div className="pathways-heading"><div><p className="eyebrow">ELIGE TU RECORRIDO</p><h2 id="roles-title">Dos formas de poner una entrega en marcha.</h2></div><p>La publicación muestra puntos, horarios y tarifa antes de que el repartidor acepte.</p></div>
      <div className="role-cards">
        <article className="role-card pathway-card" id="empresa"><span className="pathway-index">01 / EMPRESA</span><span className="role-icon" aria-hidden="true">↗</span><h3>Organiza la entrega desde tu negocio.</h3><ol><li>Guarda tu dirección y publica recogida, destino, horarios y tarifa.</li><li>Elige si el trabajo es público, de tu red o una propuesta directa.</li><li>Revisa el avance y confirma la entrega cuando la hayas comprobado.</li></ol><a href="/registro?rol=empresa">Crear cuenta de empresa <span aria-hidden="true">↗</span></a></article>
        <article className="role-card pathway-card courier-path" id="repartidor"><span className="pathway-index">02 / REPARTIDOR</span><span className="role-icon yellow" aria-hidden="true">→</span><h3>Encuentra un trabajo y completa el recorrido.</h3><ol><li>Revisa trabajos públicos, de tu red o propuestas recibidas.</li><li>Comprueba dirección, horario y tarifa antes de aceptar.</li><li>Registra recogida y llegada; luego informa la entrega.</li></ol><a href="/registro?rol=repartidor">Crear cuenta de repartidor <span aria-hidden="true">↗</span></a></article>
      </div>
      <p className="pathways-footnote">La empresa confirma el cierre. CHITA no procesa pagos ni garantiza tiempos de entrega.</p>
    </section>
    <section className="landing-section steps-section" id="como-funciona" aria-labelledby="steps-title">
      <p className="eyebrow">UN FLUJO CLARO</p><h2 id="steps-title">Publica. Acepta. Confirma.</h2>
      <ol className="landing-steps">
        <li><span>01</span><h3>Elige tu rol</h3><p>Crea tu cuenta. Puedes empezar con trabajos públicos o aceptar invitaciones para acceder a trabajos exclusivos.</p></li>
        <li><span>02</span><h3>Acuerda el recorrido</h3><p>Consulta los puntos, las ventanas de horario y la tarifa antes de aceptar.</p></li>
        <li><span>03</span><h3>Cierra la entrega</h3><p>El repartidor informa la entrega y la empresa la confirma o solicita una revisión.</p></li>
      </ol>
    </section>
    <section className="landing-section honest-section" aria-labelledby="tracking-title">
      <div><p className="eyebrow">CONECTADO CON HALCON</p><h2 id="tracking-title">La ubicación tiene contexto.</h2><p>La empresa puede consultar la última posición del repartidor vinculado durante un trabajo activo. Compartirla requiere permiso de ubicación, conexión y la app abierta.</p></div>
      <div className="limits-card"><h3>Lo que debes saber</h3><p>CHITA coordina trabajos. No procesa pagos ni garantiza tiempos de entrega.</p><p>La tarifa y el pago se acuerdan entre las partes. No hay garantía de seguimiento GPS en segundo plano.</p></div>
    </section>
    <section className="landing-section documentation" id="documentacion" aria-labelledby="guide-title">
      <p className="eyebrow">CENTRO DE AYUDA</p><h2 id="guide-title">Respuestas claras antes de empezar.</h2>
      <p className="documentation-intro">Consulta la guía de tu rol y los detalles sobre trabajos, ubicación y privacidad. Dentro de CHITA también encontrarás una guía contextual en el panel.</p>
      <nav className="documentation-nav" aria-label="Temas de la guía">
        <a href="#ayuda-cuenta">Cuenta y direcciones</a><a href="#ayuda-trabajos">Trabajos y propuestas</a><a href="#ayuda-repartidores">Red y repartidores</a><a href="#ayuda-ubicacion">Ubicación y HALCON</a>
      </nav>
      <div className="documentation-groups">
        <section className="documentation-group" id="ayuda-cuenta" aria-labelledby="ayuda-cuenta-titulo"><h3 id="ayuda-cuenta-titulo">Cuenta y direcciones</h3><div className="landing-guide">
          <details><summary>¿Cómo elijo una dirección?</summary><p>Pulsa el punto exacto en el mapa o usa tu ubicación. CHITA rellena latitud, longitud y una dirección aproximada. Revisa el número, la entrada y las referencias antes de guardar. Puedes corregir la dirección o escribirla si no se encuentra.</p></details>
          <details><summary>¿Cómo cambio mi foto?</summary><p>En Mi cuenta, pulsa Cambiar foto y selecciona una imagen JPG, PNG o WebP de hasta 5 MB. Se recorta al centro y se guarda automáticamente. Puedes quitarla desde la misma sección. El nombre y la foto de los repartidores aparecen a las empresas en el directorio.</p></details>
          <details><summary>¿Qué necesito en el móvil?</summary><p>Un navegador con conexión y permiso de ubicación. El botón Menú abre las secciones. Para compartir GPS, mantén la app abierta: el navegador puede pausar los envíos al bloquear la pantalla. CHITA es una web adaptable; no procesa pagos.</p></details>
        </div></section>
        <section className="documentation-group" id="ayuda-trabajos" aria-labelledby="ayuda-trabajos-titulo"><h3 id="ayuda-trabajos-titulo">Trabajos y propuestas</h3><div className="landing-guide">
          <details><summary>¿Público o exclusivo de una red?</summary><p>La empresa elige al publicar. Un trabajo público sin una propuesta pendiente puede verlo y aceptarlo cualquier repartidor registrado. Los exclusivos aparecen a quienes pertenecen a la red. Una empresa puede enviar una propuesta a un repartidor disponible fuera de su red: le da acceso sólo a ese trabajo, sin añadirlo automáticamente a la red.</p></details>
          <details><summary>¿Cómo funcionan las propuestas?</summary><p>Selecciona un trabajo disponible y un repartidor cercano. El trabajo queda reservado hasta dos minutos, o hasta vencer la recogida si ocurre antes. El repartidor acepta o rechaza desde «Propuestas». Si rechaza, la empresa retira la propuesta o vence el plazo, el trabajo se libera. Una propuesta no garantiza aceptación.</p></details>
          <details><summary>¿Aceptar crea un compromiso?</summary><p>El trabajo se asigna a un solo repartidor. Revisa tarifa, direcciones y horarios antes de aceptar. Tras recoger y llegar al destino, informa la entrega. La empresa debe comprobarla y confirmarla; también puede solicitar una revisión.</p></details>
          <details><summary>¿Cómo se ordenan los trabajos?</summary><p>Primero aparecen tus entregas activas. Después, las recogidas disponibles más cercanas a la ubicación de tu perfil. Usa «Ordenar cerca de mí» para actualizar el orden con tu GPS. Las distancias son en línea recta, no tiempos de viaje; los horarios siguen siendo responsabilidad de las partes.</p></details>
        </div></section>
        <section className="documentation-group" id="ayuda-repartidores" aria-labelledby="ayuda-repartidores-titulo"><h3 id="ayuda-repartidores-titulo">Red y repartidores</h3><div className="landing-guide">
          <details><summary>¿Cómo invito repartidores a mi red?</summary><p>Como empresa, entra en Directorio, busca un nombre y consulta su puntuación. Pulsa Invitar a mi red. La solicitud aparece en Invitaciones del repartidor y debe aceptarla para afiliarse. En Mi red puedes revisar el estado o invitar por correo. El directorio no indica disponibilidad ni muestra ubicaciones.</p></details>
          <details><summary>¿Cómo encuentro repartidores cercanos?</summary><p>Como empresa, abre «Buscar repartidores». La búsqueda parte de tu dirección o de la recogida del trabajo seleccionado. Sólo aparecen repartidores que activaron su disponibilidad, con posición de los últimos cinco minutos y sin entregas pendientes. La distancia es en línea recta.</p></details>
          <details><summary>¿Cuándo puedo calificar a un repartidor?</summary><p>Tu empresa debe haber confirmado al menos tres entregas completadas por ese repartidor. Los trabajos sólo aceptados, pendientes o cancelados no cuentan. Cada empresa aporta una calificación de 1 a 5 al promedio y puede actualizarla sin duplicarla.</p></details>
        </div></section>
        <section className="documentation-group" id="ayuda-ubicacion" aria-labelledby="ayuda-ubicacion-titulo"><h3 id="ayuda-ubicacion-titulo">Ubicación y HALCON</h3><div className="landing-guide">
          <details><summary>¿Cómo vinculo HALCON?</summary><p>Entra como repartidor y pulsa «Vincular con HALCON» o abre HALCON en el menú. Usa una cuenta personal de HALCON. La sesión cifrada dura como máximo 25 minutos y puede renovarse. Vincular la cuenta no activa el GPS por sí solo.</p></details>
          <details><summary>¿Quién puede ver mi ubicación?</summary><p>En CHITA, la empresa del trabajo activo puede consultar la última posición del repartidor con HALCON vinculado. Debes activar el GPS dentro del trabajo. Los destinatarios y moderadores configurados en HALCON conservan sus permisos. Puedes detener los nuevos envíos desde el trabajo. Si activas «Mostrarme disponible», también compartes una posición GPS reciente con empresas que busquen repartidores cercanos; desactívala cuando quieras dejar de aparecer.</p></details>
        </div></section>
      </div>
    </section>
    <section className="landing-final"><div><p>EL SIGUIENTE PASO ES TUYO</p><h2>Empieza por tu primera conexión.</h2></div><a className="cta-yellow" href="/registro">Empezar con CHITA <span aria-hidden="true">↗</span></a></section>
  </div>;
}
