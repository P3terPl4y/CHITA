import type { Job, User } from "./api";
import { labels } from "./api";

export function DashboardHome({ user, jobs, total, page, navigate, publish, openJob }: {
  user: User; jobs: Job[]; total: number; page: number;
  navigate: (section: string) => void; publish: () => void; openJob: (job: Job) => void;
}) {
  const company = user.role === "company";
  const attention = jobs.filter(j => company ? j.status === "delivery_reported" : j.assigned_courier_id === user.id && ["accepted", "picked_up", "arrived"].includes(j.status));
  return <div className="dashboard-home">
    <section className="panel start-panel">
      <p className="eyebrow">UN PASO A LA VEZ</p>
      <h2>{company ? "¿Qué necesitas hacer hoy?" : "Tu próximo paso, a mano"}</h2>
      <p>{company ? "Publica una entrega, ofrece el trabajo a un repartidor y confirma cuando llegue." : "Encuentra un trabajo, confirma la recogida y avisa cuando hayas entregado."}</p>
      <button className="quiet guide-link" onClick={() => navigate("guide")}>¿Primera vez? Ver la guía de uso →</button>
      <div className="task-grid">
        <button onClick={company ? publish : () => navigate("jobs")}><span className="task-number">01</span><strong>{company ? "Crear una entrega" : "Encontrar trabajo"}</strong><span>{company ? "Define los puntos, horarios y tarifa." : "Consulta trabajos públicos y de tus redes."}</span><span aria-hidden="true">→</span></button>
        <button onClick={() => navigate(company ? "nearby" : "offers")}><span className="task-number">02</span><strong>{company ? "Encontrar repartidor" : "Revisar propuestas"}</strong><span>{company ? "Elige un trabajo y consulta quién está cerca." : "Acepta o rechaza antes de que venza el plazo."}</span><span aria-hidden="true">→</span></button>
        <button onClick={() => navigate("jobs")}><span className="task-number">03</span><strong>{company ? "Seguir mis entregas" : "Continuar mi entrega"}</strong><span>{company ? "Revisa el estado y confirma las entregas recibidas." : "Abre tu trabajo para registrar el siguiente paso."}</span><span aria-hidden="true">→</span></button>
      </div>
    </section>
    <section className="panel"><div className="section-heading"><div><p className="eyebrow">SIGUIENTE ACCIÓN</p><h2>{company ? "Entregas por confirmar" : "Tus trabajos en curso"}</h2></div><button onClick={() => navigate("jobs")}>Ver trabajos</button></div>
      <p className="muted">Resumen de la página {page} · {jobs.length} de {total} trabajos. Revisa las demás páginas para ver el resto.</p>
      {attention.length ? <div className="attention-list">{attention.map(job => <button key={job.id} onClick={() => openJob(job)}><span><strong>{job.title}</strong><span className="muted">{job.dropoff_address}</span></span><span className="badge">{labels[job.status]}</span><span aria-hidden="true">→</span></button>)}</div> : <div className="empty">{company ? "No hay entregas pendientes de confirmación en esta página." : "No tienes trabajos en curso en esta página."}</div>}
    </section>
    <aside className="dashboard-tip"><strong>{company ? "¿Trabajas con los mismos repartidores?" : "Tú decides cuándo compartir tu ubicación"}</strong><p>{company ? "Invítalos a tu red. Los trabajos exclusivos sólo se muestran a quienes hayan aceptado la invitación." : "HALCON permite compartir tu ubicación durante una entrega activa. Vincular la cuenta no activa el GPS."}</p><button onClick={() => navigate(company ? "network" : "halcon")}>{company ? "Gestionar mi red" : "Configurar HALCON"}</button></aside>
  </div>;
}
