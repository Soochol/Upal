import { test, expect } from './fixtures/base'

test.describe('Schedules page — Webhook Triggers tab', () => {
  test('shows empty state in webhooks tab', async ({ page }) => {
    await page.goto('/schedules')
    await expect(page.getByRole('heading', { name: 'Schedules' })).toBeVisible()

    await page.getByRole('button', { name: 'Webhook Triggers' }).click()
    await expect(page.getByText('No webhook triggers configured.')).toBeVisible()
  })

  test('creates a webhook trigger', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-trigger-create')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByRole('button', { name: 'Webhook Triggers' }).click()

    await page.getByText('Add Webhook').click()
    await expect(page.getByText('New Webhook Trigger')).toBeVisible()

    const select = page.getByTestId('trigger-workflow-select')
    await expect(select.locator('option[value="e2e-trigger-create"]')).toBeAttached()
    await select.selectOption('e2e-trigger-create')

    await page.getByRole('button', { name: 'Create Trigger' }).click()

    const row = page.locator('[data-testid^="trigger-row-"]')
    await expect(row).toBeVisible()
    await expect(row.getByText('e2e-trigger-create')).toBeVisible()
    await expect(page.getByText(/\/api\/hooks\//)).toBeVisible()
  })

  test('deletes a webhook trigger', async ({ page, seedWorkflow, seedTrigger }) => {
    await seedWorkflow('e2e-trigger-del')
    const trigId = await seedTrigger('e2e-trigger-del')
    await page.goto('/schedules')

    await page.getByRole('button', { name: 'Webhook Triggers' }).click()

    const row = page.getByTestId(`trigger-row-${trigId}`)
    await expect(row).toBeVisible()

    page.on('dialog', (dialog) => dialog.accept())
    await row.getByRole('button', { name: 'Delete' }).click()

    await expect(page.getByText('No webhook triggers configured.')).toBeVisible({ timeout: 10000 })
  })

  test('tab switching preserves data in each tab', async ({ page, seedWorkflow, seedSchedule, seedTrigger }) => {
    await seedWorkflow('e2e-tab-switch')
    const schedId = await seedSchedule('e2e-tab-switch')
    await seedTrigger('e2e-tab-switch')
    await page.goto('/schedules')

    // Cron tab — schedule visible
    const schedRow = page.getByTestId(`schedule-row-${schedId}`)
    await expect(schedRow).toBeVisible()

    // Switch to Webhooks tab — trigger visible
    await page.getByRole('button', { name: 'Webhook Triggers' }).click()
    await expect(page.getByText(/\/api\/hooks\//)).toBeVisible()

    // Switch back to Cron tab — schedule still there
    await page.getByRole('button', { name: 'Cron Schedules' }).click()
    await expect(schedRow).toBeVisible()
  })
})
