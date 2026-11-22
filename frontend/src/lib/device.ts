const DEVICE_KEY = 'jellystics-device-id'

/**
 * 32 random hex chars. Uses crypto.getRandomValues, which (unlike
 * crypto.randomUUID) is also available on plain-HTTP origins such as
 * http://192.168.1.10:3000 — the usual self-hosted setup.
 */
function randomHex(): string {
  const bytes = new Uint8Array(16)
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256)
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

/**
 * Stable per-browser device id. Jellyfin revokes the previous access token of
 * the same user + device on login, so each browser needs its own id.
 */
export function getDeviceId(): string {
  try {
    let id = localStorage.getItem(DEVICE_KEY)
    if (!id) {
      id = randomHex()
      localStorage.setItem(DEVICE_KEY, id)
    }
    return id
  } catch {
    return randomHex()
  }
}

export function getDeviceName(): string {
  const ua = navigator.userAgent
  const browser = /Firefox\//.test(ua) ? 'Firefox' : /Edg\//.test(ua) ? 'Edge' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'Browser'
  return `Jellystics (${browser})`
}
