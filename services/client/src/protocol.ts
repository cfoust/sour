export enum MessageType {
  Info,
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
}

export type ConnectMessage = {
  Op: MessageType.Connect
  Target: string
}

export type DisconnectMessage = {
  Op: MessageType.Disconnect
  Target: string
}

export type ServerMessage = InfoMessage | PacketMessage
export type ClientMessage = PacketMessage | ConnectMessage | DisconnectMessage
