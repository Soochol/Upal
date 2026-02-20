import { test, expect } from './fixtures/base'

test.describe('Schedules page — Cron Schedules tab', () => {
  test('shows empty state when no schedules exist', async ({ page }) => {
    await page.goto('/schedules')
    await expect(page.getByText('No schedules configured.')).toBeVisible()
    await expect(page.getByText('Total Schedules')).toBeVisible()
  })

  test('creates a schedule via the form', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-sched-create')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByText('Add Schedule').click()
    await expect(page.getByText('New Schedule')).toBeVisible()

    const select = page.getByTestId('workflow-select')
    await expect(select.locator('option[value="e2e-sched-create"]')).toBeAttached()
    await select.selectOption('e2e-sched-create')

    await page.getByRole('button', { name: 'Daily at midnight' }).click()
    await expect(page.getByTestId('cron-input')).toHaveValue('0 0 * * *')

    await page.getByRole('button', { name: 'Create Schedule' }).click()

    // Row should appear with correct data
    const row = page.locator('[data-testid^="schedule-row-"]')
    await expect(row).toBeVisible()
    await expect(row.getByText('e2e-sched-create')).toBeVisible()
    await expect(row.locator('code', { hasText: '0 0 * * *' })).toBeVisible()
  })

  test('cron presets update the input value', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-presets')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()
    await page.getByText('Add Schedule').click()

    const cronInput = page.getByTestId('cron-input')

    await expect(cronInput).toHaveValue('0 * * * *')

    await page.getByRole('button', { name: 'Every 6 hours' }).click()
    await expect(cronInput).toHaveValue('0 */6 * * *')

    await page.getByRole('button', { name: 'Daily at midnight' }).click()
    await expect(cronInput).toHaveValue('0 0 * * *')

    await page.getByRole('button', { name: 'Weekly Mon 9am' }).click()
    await expect(cronInput).toHaveValue('0 9 * * 1')

    await page.getByRole('button', { name: 'Every hour' }).click()
    await expect(cronInput).toHaveValue('0 * * * *')
  })

  test('custom cron expression is used in created schedule', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-custom-cron')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByText('Add Schedule').click()
    const select = page.getByTestId('workflow-select')
    await expect(select.locator('option[value="e2e-custom-cron"]')).toBeAttached()
    await select.selectOption('e2e-custom-cron')

    const cronInput = page.getByTestId('cron-input')
    await cronInput.fill('*/5 * * * *')

    await page.getByRole('button', { name: 'Create Schedule' }).click()

    // Verify cron code badge in the row
    const row = page.locator('[data-testid^="schedule-row-"]')
    await expect(row.locator('code', { hasText: '*/5 * * * *' })).toBeVisible()
  })

  test('timezone selection persists in created schedule', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-tz')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByText('Add Schedule').click()
    const select = page.getByTestId('workflow-select')
    await expect(select.locator('option[value="e2e-tz"]')).toBeAttached()
    await select.selectOption('e2e-tz')
    await page.getByTestId('timezone-select').selectOption('Asia/Seoul')
    await page.getByRole('button', { name: 'Create Schedule' }).click()

    const row = page.locator('[data-testid^="schedule-row-"]')
    await expect(row.getByText('Asia/Seoul')).toBeVisible()
  })

  test('pause and resume a schedule', async ({ page, seedWorkflow, seedSchedule }) => {
    await seedWorkflow('e2e-pause')
    const schedId = await seedSchedule('e2e-pause')
    await page.goto('/schedules')

    const row = page.getByTestId(`schedule-row-${schedId}`)
    await expect(row).toBeVisible()

    // Initially active
    await expect(row.getByText('active', { exact: true })).toBeVisible()

    // Pause — wait for the reload to complete
    await row.getByRole('button', { name: 'Pause' }).click()
    await expect(page.getByTestId(`schedule-row-${schedId}`).getByText('paused', { exact: true })).toBeVisible({ timeout: 10000 })

    // Resume
    await page.getByTestId(`schedule-row-${schedId}`).getByRole('button', { name: 'Resume' }).click()
    await expect(page.getByTestId(`schedule-row-${schedId}`).getByText('active', { exact: true })).toBeVisible({ timeout: 10000 })
  })

  test('delete a schedule with confirmation', async ({ page, seedWorkflow, seedSchedule }) => {
    await seedWorkflow('e2e-del')
    const schedId = await seedSchedule('e2e-del')
    await page.goto('/schedules')

    await expect(page.getByTestId(`schedule-row-${schedId}`)).toBeVisible()

    page.on('dialog', (dialog) => dialog.accept())
    await page.getByTestId(`schedule-row-${schedId}`).getByRole('button', { name: 'Delete' }).click()

    await expect(page.getByText('No schedules configured.')).toBeVisible({ timeout: 10000 })
  })

  test('cancel delete keeps the schedule', async ({ page, seedWorkflow, seedSchedule }) => {
    await seedWorkflow('e2e-cancel-del')
    const schedId = await seedSchedule('e2e-cancel-del')
    await page.goto('/schedules')

    const row = page.getByTestId(`schedule-row-${schedId}`)
    await expect(row).toBeVisible()

    page.on('dialog', (dialog) => dialog.dismiss())
    await row.getByRole('button', { name: 'Delete' }).click()

    // Row should remain
    await expect(row).toBeVisible()
  })

  test('create button is disabled without workflow selection', async ({ page, seedWorkflow }) => {
    await seedWorkflow('e2e-disabled')
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByText('Add Schedule').click()

    const createBtn = page.getByRole('button', { name: 'Create Schedule' })
    await expect(createBtn).toBeDisabled()

    const select = page.getByTestId('workflow-select')
    await expect(select.locator('option[value="e2e-disabled"]')).toBeAttached()
    await select.selectOption('e2e-disabled')
    await expect(createBtn).toBeEnabled()
  })

  test('cancel button closes the form', async ({ page }) => {
    await page.goto('/schedules')
    await expect(page.getByText('Add Schedule')).toBeVisible()

    await page.getByText('Add Schedule').click()
    await expect(page.getByText('New Schedule')).toBeVisible()

    await page.getByRole('button', { name: 'Cancel' }).click()

    await expect(page.getByText('New Schedule')).not.toBeVisible()
    await expect(page.getByText('Add Schedule')).toBeVisible()
  })
})
