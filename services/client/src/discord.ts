import * as React from 'react'

import { CONFIG } from './config'
import { mountFile } from './assets/hook'
import * as log from './logging'

import { MessageType } from './protocol'
import type { ClientAuthMessage, ServerAuthMessage } from './protocol'

enum AuthStatus {
  Unauthenticated,
  Authenticated,
  Failed,
  AvatarMounted,
}

export type DiscordUser = {
  Id: string
  Username: string
  Discriminator: string
  Avatar: string
}

export type UnauthenticatedState = {
  status: AuthStatus.Unauthenticated
}

export type AuthenticatedState = {
  status: AuthStatus.Authenticated
  user: DiscordUser
}

export type FailedState = {
  status: AuthStatus.Failed
}

export type AvatarMountedState = {
  status: AuthStatus.AvatarMounted
  user: DiscordUser
  avatarPath: string
}

export type AuthState =
  | UnauthenticatedState
  | AuthenticatedState
  | FailedState
  | AvatarMountedState

const getAvatarPath = (id: string) => `/packages/textures/avatars/${id}.png`
async function mountImage(filename: string, url: string): Promise<void> {
  const response = await fetch(url)
  const buffer = await response.arrayBuffer()
  Module.FS_createPath('/packages/textures/', 'avatars', true, true)
  await mountFile(filename, new Uint8Array(buffer))
}

export const DISCORD_CODE = 'discord'

export default function useAuth(
  sendMessage: (message: ClientAuthMessage) => void
): {
  menu: string
  initialize: (code: Maybe<string>) => void
  receiveMessage: (message: ServerAuthMessage) => void
} {
  const [state, setState] = React.useState<AuthState>({
    status: AuthStatus.Unauthenticated,
  })

  const initialize = React.useCallback(
    (urlCode: Maybe<string>) => {
      if (!CONFIG.auth.enabled) return

      let code: Maybe<string> = urlCode
      // Look in localStorage
      if (code == null) {
        code = localStorage.getItem(DISCORD_CODE)
      }

      if (code == null) {
        return
      }

      sendMessage({
        Op: MessageType.DiscordCode,
        Code: code,
      })
    },
    [sendMessage]
  )

  React.useEffect(() => {
    Module.discord = {
      login: () => {
        const { enabled, authorizationURL, redirectURI } = CONFIG.auth
        if (!enabled) return
        window.location.assign(
          authorizationURL.replace(
            '{{redirectURI}}',
            encodeURIComponent(redirectURI)
          )
        )
      },
      copyKey: () => {
        console.log('copyKey');
      },
      regenKey: () => {},
      logout: () => {
        console.log('logging out');
        localStorage.removeItem(DISCORD_CODE)
        setState({
          status: AuthStatus.Unauthenticated,
        })
      },
    }
  }, [])

  const receiveMessage = React.useCallback((message: ServerAuthMessage) => {
    if (message.Op === MessageType.AuthSucceeded) {
      localStorage.setItem(DISCORD_CODE, message.Code)
      const { User: user } = message
      setState({
        status: AuthStatus.Authenticated,
        user: user,
      })

      const { Id, Avatar } = user
      ;(async () => {
        const path = getAvatarPath(Id)
        const filename = await mountImage(
          path,
          `https://cdn.discordapp.com/avatars/${Id}/${Avatar}.png?size=32`
        )

        setState({
          status: AuthStatus.AvatarMounted,
          user: user,
          avatarPath: path,
        })
      })()
      return
    }

    if (message.Op === MessageType.AuthFailed) {
      setState({
        status: AuthStatus.Failed,
      })

      setTimeout(() => {
        setState({
          status: AuthStatus.Unauthenticated,
        })
      }, 4000)
      return
    }
  }, [])

  const menu = React.useMemo<string>(() => {
    if (state.status === AuthStatus.Unauthenticated) {
      return `
          guibutton "log in.." [js "Module.discord.login()"]
      `
    }

    if (state.status === AuthStatus.Authenticated) {
      return `
          guitext "logging in.." 0
      `
    }

    if (state.status === AuthStatus.Failed) {
      return `
          guitext "${log.error('failed to login')}" 0
      `
    }

    if (state.status === AuthStatus.AvatarMounted) {
      const {
        avatarPath,
        user: { Username, Discriminator },
      } = state

      return `
        guilist [
          guiimage "${avatarPath}" [] 0.5
          guitext "${log.colors.blue(`${Username}#${Discriminator}`)}" 0
        ]
        guibutton "copy addauthkey.." [js "Module.discord.copyKey()"]
        guibutton "regenerate auth key.." [js "Module.discord.regenKey()"]
        guibutton "log out.." [js "Module.discord.logout()"]
      `
    }

    return ''
  }, [state])

  return {
    menu,
    initialize,
    receiveMessage,
  }
}
