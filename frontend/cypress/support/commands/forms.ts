// =============================================================================
// Form Commands
// =============================================================================

/**
 * Fill form fields with provided data
 */
Cypress.Commands.add(
  'fillForm',
  (formData: Record<string, string | number | boolean>) => {
    Object.entries(formData).forEach(([fieldName, value]) => {
      const selectors = [
        `[name="${fieldName}"]`,
        `[data-testid="${fieldName}"]`,
        `[data-testid="${fieldName}-input"]`,
        `#${fieldName}`,
      ];

      cy.get(selectors.join(', '))
        .first()
        .then(($el) => {
          const tagName = $el.prop('tagName').toLowerCase();
          const inputType = $el.attr('type')?.toLowerCase();

          if (tagName === 'select') {
            // Handle select element
            cy.wrap($el).select(String(value));
          } else if (inputType === 'checkbox') {
            // Handle checkbox
            if (value) {
              cy.wrap($el).check({ force: true });
            } else {
              cy.wrap($el).uncheck({ force: true });
            }
          } else if (inputType === 'radio') {
            // Handle radio button
            cy.wrap($el).check(String(value), { force: true });
          } else if (inputType === 'file') {
            // Handle file input - value should be fixture filename
            cy.wrap($el).selectFile(`cypress/fixtures/${value}`, {
              force: true,
            });
          } else {
            // Handle text inputs, textareas, etc.
            cy.wrap($el).clear().type(String(value));
          }
        });
    });
  }
);

/**
 * Submit form
 */
Cypress.Commands.add('submitForm', () => {
  cy.get(
    'form button[type="submit"], form [data-testid="submit-button"], form input[type="submit"]'
  )
    .first()
    .click();
});

/**
 * Clear all form fields
 */
Cypress.Commands.add('clearForm', () => {
  cy.get('form').within(() => {
    cy.get(
      'input[type="text"], input[type="email"], input[type="password"], input[type="number"], textarea'
    ).each(($el) => {
      cy.wrap($el).clear();
    });

    cy.get('input[type="checkbox"]:checked').each(($el) => {
      cy.wrap($el).uncheck({ force: true });
    });
  });
});

/**
 * Select option from dropdown
 */
Cypress.Commands.add('selectOption', (selector: string, value: string) => {
  cy.get(selector).then(($el) => {
    const tagName = $el.prop('tagName').toLowerCase();

    if (tagName === 'select') {
      // Native select
      cy.wrap($el).select(value);
    } else {
      // Custom dropdown (e.g., React Select, MUI Select)
      cy.wrap($el).click();
      cy.get(
        '[role="listbox"], [role="option"], .dropdown-menu, .select-options'
      )
        .contains(value)
        .click();
    }
  });
});

/**
 * Check checkbox
 */
Cypress.Commands.add('checkCheckbox', (selector: string) => {
  cy.get(selector).check({ force: true });
});

/**
 * Uncheck checkbox
 */
Cypress.Commands.add('uncheckCheckbox', (selector: string) => {
  cy.get(selector).uncheck({ force: true });
});

/**
 * Assert form validation error
 */
Cypress.Commands.add(
  'assertFieldError',
  (fieldName: string, errorMessage: string) => {
    const errorSelectors = [
      `[data-testid="${fieldName}-error"]`,
      `[name="${fieldName}"] + .error`,
      `[name="${fieldName}"] ~ .error-message`,
      `#${fieldName}-error`,
      `.field-error[data-field="${fieldName}"]`,
    ];

    cy.get(errorSelectors.join(', '))
      .should('be.visible')
      .and('contain.text', errorMessage);
  }
);

/**
 * Assert no validation errors on field
 */
Cypress.Commands.add('assertNoFieldError', (fieldName: string) => {
  cy.get(
    `[data-testid="${fieldName}-error"], [name="${fieldName}"] + .error`
  ).should('not.exist');
});

/**
 * Fill address fields
 */
Cypress.Commands.add(
  'fillAddress',
  (address: {
    street?: string;
    city?: string;
    state?: string;
    zip?: string;
    country?: string;
  }) => {
    if (address.street) {
      cy.get('[name="street"], [name="address"], [data-testid="street-input"]')
        .clear()
        .type(address.street);
    }
    if (address.city) {
      cy.get('[name="city"], [data-testid="city-input"]')
        .clear()
        .type(address.city);
    }
    if (address.state) {
      cy.get('[name="state"], [data-testid="state-input"]').then(($el) => {
        if ($el.prop('tagName').toLowerCase() === 'select') {
          cy.wrap($el).select(address.state!);
        } else {
          cy.wrap($el).clear().type(address.state!);
        }
      });
    }
    if (address.zip) {
      cy.get(
        '[name="zip"], [name="postalCode"], [name="zipCode"], [data-testid="zip-input"]'
      )
        .clear()
        .type(address.zip);
    }
    if (address.country) {
      cy.get('[name="country"], [data-testid="country-input"]').then(($el) => {
        if ($el.prop('tagName').toLowerCase() === 'select') {
          cy.wrap($el).select(address.country!);
        } else {
          cy.wrap($el).clear().type(address.country!);
        }
      });
    }
  }
);

/**
 * Fill date picker
 */
Cypress.Commands.add('fillDatePicker', (selector: string, date: string) => {
  cy.get(selector).then(($el) => {
    const inputType = $el.attr('type');

    if (inputType === 'date') {
      // Native date input
      cy.wrap($el).clear().type(date);
    } else {
      // Custom date picker
      cy.wrap($el).click();
      // Try to find and click the date in calendar
      cy.get('.calendar, .date-picker, [role="dialog"]')
        .contains(date.split('-').pop() || date)
        .click();
    }
  });
});

/**
 * Fill time picker
 */
Cypress.Commands.add('fillTimePicker', (selector: string, time: string) => {
  cy.get(selector).then(($el) => {
    const inputType = $el.attr('type');

    if (inputType === 'time') {
      // Native time input
      cy.wrap($el).clear().type(time);
    } else {
      // Custom time picker
      cy.wrap($el).clear().type(time);
    }
  });
});

/**
 * Fill currency/money field
 */
Cypress.Commands.add('fillMoneyField', (selector: string, amount: number) => {
  cy.get(selector).clear().type(amount.toFixed(2));
});

/**
 * Assert form field value
 */
Cypress.Commands.add(
  'assertFieldValue',
  (fieldName: string, expectedValue: string | number | boolean) => {
    const selectors = [
      `[name="${fieldName}"]`,
      `[data-testid="${fieldName}"]`,
      `#${fieldName}`,
    ];

    cy.get(selectors.join(', '))
      .first()
      .then(($el) => {
        const inputType = $el.attr('type')?.toLowerCase();

        if (inputType === 'checkbox') {
          if (expectedValue) {
            cy.wrap($el).should('be.checked');
          } else {
            cy.wrap($el).should('not.be.checked');
          }
        } else {
          cy.wrap($el).should('have.value', String(expectedValue));
        }
      });
  }
);

/**
 * Upload file to file input
 */
Cypress.Commands.add(
  'uploadFileToInput',
  (selector: string, fileName: string, mimeType?: string) => {
    cy.fixture(fileName, 'base64').then((fileContent) => {
      const blob = Cypress.Blob.base64StringToBlob(
        fileContent,
        mimeType || 'application/octet-stream'
      );
      const testFile = new File([blob], fileName, {
        type: mimeType || 'application/octet-stream',
      });
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(testFile);

      cy.get(selector).then(($el) => {
        const input = $el[0] as HTMLInputElement;
        input.files = dataTransfer.files;
        cy.wrap($el).trigger('change', { force: true });
      });
    });
  }
);

/**
 * Search in autocomplete field
 */
Cypress.Commands.add(
  'searchAutocomplete',
  (selector: string, searchTerm: string, selectOption: string) => {
    cy.get(selector).clear().type(searchTerm);

    // Wait for autocomplete results
    cy.get('[role="listbox"], .autocomplete-results, .suggestions')
      .should('be.visible')
      .contains(selectOption)
      .click();
  }
);

// Add type declarations
declare global {
  namespace Cypress {
    interface Chainable {
      fillForm(
        formData: Record<string, string | number | boolean>
      ): Chainable<void>;
      submitForm(): Chainable<void>;
      clearForm(): Chainable<void>;
      selectOption(selector: string, value: string): Chainable<void>;
      checkCheckbox(selector: string): Chainable<void>;
      uncheckCheckbox(selector: string): Chainable<void>;
      assertFieldError(
        fieldName: string,
        errorMessage: string
      ): Chainable<void>;
      assertNoFieldError(fieldName: string): Chainable<void>;
      fillAddress(address: {
        street?: string;
        city?: string;
        state?: string;
        zip?: string;
        country?: string;
      }): Chainable<void>;
      fillDatePicker(selector: string, date: string): Chainable<void>;
      fillTimePicker(selector: string, time: string): Chainable<void>;
      fillMoneyField(selector: string, amount: number): Chainable<void>;
      assertFieldValue(
        fieldName: string,
        expectedValue: string | number | boolean
      ): Chainable<void>;
      uploadFileToInput(
        selector: string,
        fileName: string,
        mimeType?: string
      ): Chainable<void>;
      searchAutocomplete(
        selector: string,
        searchTerm: string,
        selectOption: string
      ): Chainable<void>;
    }
  }
}

export {};
