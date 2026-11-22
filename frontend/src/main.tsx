import React from 'react'
import ReactDOM from 'react-dom/client'
import '@/styles/tokens.css'
import '@/styles/base.css'
import '@/lib/i18n'
import App from '@/app/App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
