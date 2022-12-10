export enum MessageType {
  Info,
  ServerConnected,
  ServerDisconnected,
  Connect,
  Disconnect,
  Packet,
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
}

export type ServerDisconnectedMessage = {
  Op: MessageType.ServerDisconnected
}

export type ConnectMessage = {
  Op: MessageType.Connect
  Target: string
}

export type DisconnectMessage = {
  Op: MessageType.Disconnect
  Target: string
}

export type ServerMessage =
  | InfoMessage
  | PacketMessage
  | ServerConnectedMessage
  | ServerDisconnectedMessage
export type ClientMessage = PacketMessage | ConnectMessage | DisconnectMessage
