import { useEffect, useState } from "react";

type Theme = "system" | "light" | "dark";
const key = "chita-theme";
const media = window.matchMedia("(prefers-color-scheme: dark)");
const valid = (value: string | null): Theme =>
  value === "light" || value === "dark" ? value : "system";
function saved(): Theme {
  try { return valid(localStorage.getItem(key)); } catch { return "system"; }
}
function apply(theme: Theme) {
  const dark = theme === "dark" || (theme === "system" && media.matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
  document.querySelector('meta[name="theme-color"]')?.setAttribute("content", dark ? "#091711" : "#f2f7f3");
}
apply(saved());

export function ThemePicker() {
  const [theme, setTheme] = useState<Theme>(saved);
  useEffect(() => {
    apply(theme);
    const change = () => apply(theme);
    const sync = (event: StorageEvent) => {
      if (event.key === key || event.key === null) setTheme(saved());
    };
    media.addEventListener("change", change);
    window.addEventListener("storage", sync);
    return () => {
      media.removeEventListener("change", change);
      window.removeEventListener("storage", sync);
    };
  }, [theme]);
  return <label className="theme-picker">Apariencia
    <select value={theme} onChange={event => {
      const next = valid(event.target.value);
      setTheme(next);
      try { localStorage.setItem(key, next); } catch { /* Usar la preferencia durante esta visita. */ }
    }}>
      <option value="system">Sistema</option>
      <option value="light">Claro</option>
      <option value="dark">Oscuro</option>
    </select>
  </label>;
}
