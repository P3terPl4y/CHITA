import { useId, useState } from "react";

export function PasswordField({ register }: { register: boolean }) {
  const [visible, setVisible] = useState(false);
  const id = useId();
  return <div className="password-field">
    <label htmlFor={id}>Contraseña</label>
    <div className="password-input">
      <input id={id} name="password" type={visible ? "text" : "password"}
        required minLength={register ? 8 : 1} maxLength={72}
        autoComplete={register ? "new-password" : "current-password"}
        aria-describedby={register ? `${id}-help` : undefined} />
      <button type="button" aria-controls={id} aria-pressed={visible}
        aria-label={visible ? "Ocultar contraseña" : "Mostrar contraseña"}
        onClick={() => setVisible(value => !value)}>{visible ? "Ocultar" : "Mostrar"}</button>
    </div>
    {register && <p className="muted" id={`${id}-help`}>Usa al menos 8 caracteres. Evita reutilizar otra contraseña.</p>}
  </div>;
}
