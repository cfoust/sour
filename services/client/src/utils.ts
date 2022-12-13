
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
