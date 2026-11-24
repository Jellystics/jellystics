import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import HttpBackend from 'i18next-http-backend'
import { LANGUAGES, type Language } from './prefs'

/** i18next of the me page: same translation files as the dashboard, loaded relative to the page (./locales/…). */
export function initMeI18n(lng: Language) {
  document.documentElement.lang = lng
  return i18n
    .use(HttpBackend)
    .use(initReactI18next)
    .init({
      lng,
      fallbackLng: 'en',
      supportedLngs: [...LANGUAGES],
      backend: { loadPath: 'locales/{{lng}}/translation.json' },
      interpolation: { escapeValue: false },
    })
}
