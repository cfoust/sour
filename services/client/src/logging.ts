export enum LogLevel {
  Info = 1 << 0,
  Warn = 1 << 1,
  Err = 1 << 2,
  Debug = 1 << 3,
  Init = 1 << 4,
  Echo = 1 << 5,
}

export enum Color {
  Green = '\f0', // player talk
  Blue = '\f1', // "echo" command
  Yellow = '\f2', // gameplay messages
  Red = '\f3', // important errors
  Gray = '\f4',
  Magenta = '\f5',
  Orange = '\f6',
  White = '\f7',

  Save = '\fs',
  Restore = '\fr',
}

const wrap = (s: string, color: Color): string =>
  `${Color.Save}${color}${s}${Color.Restore}`

export const colors = {
  green: (s: string): string => wrap(s, Color.Green),
  blue: (s: string): string => wrap(s, Color.Blue),
  yellow: (s: string): string => wrap(s, Color.Yellow),
  red: (s: string): string => wrap(s, Color.Red),
  gray: (s: string): string => wrap(s, Color.Gray),
  magenta: (s: string): string => wrap(s, Color.Magenta),
  orange: (s: string): string => wrap(s, Color.Orange),
  white: (s: string): string => wrap(s, Color.White),
  success: (s: string): string => colors.green(s),
  fail: (s: string): string => colors.orange(s),
  error: (s: string): string => colors.red(s),
}

export const sour = (message: string) => `${colors.yellow('sour')} ${message}`
export const info = (message: string) =>
  BananaBread.conoutf(LogLevel.Info, sour(message))
export const success = (message: string) =>
  BananaBread.conoutf(LogLevel.Info, sour(colors.success(message)))
export const warn = (message: string) =>
  BananaBread.conoutf(LogLevel.Warn, sour(colors.fail(message)))
export const error = (message: string) =>
  BananaBread.conoutf(LogLevel.Err, sour(colors.error(message)))
