const DEVICE_KEY = 'jellystics-device-id'

/**
 * Stable per-browser device id. Jellyfin revokes the previous access token of
 * the same user + device on login, so each browser needs its own id.
 */
export function getDeviceId(): string {
  try {
    let id = localStorage.getItem(DEVICE_KEY)
    if (!id) {
      id = crypto.randomUUID().replace(/-/g, '')
      localStorage.setItem(DEVICE_KEY, id)
    }
    return id
  } catch {
    return crypto.randomUUID().replace(/-/g, '')
  }
}

export function getDeviceName(): string {
  const ua = navigator.userAgent
  const browser = /Firefox\//.test(ua) ? 'Firefox' : /Edg\//.test(ua) ? 'Edge' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'Browser'
  return `Jellystics (${browser})`
}
