// Jellyfin images through the backend proxy (public, whitelisted image paths).

/** Primary image (poster / cover) of a Jellyfin item. */
export function itemImage(id: string, width = 240): string {
  return `/proxy/Items/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=90`
}

/** Profile picture of a Jellyfin user. */
export function userImage(id: string, width = 64): string {
  return `/proxy/Users/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=90`
}

/** Jellyfin ids are GUIDs (32 hex digits, dashes optional). */
export const isJellyfinId = (id: string) => /^[0-9a-f]{32}$/i.test(id.replace(/-/g, ''))
