export type SurgeAlert = {
  id: string
  pipeline_id: string
  pipeline_name: string
  keyword: string
  multiplier: number   // e.g. 10 = 10× spike
  detected_at: string
  dismissed: boolean
}
