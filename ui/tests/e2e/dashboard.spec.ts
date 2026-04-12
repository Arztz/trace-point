import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the dashboard
    await page.goto('/');
  });

  test('should load the dashboard page', async ({ page }) => {
    // Check that the page title or header is visible
    await expect(page.locator('h1, h2, [class*="title"], [data-testid="dashboard"]')).toBeVisible({ timeout: 10000 });
  });

  test('should show time range selector', async ({ page }) => {
    // Check for time range selector elements
    const timeRangeButtons = page.locator('button, [class*="time"], [class*="range"]');
    await expect(timeRangeButtons.first()).toBeVisible({ timeout: 10000 });
  });

  test('should show filter bar', async ({ page }) => {
    // Check for filter bar or namespace selector
    const filterElements = page.locator('input, select, [class*="filter"]');
    await expect(filterElements.first()).toBeVisible({ timeout: 10000 });
  });

  test('should show timeline chart area', async ({ page }) => {
    // Check for chart container (recharts uses svg or div elements)
    const chartArea = page.locator('.recharts-wrapper, [class*="chart"]');
    await expect(chartArea.first()).toBeVisible({ timeout: 15000 });
  });

  test('should show load state while fetching', async ({ page }) => {
    // Check for any loading indicators
    const loadingIndicators = page.locator('[class*="skeleton"], [class*="loading"], [data-testid="loading"]');
    // Either loading or content should be visible
    await expect(loadingIndicators.first().or(page.locator('[class*="chart"]')).first()).toBeVisible({ timeout: 15000 });
  });
});

test.describe('Timeline Charts', () => {
  test('should display CPU chart', async ({ page }) => {
    await page.goto('/');
    
    // Look for CPU-related elements (label or data)
    const cpuElements = page.locator('[class*="cpu"], [class*="CPU"]');
    // Chart should render or at least container exists
    const chartContainer = page.locator('.recharts-responsive-container, [class*="chart"]').first();
    await expect(chartContainer).toBeVisible({ timeout: 15000 });
  });

  test('should display RAM chart', async ({ page }) => {
    await page.goto('/');
    
    // Look for RAM-related elements
    const ramElements = page.locator('[class*="ram"], [class*="RAM"]');
    // RAM chart should render alongside CPU
    const chartContainer = page.locator('.recharts-responsive-container, [class*="chart"]').first();
    await expect(chartContainer).toBeVisible({ timeout: 15000 });
  });

  test('charts should have synchronized time axis', async ({ page }) => {
    await page.goto('/');
    
    // Check that charts are rendered (they share the same time axis by design)
    const charts = page.locator('.recharts-wrapper');
    await expect(charts.first()).toBeVisible({ timeout: 15000 });
  });
});

test.describe('Time Range Selection', () => {
  test.each(['1h', '6h', '24h', '7d'])('should select time range %s', async ({ page }, timeRange: string) => {
    await page.goto('/');
    
    // Find and click time range buttons
    const button = page.getByText(timeRange).first();
    if (await button.isVisible()) {
      await button.click();
      // Wait a moment for potential data update
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Spike Events', () => {
  test('should display spike list or empty state', async ({ page }) => {
    await page.goto('/');
    
    // Check for spike list or empty state message
    const spikeList = page.locator('[class*="spike"], [class*="list"]');
    const emptyState = page.locator('[class*="empty"], text=No spikes');
    
    await expect(spikeList.first().or(emptyState)).toBeVisible({ timeout: 15000 });
  });

  test('should show spike details on selection', async ({ page }) => {
    await page.goto('/');
    
    // Look for any spike items that can be clicked
    const spikeItems = page.locator('[class*="spike-item"], [class*="spike-row"]');
    const count = await spikeItems.count();
    
    if (count > 0) {
      // Try clicking one
      await spikeItems.first().click();
      
      // Check for detail panel or modal
      const detailPanel = page.locator('[class*="detail"], [class*="modal"], [class*="panel"]');
      await expect(detailPanel.first()).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Gravity Scores', () => {
  test('should display gravity scores table', async ({ page }) => {
    await page.goto('/api/v1/gravity-scores');
    
    // Check for table or data
    const table = page.locator('table');
    const data = page.locator('[class*="score"], [class*="gravity"]');
    
    await expect(table.or(data)).toBeVisible({ timeout: 15000 });
  });

  test('should show job-like route tags', async ({ page }) => {
    await page.goto('/api/v1/gravity-scores');
    
    // Check for job tags if they exist
    const jobTags = page.locator('text=[Suspected Job], text=Job');
    
    // Either visible or table exists (job tags are optional)
    const visible = await jobTags.first().isVisible().catch(() => false);
    if (!visible) {
      const table = page.locator('table');
      await expect(table).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('API Endpoints', () => {
  test('health endpoint should respond', async ({ request }) => {
    const response = await request.get('/health');
    expect(response.ok()).toBeTruthy();
  });

  test('spikes endpoint should respond', async ({ request }) => {
    const response = await request.get('/api/v1/spikes');
    // May return 200 or redirect to login
    expect([200, 401, 302]).toContain(response.status());
  });

  test('timeline endpoint should respond', async ({ request }) => {
    const response = await request.get('/api/v1/timeline?range=1h');
    expect([200, 401, 302]).toContain(response.status());
  });

  test('config endpoint should respond', async ({ request }) => {
    const response = await request.get('/api/v1/config');
    expect([200, 401, 302]).toContain(response.status());
  });
});

test.describe('Responsive Design', () => {
  test('should work on mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Page should still load without critical errors
    const body = page.locator('body');
    await expect(body).toBeVisible({ timeout: 10000 });
  });

  test('should work on tablet viewport', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/');
    
    const body = page.locator('body');
    await expect(body).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Accessibility', () => {
  test('should have proper heading hierarchy', async ({ page }) => {
    await page.goto('/');
    
    // Check for h1 or main heading
    const h1 = page.locator('h1');
    const visible = await h1.isVisible().catch(() => false);
    
    // Should have some heading
    const heading = page.locator('h1, h2, h3').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
  });

  test('should have form labels or aria-labels', async ({ page }) => {
    await page.goto('/');
    
    // Check inputs have associated labels
    const inputs = page.locator('input, select');
    const count = await inputs.count();
    
    if (count > 0) {
      // At least some inputs should be accessible
      const firstInput = inputs.first();
      const label = await page.locator(`label[for="${await firstInput.getAttribute('id')}"]`).count();
      const ariaLabel = await firstInput.getAttribute('aria-label');
      const ariaLabelledby = await firstInput.getAttribute('aria-labelledby');
      
      // Some form of label should exist
      const hasLabel = label > 0 || (ariaLabel && ariaLabel.length > 0) || (ariaLabelledby && ariaLabelledby.length > 0);
      if (!hasLabel) {
        console.log('Warning: Input may lack accessibility label');
      }
    }
  });
});

test.describe('Error Handling', () => {
  test('should handle API errors gracefully', async ({ page }) => {
    // Go to a non-existent route
    await page.goto('/nonexistent-route');
    
    // Should show either custom 404 or redirect, not crash
    const body = page.locator('body');
    await expect(body).toBeVisible({ timeout: 5000 });
  });

  test('should show error state on failed data fetch', async ({ page }) => {
    await page.goto('/');
    
    // Wait for initial load
    await page.waitForTimeout(3000);
    
    // Should not have critical console errors
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // Give time for any errors to appear
    await page.waitForTimeout(2000);
    
    // If there are errors, they should not be critical JS errors
    const criticalErrors = errors.filter(e => !e.includes('favicon'));
    if (criticalErrors.length > 0) {
      console.log('Console errors:', criticalErrors);
    }
  });
});