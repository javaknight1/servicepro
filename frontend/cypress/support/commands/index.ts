// =============================================================================
// Core Custom Commands
// =============================================================================

// =============================================================================
// Element Selection Commands
// =============================================================================

/**
 * Get element by data-testid attribute
 */
Cypress.Commands.add('getByTestId', (testId: string) => {
  return cy.get(`[data-testid="${testId}"]`);
});

/**
 * Get element by role with optional name
 */
Cypress.Commands.add(
  'getByRole',
  (role: string, options?: { name?: string | RegExp }) => {
    if (options?.name) {
      return cy.get(`[role="${role}"]`).filter(`:contains("${options.name}")`);
    }
    return cy.get(`[role="${role}"]`);
  }
);

// =============================================================================
// Visibility Assertions
// =============================================================================

/**
 * Assert element is visible
 */
Cypress.Commands.add(
  'shouldBeVisible',
  { prevSubject: 'element' },
  (subject) => {
    cy.wrap(subject).should('be.visible');
  }
);

/**
 * Assert element does not exist
 */
Cypress.Commands.add(
  'shouldNotExist',
  { prevSubject: 'element' },
  (subject) => {
    cy.wrap(subject).should('not.exist');
  }
);

// =============================================================================
// Wait Commands
// =============================================================================

/**
 * Wait for an API call to complete
 */
Cypress.Commands.add('waitForApi', (alias: string) => {
  cy.wait(`@${alias}`);
});

/**
 * Wait for page to fully load
 */
Cypress.Commands.add('waitForPageLoad', () => {
  cy.document().its('readyState').should('eq', 'complete');
  cy.get('body').should('be.visible');
});

// =============================================================================
// Toast/Notification Assertions
// =============================================================================

/**
 * Assert a toast notification appears with message
 */
Cypress.Commands.add(
  'assertToast',
  (message: string, type?: 'success' | 'error' | 'warning') => {
    const toastSelector = type
      ? `[data-testid="toast-${type}"], .toast-${type}, [class*="${type}"]`
      : '[data-testid="toast"], .toast, [role="alert"]';

    cy.get(toastSelector).should('be.visible').and('contain.text', message);
  }
);

// =============================================================================
// Local Storage Commands
// =============================================================================

/**
 * Get item from local storage
 */
Cypress.Commands.add('getLocalStorage', (key: string) => {
  return cy.window().then((win) => {
    return win.localStorage.getItem(key);
  });
});

/**
 * Set item in local storage
 */
Cypress.Commands.add('setLocalStorage', (key: string, value: string) => {
  cy.window().then((win) => {
    win.localStorage.setItem(key, value);
  });
});

// =============================================================================
// Visual Testing Commands (if enabled)
// =============================================================================

/**
 * Take and compare screenshot for visual regression
 */
Cypress.Commands.add('matchImageSnapshot', (name?: string) => {
  if (Cypress.env('enableVisualTesting')) {
    // Using cypress-image-snapshot or similar
    const snapshotName = name || Cypress.currentTest.title.replace(/\s+/g, '-');
    cy.screenshot(snapshotName, { capture: 'viewport' });
  }
});

// =============================================================================
// Drag and Drop Commands
// =============================================================================

/**
 * Drag element to target
 */
Cypress.Commands.add(
  'dragTo',
  { prevSubject: 'element' },
  (subject, targetSelector: string) => {
    cy.wrap(subject).trigger('dragstart');
    cy.get(targetSelector).trigger('drop');
    cy.wrap(subject).trigger('dragend');
  }
);

// =============================================================================
// File Upload Commands
// =============================================================================

/**
 * Upload file to input
 */
Cypress.Commands.add(
  'uploadFile',
  { prevSubject: 'element' },
  (subject, fileName: string, mimeType: string) => {
    cy.fixture(fileName, 'base64').then((fileContent) => {
      const blob = Cypress.Blob.base64StringToBlob(fileContent, mimeType);
      const testFile = new File([blob], fileName, { type: mimeType });
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(testFile);

      cy.wrap(subject).then((el) => {
        (el[0] as HTMLInputElement).files = dataTransfer.files;
        cy.wrap(subject).trigger('change', { force: true });
      });
    });
  }
);

export {};
