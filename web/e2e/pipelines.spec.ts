import { test, expect } from './fixtures/base'

test.describe('Pipelines page & Split View Details', () => {
    test('shows empty state when no pipelines exist', async ({ page }) => {
        await page.goto('/pipelines')
        await expect(page.getByRole('heading', { name: 'Pipelines' })).toBeVisible()
        await expect(page.getByText('파이프라인이 없습니다')).toBeVisible()
    })

    test('can create a pipeline and view the split details', async ({ page }) => {
        // 1. Visit Pipelines
        await page.goto('/pipelines')

        // 2. Click New Pipeline
        await page.getByRole('button', { name: 'New Pipeline' }).click()

        // 3. Fill in the modal
        const nameInput = page.getByRole('textbox', { name: /name/i }).first()
        await nameInput.waitFor({ state: 'visible' })
        await nameInput.fill('e2e-pipeline')
        await page.getByRole('button', { name: 'Create Pipeline' }).click()

        // 4. Should redirect to the new split-view pipeline details page
        await expect(page).toHaveURL(/\/pipelines\/[\w-]+$/)

        // 5. Verify the Split View components are visible
        await expect(page.getByText('Session History')).toBeVisible()
        await expect(page.getByText('Data Sources & Schedule')).toBeVisible()
        await expect(page.getByText('Editorial Brief & Context')).toBeVisible()

        // 6. Verify basic functionality in the right panel (Add Data Source)
        await page.getByRole('button', { name: 'Add Data Source' }).click()
        const sourceSelect = page.getByRole('combobox').first()
        await expect(sourceSelect).toBeVisible()
    })
})
