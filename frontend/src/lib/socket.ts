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

/** Subscribe to a socket event for the lifetime of the component. */
export function useSocketEvent(event: string, handler: (data: unknown) => void) {
  const ref = useRef(handler)
  useEffect(() => { ref.current = handler })
  useEffect(() => {
    const fn = (data: unknown) => ref.current(data)
    socket.on(event, fn)
    return () => { socket.off(event, fn) }
  }, [event])
}
