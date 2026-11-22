import { useEffect, useRef } from 'react'
import { io } from 'socket.io-client'
import { getToken } from './auth'

const socket = io('/', {
  withCredentials: true,
  autoConnect: false,
  auth: (cb) => {
    cb({ token: getToken() })
  },
})

socket.on('connect_error', (err) => {
  // Auth failure: stop the auto-reconnect loop (the next API 401 redirects to /login).
  if (/unauthori[sz]ed|invalid token|jwt|forbidden|auth/i.test(err.message)) socket.disconnect()
})

export default socket

/** Subscribe to one socket event (or several, sharing a handler) for the lifetime of the component. */
export function useSocketEvent(event: string | readonly string[], handler: (data: unknown) => void) {
  const ref = useRef(handler)
  useEffect(() => { ref.current = handler })
  const key = typeof event === 'string' ? event : event.join('\u0000')
  useEffect(() => {
    const fn = (data: unknown) => ref.current(data)
    const names = key.split('\u0000')
    names.forEach((e) => socket.on(e, fn))
    return () => { names.forEach((e) => socket.off(e, fn)) }
  }, [key])
}
