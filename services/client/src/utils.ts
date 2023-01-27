import { detect } from 'detect-browser'

export type PromiseSet<T> = {
  promise: Promise<T>
  resolve: (value: T) => void
  reject: (reason?: Error) => void
}

// Break up a promise into its resolve and reject functions for ease of use.
export function breakPromise<T>(): PromiseSet<T> {
  let resolve: (value: T) => void = () => {}
  let reject: (reason?: Error) => void = () => {}
  const promise = new Promise<T>((_resolve, _reject) => {
    resolve = _resolve
    reject = _reject
  })

  return {
    promise,
    resolve,
    reject,
  }
}

export function getBrowser(): {
  isFirefox: boolean
  isSafari: boolean
  isMobile: boolean
} {
  const result = detect()

  return {
    isFirefox: result?.name === 'firefox',
    isSafari: result?.name === 'safari',
    isMobile: result?.os === 'iOS' || result?.os === 'Android OS',
  }
}

export const BROWSER = getBrowser()
