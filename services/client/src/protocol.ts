export enum MessageType {
  Info,
  ServerConnected,
  ServerDisconnected,
  ServerResponse,
  Connect,
  Disconnect,
  Command,
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

export type ConnectMessage = {
  Op: MessageType.Connect
  Target: string
}

export type DisconnectMessage = {
  Op: MessageType.Disconnect
  Target: string
}

export type CommandMessage = {
  Op: MessageType.Command
  Command: string
  Id: number
}

export type ResponseMessage = {
  Op: MessageType.ServerResponse
  Response: string
  Success: boolean
  Id: number
}

export type ServerMessage =
  | InfoMessage
  | PacketMessage
  | ServerConnectedMessage
  | ServerDisconnectedMessage
  | ResponseMessage

export type SocketMessage =
  | PacketMessage
  | ServerConnectedMessage
  | ServerDisconnectedMessage

export type ClientMessage =
  | PacketMessage
  | ConnectMessage
  | DisconnectMessage
  | CommandMessage
