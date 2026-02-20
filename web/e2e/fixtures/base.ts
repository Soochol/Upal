import { test as base, type APIRequestContext } from '@playwright/test'

const API = 'http://localhost:8081'

/** Remove all e2e test data (workflows starting with "e2e-" and their schedules). */
async function cleanupTestData(api: APIRequestContext) {
  // Delete all schedules
  const schedules = await api.get('/api/schedules').then(r => r.json()).catch(() => [])
  for (const s of (schedules ?? [])) {
    if (s.id) await api.delete(`/api/schedules/${s.id}`).catch(() => {})
  }

  // Delete all e2e workflows (and their triggers)
  const workflows = await api.get('/api/workflows').then(r => r.json()).catch(() => [])
  for (const wf of (workflows ?? [])) {
    if (wf.name?.startsWith('e2e-')) {
      // Delete triggers for this workflow
      const triggers = await api
        .get(`/api/workflows/${encodeURIComponent(wf.name)}/triggers`)
        .then(r => r.json())
        .catch(() => [])
      for (const t of (triggers ?? [])) {
        if (t.id) await api.delete(`/api/triggers/${t.id}`).catch(() => {})
      }
      await api.delete(`/api/workflows/${encodeURIComponent(wf.name)}`).catch(() => {})
    }
  }
}

type Fixtures = {
  api: APIRequestContext
  seedWorkflow: (name: string) => Promise<void>
  seedSchedule: (workflowName: string, cronExpr?: string) => Promise<string>
  seedTrigger: (workflowName: string) => Promise<string>
}

export const test = base.extend<Fixtures>({
  api: async ({ playwright }, use) => {
    const ctx = await playwright.request.newContext({ baseURL: API })
    // Clean up before each test for isolation
    await cleanupTestData(ctx)
    await use(ctx)
    // Clean up after each test
    await cleanupTestData(ctx)
    await ctx.dispose()
  },

  seedWorkflow: async ({ api }, use) => {
    const seed = async (name: string) => {
      await api.post('/api/workflows', {
        data: {
          name,
          nodes: [
            { id: 'input_1', type: 'input', label: 'Input', config: { value: 'test' }, position: { x: 0, y: 0 } },
            { id: 'output_1', type: 'output', label: 'Output', config: {}, position: { x: 300, y: 0 } },
          ],
          edges: [{ source: 'input_1', target: 'output_1' }],
        },
      })
    }

    await use(seed)
  },

  seedSchedule: async ({ api }, use) => {
    const seed = async (workflowName: string, cronExpr = '0 * * * *') => {
      const resp = await api.post('/api/schedules', {
        data: {
          workflow_name: workflowName,
          cron_expr: cronExpr,
          timezone: 'UTC',
          enabled: true,
        },
      })
      const schedule = await resp.json()
      return schedule.id as string
    }

    await use(seed)
  },

  seedTrigger: async ({ api }, use) => {
    const seed = async (workflowName: string) => {
      const resp = await api.post('/api/triggers', {
        data: {
          workflow_name: workflowName,
          type: 'webhook',
          enabled: true,
        },
      })
      const body = await resp.json()
      return (body.trigger?.id ?? body.id) as string
    }

    await use(seed)
  },
})

export { expect } from '@playwright/test'
