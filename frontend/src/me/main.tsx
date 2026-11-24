import React, { Suspense } from 'react'
import ReactDOM from 'react-dom/client'
import '@/styles/tokens.css'
import '@/styles/base.css'
import { initMeI18n } from './i18n'
import { MeApp } from './MeApp'
import { jellyfinSetting, pickLanguage, resolveTheme } from './prefs'
import { basePathFrom, readSession } from './session'

start()

function storage(): Storage | null {
  try { return window.localStorage } catch { return null }
}

function start() {
  const store = storage()
  const session = readSession(store, window.location)
  const userId = session?.userId ?? null
  const basePath = basePathFrom(window.location.pathname)

  const applyTheme = () => { document.documentElement.dataset.theme = resolveTheme(jellyfinSetting(store, userId, 'appTheme')) }
  applyTheme()

  initMeI18n(pickLanguage([jellyfinSetting(store, userId, 'language'), ...(navigator.languages ?? []), navigator.language]))

  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Suspense fallback={null}>
        <MeApp session={session} basePath={basePath} />
      </Suspense>
    </React.StrictMode>,
  )
}
