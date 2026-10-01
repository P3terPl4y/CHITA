import { useEffect, useRef, useState } from "react";

export function Navigation({ role, tab, unread, select }: { role: string; tab: string; unread: boolean; select: (tab: string) => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [open, setOpen] = useState(false);
  const choices = role === "admin" ? [["overview","Resumen"],["jobs","Trabajos"],["companies","Empresas"],["couriers","Repartidores"],["audits","Historial"],["security","Seguridad"]] : [["jobs", "Trabajos"], ["network", role === "company" ? "Mi red" : "Invitaciones"], ["notifications", "Avisos"], ["profile", "Mi cuenta"], ...(role === "courier" ? [["halcon", "HALCON"]] : [])];
  function close() { dialog.current?.close(); setOpen(false); }
  useEffect(() => {
    if (!open) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const resize = () => { if (window.matchMedia("(min-width: 48.01rem)").matches) close(); };
    window.addEventListener("resize", resize);
    return () => { document.body.style.overflow = previous; window.removeEventListener("resize", resize); };
  }, [open]);
  function items() { return choices.map(([key, label]) => <button key={key} type="button" aria-current={tab === key ? "page" : undefined} className={tab === key ? "chosen" : ""} onClick={() => { select(key); close(); }}>
    {label}{key === "notifications" && unread && <span className="dot" aria-label="Sin leer" />}
  </button>); }
  return <>
    <nav className="desktop-nav" aria-label="Secciones">{items()}</nav>
    <button className="menu-toggle" type="button" aria-haspopup="dialog" aria-expanded={open} aria-controls="mobile-menu" onClick={() => { dialog.current?.showModal(); setOpen(true); }}><span aria-hidden="true">☰</span> <span>Menú</span></button>
    <dialog ref={dialog} id="mobile-menu" className="mobile-drawer" aria-labelledby="menu-title" onClose={() => setOpen(false)} onCancel={() => setOpen(false)} onClick={event => { if (event.target === dialog.current) close(); }}>
      <div className="drawer-content"><div className="drawer-heading"><h2 id="menu-title">Tu plataforma</h2><button type="button" aria-label="Cerrar menú" onClick={close} autoFocus>×</button></div><nav aria-label="Secciones móviles">{items()}</nav><p className="muted">{role === "admin" ? "Gestiona cuentas y trabajos con trazabilidad." : role === "company" ? "Publica y coordina tus entregas." : "Encuentra trabajos y organiza tu recorrido."}</p></div>
    </dialog>
  </>;
}
