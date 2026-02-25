export type ConnectionType = 'telegram' | 'slack' | 'http' | 'smtp'

export type Connection = {
  id: string
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  extras?: Record<string, string>
}

export type ConnectionCreate = {
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  password?: string
  token?: string
  extras?: Record<string, string>
}
