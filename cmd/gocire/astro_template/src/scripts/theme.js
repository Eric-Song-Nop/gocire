const storageKey = "gocire-theme";
const validThemes = new Set(["light", "dark"]);
const root = document.documentElement;
const buttons = Array.from(document.querySelectorAll("[data-theme-toggle]"));

const readStoredTheme = () => {
  try {
    return localStorage.getItem(storageKey);
  } catch {
    return null;
  }
};

const writeStoredTheme = (theme) => {
  try {
    localStorage.setItem(storageKey, theme);
  } catch {
    // Theme persistence is optional; the current page still updates.
  }
};

const systemTheme = () => {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const currentTheme = () => {
  const theme = root.getAttribute("data-theme");
  if (validThemes.has(theme)) {
    return theme;
  }
  const storedTheme = readStoredTheme();
  return validThemes.has(storedTheme) ? storedTheme : systemTheme();
};

const updateToggleLabels = (theme) => {
  const nextTheme = theme === "dark" ? "light" : "dark";
  const label = nextTheme === "dark" ? "Switch to dark theme" : "Switch to light theme";
  for (const button of buttons) {
    button.setAttribute("aria-label", label);
    button.setAttribute("title", label);
  }
};

const applyTheme = (theme, options = {}) => {
  const nextTheme = validThemes.has(theme) ? theme : systemTheme();
  root.setAttribute("data-theme", nextTheme);
  updateToggleLabels(nextTheme);

  if (options.persist) {
    writeStoredTheme(nextTheme);
  }
};

applyTheme(currentTheme());

for (const button of buttons) {
  button.addEventListener("click", () => {
    applyTheme(currentTheme() === "dark" ? "light" : "dark", { persist: true });
  });
}
