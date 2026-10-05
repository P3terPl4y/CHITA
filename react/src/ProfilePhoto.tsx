import { useState } from "react";
import { api, type User } from "./api";

export function Avatar({ user, className = "user-avatar" }: { user: Pick<User, "name" | "avatar_url">; className?: string }) {
  return user.avatar_url ? <img className={className} src={user.avatar_url} alt="" /> : <span className={className} aria-hidden="true">{user.name.trim().charAt(0).toUpperCase()}</span>;
}
export function ProfilePhoto({ user, updated, sessionCurrent }: { user: User; updated: (url: string) => void; sessionCurrent: () => boolean }) {
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState("");
  async function save(avatar: string) {
    if (!sessionCurrent()) throw new Error("La sesión cambió. Vuelve a entrar para cambiar tu foto.");
    const response = await api<{ avatar_url: string }>("/profile/avatar", "POST", { avatar });
    if (!sessionCurrent()) return;
    updated(response.avatar_url);
    setNotice(avatar ? "Foto de perfil guardada" : "Foto de perfil eliminada");
  }
  async function choose(file?: File) {
    if (!file || busy) return;
    setBusy(true); setError(""); setNotice("");
    let bitmap: ImageBitmap | undefined;
    try {
      if (!/^image\/(jpeg|png|webp)$/.test(file.type) || file.size > 5 * 1024 * 1024) throw new Error("Elige una foto JPG, PNG o WebP de hasta 5 MB.");
      bitmap = await createImageBitmap(file);
      if (bitmap.width * bitmap.height > 20000000) throw new Error("La foto tiene demasiada resolución. Elige una más pequeña.");
      const canvas = document.createElement("canvas"); canvas.width = canvas.height = 128;
      const context = canvas.getContext("2d");
      if (!context) throw new Error("No se pudo preparar la foto en este navegador.");
      const side = Math.min(bitmap.width, bitmap.height);
      context.fillStyle = "#ffffff"; context.fillRect(0, 0, 128, 128);
      context.drawImage(bitmap, (bitmap.width-side)/2, (bitmap.height-side)/2, side, side, 0, 0, 128, 128);
      await save(canvas.toDataURL("image/jpeg", .8));
    } catch (e) { setError(e instanceof Error ? e.message : "No se pudo guardar la foto. Inténtalo otra vez."); }
    finally { bitmap?.close(); setBusy(false); }
  }
  return <section className="profile-photo" aria-label="Foto de perfil" aria-busy={busy}>
    <Avatar user={user} className="account-avatar" />
    <h3>Tu foto de perfil</h3><p className="muted">Elige una foto: se recorta al centro y se guarda automáticamente. {user.role === "courier" ? "Tu nombre y foto aparecen a las empresas en el directorio de repartidores." : "Se usa para identificar tu cuenta."}</p>
    <label className="file-control">{busy ? "Guardando foto…" : "Cambiar foto"}<input type="file" accept="image/jpeg,image/png,image/webp" disabled={busy} onChange={e => { const file = e.target.files?.[0]; e.target.value = ""; void choose(file); }} /></label>
    {user.avatar_url && <button disabled={busy} onClick={async () => { setBusy(true); setError(""); setNotice(""); try { await save(""); } catch (e) { setError(e instanceof Error ? e.message : "No se pudo quitar la foto"); } finally { setBusy(false); } }}>Quitar foto</button>}
    {error && <p role="alert" className="alert">{error}</p>}{notice && <p role="status" className="success">{notice}</p>}
  </section>;
}
