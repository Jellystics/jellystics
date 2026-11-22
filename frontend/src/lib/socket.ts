import { useEffect, useRef } from 'react'
import { io } from 'socket.io-client'

const socket = io('/', {
  withCredentials: true,
  autoConnect: false,
  auth: (cb) => {
    cb({ token: localStorage.getItem('jellystics-token') })
  },
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
