const PRESETS: Record<string, string> = {
  '0 * * * *': 'Hourly',
  '0 */3 * * *': 'Every 3h',
  '0 */6 * * *': 'Every 6h',
  '0 */12 * * *': 'Every 12h',
  '0 9 * * *': 'Daily 9AM',
  '0 9 * * 1-5': 'Weekdays 9AM',
  '0 9 * * 1': 'Weekly Mon',
  '0 9 1 * *': 'Monthly 1st',
}

export function humanReadableCron(expr: string): string {
  return PRESETS[expr] ?? expr
}
