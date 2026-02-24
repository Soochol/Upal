export type PublishChannelType =
  | 'wordpress' | 'youtube' | 'slack' | 'telegram'
  | 'substack' | 'discord' | 'medium' | 'tiktok' | 'http'

export type PublishChannel = {
  id: string
  name: string
  type: PublishChannelType
}
