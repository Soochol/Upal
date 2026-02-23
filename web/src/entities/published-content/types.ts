export type PublishedChannel = 'youtube' | 'substack' | 'discord' | 'telegram' | 'other'

export type PublishedContent = {
  id: string
  pipeline_id: string
  pipeline_name: string
  session_id: string
  session_number: number
  run_id: string
  channel: PublishedChannel
  title: string
  url?: string
  published_at: string
}
