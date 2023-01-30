import * as nipplejs from 'nipplejs'

declare module 'nipplejs' {
  export class FixedJoystickManager {
    create(options?: JoystickManagerOptions): JoystickManager

    on(
      type: JoystickManagerEventTypes | JoystickManagerEventTypes[],
      handler: (evt: EventData, data: Joystick) => void
    ): void
    off(
      type: JoystickManagerEventTypes | JoystickManagerEventTypes[],
      handler: (evt: EventData, data: Joystick) => void
    ): void
    get(identifier: number): Joystick
    destroy(): void
  }

  export function create(options: JoystickManagerOptions): FixedJoystickManager;
}
