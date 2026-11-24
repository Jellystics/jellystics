// Language and theme of the « Mes statistiques » page, taken from jellyfin-web's own settings
// (localStorage "<userId>-language" / "<userId>-appTheme", see jellyfin-web userSettings.js).

export const LANGUAGES = ['en', 'fr', 'es'] as const
export type Language = (typeof LANGUAGES)[number]

const supported = (tag: string | null | undefined): Language | null => {
  const base = tag?.trim().toLowerCase().split(/[-_]/)[0]
  return base && (LANGUAGES as readonly string[]).includes(base) ? (base as Language) : null
}

/** First supported language among the Jellyfin user setting, the document language and the browser languages; en otherwise. */
export function pickLanguage(candidates: (string | null | undefined)[]): Language {
  for (const c of candidates) {
    const l = supported(c)
    if (l) return l
  }
  return 'en'
}

/** jellyfin-web themes with a light background. */
const LIGHT_THEMES = new Set(['light', 'appletv', 'catppuccin-latte'])

/**
 * Page theme: the Jellyfin theme of the user when known, else light only when the browser asks for it,
 * dark otherwise (Jellyfin's default).
 */
export function resolveTheme(jellyfinTheme: string | null | undefined, prefersLight: boolean): 'light' | 'dark' {
  if (jellyfinTheme) return LIGHT_THEMES.has(jellyfinTheme.toLowerCase()) ? 'light' : 'dark'
  return prefersLight ? 'light' : 'dark'
}

/** Per-user jellyfin-web setting (null when absent or storage is blocked). */
export function jellyfinSetting(storage: Pick<Storage, 'getItem'> | null, userId: string | null, name: string): string | null {
  if (!storage || !userId) return null
  try { return storage.getItem(`${userId}-${name}`) } catch { return null }
}
