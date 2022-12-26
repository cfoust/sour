export enum MessageType {
  Info,
  ServerConnected,
  ServerDisconnected,
  ServerResponse,
  AuthSucceeded,
  AuthFailed,
  Chat,
  Connect,
  Disconnect,
  Command,
  DiscordCode,
  Packet,
}

export enum ENetEventType {
  None,
  Connect,
  Disconnect,
  Receive,
}

export type ServerInfo = {
  Host: string
  Port: number
  Info: Uint8Array
  Length: number
}

export type InfoMessage = {
  Op: MessageType.Info
  Master: ServerInfo[]
  Cluster: string[]
}

export type PacketMessage = {
  Op: MessageType.Packet
  Data: Uint8Array
  Length: number
  Channel: number
}

export type ServerConnectedMessage = {
  Op: MessageType.ServerConnected
  Server: string
  Internal: boolean
  Owned: boolean
}

export type ServerDisconnectedMessage = {
  Op: MessageType.ServerDisconnected
  Message: string
  Reason: number
}

export type ChatMessage = {
  Op: MessageType.Chat
  Message: string
}

export type ConnectMessage = {
  Op: MessageType.Connect
  Target: string
}

export type DisconnectMessage = {
  Op: MessageType.Disconnect
  Target: string
}

export type AuthSucceededMessage = {
  Op: MessageType.AuthSucceeded
  Code: string
  User: {
    Avatar: string
    Id: string
    Discriminator: string
    Username: string
  }
  PrivateKey: string
}

export type AuthFailedMessage = {
  Op: MessageType.AuthFailed
  Code: string
}

export type CommandMessage = {
  Op: MessageType.Command
  Command: string
  Id: number
}

export type DiscordCodeMessage = {
  Op: MessageType.DiscordCode
  Code: string
}

export type ResponseMessage = {
  Op: MessageType.ServerResponse
  Response: string
  Success: boolean
  Id: number
}

export type ClientAuthMessage = DiscordCodeMessage

export type ServerAuthMessage = AuthSucceededMessage | AuthFailedMessage

export type ServerMessage =
  | InfoMessage
  | PacketMessage
  | ServerConnectedMessage
  | ServerDisconnectedMessage
  | ResponseMessage
  | AuthSucceededMessage
  | AuthFailedMessage
  | ChatMessage

export type SocketMessage =
  | PacketMessage
  | ServerConnectedMessage
  | ServerDisconnectedMessage

export type ClientMessage =
  | PacketMessage
  | ConnectMessage
  | DisconnectMessage
  | CommandMessage
  | DiscordCodeMessage
