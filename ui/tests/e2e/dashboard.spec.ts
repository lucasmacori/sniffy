import { test, expect } from '@playwright/test'

test.describe('Sniffy Dashboard', () => {
  test('findings page loads', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Credential Findings' })).toBeVisible()
    await expect(page.getByText('Loading findings...')).not.toBeVisible()
  })

  test('statistics page loads', async ({ page }) => {
    await page.goto('/stats')
    await expect(page.getByRole('heading', { name: 'Statistics' })).toBeVisible()
  })

  test('navigation works', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: 'Statistics' }).click()
    await expect(page).toHaveURL('/stats')
    await page.getByRole('link', { name: 'Findings' }).click()
    await expect(page).toHaveURL('/')
  })
})
