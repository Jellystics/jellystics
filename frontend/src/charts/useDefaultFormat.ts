import { useTranslation } from 'react-i18next'
import { defaultFormat } from './scale'

/** Default axis/value formatter bound to the current app locale. */
export function useDefaultFormat(): (v: number) => string {
  const { i18n } = useTranslation()
  const locale = i18n.resolvedLanguage || 'fr'
  return (v) => defaultFormat(v, locale)
}
