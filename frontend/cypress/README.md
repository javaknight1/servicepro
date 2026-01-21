# ServicePro E2E Tests

End-to-end testing suite for ServicePro using Cypress.

## Quick Start

```bash
# Install dependencies
npm install

# Open Cypress Test Runner (interactive mode)
npm run cy:open

# Run all tests headlessly
npm run cy:run

# Run specific test suite
npm run cy:run -- --spec "cypress/e2e/auth/**/*"
```

## Project Structure

```
cypress/
├── e2e/                    # Test files
│   ├── auth/              # Authentication tests
│   │   ├── login.cy.ts
│   │   └── register.cy.ts
│   ├── customers/         # Customer management tests
│   │   └── customer-crud.cy.ts
│   ├── jobs/              # Job scheduling tests
│   │   └── job-scheduling.cy.ts
│   ├── quotes/            # Quote workflow tests
│   │   └── quote-workflow.cy.ts
│   ├── invoices/          # Invoice workflow tests
│   │   └── invoice-workflow.cy.ts
│   └── payments/          # Payment processing tests
│       └── payment-processing.cy.ts
├── fixtures/              # Test data
│   ├── user.json
│   ├── customers.json
│   ├── jobs.json
│   ├── quotes.json
│   └── invoices.json
├── pages/                 # Page Object Models
│   ├── BasePage.ts
│   ├── LoginPage.ts
│   ├── RegisterPage.ts
│   ├── DashboardPage.ts
│   ├── CustomersPage.ts
│   ├── JobsPage.ts
│   ├── QuotesPage.ts
│   ├── InvoicesPage.ts
│   └── PaymentsPage.ts
├── support/               # Support files
│   ├── commands/          # Custom commands
│   │   ├── index.ts       # Core commands
│   │   ├── auth.ts        # Authentication commands
│   │   ├── api.ts         # API commands
│   │   ├── navigation.ts  # Navigation commands
│   │   └── forms.ts       # Form commands
│   └── e2e.ts             # Support file loaded before tests
├── reports/               # Test reports (gitignored)
├── screenshots/           # Failure screenshots (gitignored)
├── videos/                # Test videos (gitignored)
└── scripts/               # Helper scripts
    └── run-tests.sh
```

## Running Tests

### Interactive Mode

```bash
# Open Cypress Test Runner
npm run cy:open
```

### Headless Mode

```bash
# Run all tests
npm run cy:run

# Run specific spec
npm run cy:run -- --spec "cypress/e2e/auth/login.cy.ts"

# Run tests in specific browser
npm run cy:run -- --browser firefox

# Run tests against staging
npm run cy:run -- --env environment=staging
```

### Using the Runner Script

```bash
# Make script executable
chmod +x cypress/scripts/run-tests.sh

# Run with defaults
./cypress/scripts/run-tests.sh

# Run against staging
./cypress/scripts/run-tests.sh -e staging

# Run specific tests in headed mode
./cypress/scripts/run-tests.sh -s "cypress/e2e/auth/**/*" --headed
```

## Custom Commands

### Authentication

```typescript
// Login via UI (with session caching)
cy.login();
cy.login('custom@email.com', 'password');

// Login as specific roles
cy.loginAsAdmin();
cy.loginAsUser();

// Login via API (faster)
cy.apiLogin('email@example.com', 'password');

// Logout
cy.logout();

// Register new user
cy.register({ email, password, firstName, lastName });

// Visit page with authentication
cy.visitWithAuth('/dashboard');
```

### API Commands

```typescript
// Make authenticated API request
cy.apiRequest({ method: 'GET', url: '/api/v1/customers' });

// Stub API endpoints
cy.stubApi('GET', '**/customers*', 'customers.json', 'getCustomers');
cy.stubApiError('POST', '**/login', 401, 'Invalid credentials', 'loginError');

// Spy on API calls
cy.spyOnApi('POST', '**/api/v1/jobs', 'createJob');

// Simulate network conditions
cy.simulateSlowNetwork(2000);
cy.simulateNetworkFailure('**/api/**', 'networkError');
```

### Form Commands

```typescript
// Fill form with data
cy.fillForm({
  email: 'test@example.com',
  password: 'password123',
  rememberMe: true,
});

// Submit form
cy.submitForm();

// Assert validation errors
cy.assertFieldError('email', 'Invalid email format');
cy.assertNoFieldError('password');

// Fill specialized inputs
cy.fillAddress({ street, city, state, zip });
cy.fillDatePicker('[name="date"]', '2024-02-15');
cy.fillMoneyField('[name="amount"]', 99.99);
```

### Navigation Commands

```typescript
// Navigate to named pages
cy.navigateTo('dashboard');
cy.navigateTo('customers');

// Navigate to detail pages
cy.goToCustomer('cust-123');
cy.goToJob('job-456');

// Assert URL
cy.assertUrl('/customers');
cy.assertPageTitle('Customers');
```

### Utility Commands

```typescript
// Select by test ID
cy.getByTestId('submit-button').click();

// Wait for API calls
cy.waitForApi('getCustomers');

// Assert toast notifications
cy.assertToast('Success!', 'success');

// Take screenshot
cy.matchImageSnapshot('login-page');
```

## Page Objects

Page objects encapsulate page-specific selectors and actions:

```typescript
import { CustomersPage } from '../pages';

describe('Customers', () => {
  const customersPage = new CustomersPage();

  it('should create a customer', () => {
    customersPage.visit();
    customersPage.createCustomer({
      firstName: 'John',
      lastName: 'Doe',
      email: 'john@example.com',
    });
    customersPage.assertToast('Customer created', 'success');
  });
});
```

## Network Stubbing

### Stubbing API Responses

```typescript
// Stub with fixture
cy.intercept('GET', '**/api/v1/customers*', { fixture: 'customers.json' }).as(
  'getCustomers'
);

// Stub inline response
cy.intercept('POST', '**/api/v1/login', {
  statusCode: 200,
  body: { token: 'fake-token' },
}).as('login');

// Stub error response
cy.intercept('DELETE', '**/api/v1/customers/*', {
  statusCode: 403,
  body: { error: 'Cannot delete customer with active jobs' },
}).as('deleteError');
```

### Waiting for API Calls

```typescript
cy.intercept('GET', '**/api/v1/customers*').as('getCustomers');
cy.visit('/customers');
cy.wait('@getCustomers');
```

## Test Reports

After running tests, reports are generated in `cypress/reports/`:

- **HTML Report**: `cypress/reports/merged-report.html`
- **JSON Report**: `cypress/reports/merged-report.json`
- **Screenshots**: `cypress/screenshots/` (on failure)
- **Videos**: `cypress/videos/`

To generate merged report manually:

```bash
npx mochawesome-merge cypress/reports/mochawesome/*.json -o cypress/reports/merged-report.json
npx marge cypress/reports/merged-report.json --reportDir cypress/reports --reportFilename merged-report --inline
```

## Environment Configuration

Tests can run against different environments:

| Environment | Base URL                       | API URL                            |
| ----------- | ------------------------------ | ---------------------------------- |
| development | http://localhost:3000          | http://localhost:8080              |
| staging     | https://staging.servicepro.app | https://staging-api.servicepro.app |
| production  | https://app.servicepro.app     | https://api.servicepro.app         |

Configure via command line:

```bash
npm run cy:run -- --env environment=staging
```

Or in `cypress.config.ts`:

```typescript
env: {
  environment: 'development';
}
```

## Best Practices

### Test Structure

```typescript
describe('Feature', () => {
  beforeEach(() => {
    cy.login();
    cy.intercept('GET', '**/api/v1/data*', { fixture: 'data.json' });
  });

  describe('Scenario Group', () => {
    it('should do something specific', () => {
      // Arrange - set up test state
      // Act - perform action
      // Assert - verify result
    });
  });
});
```

### Avoid Flaky Tests

- Use `cy.wait('@alias')` instead of `cy.wait(1000)`
- Use `.should()` for assertions (auto-retry)
- Avoid testing implementation details
- Keep tests independent

### Data Management

- Use fixtures for consistent test data
- Clean up data between tests when using real API
- Use factory functions for dynamic data

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  cypress:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: 18
      - name: Install dependencies
        run: npm ci
      - name: Start server
        run: npm start &
      - name: Wait for server
        run: npx wait-on http://localhost:3000
      - name: Run Cypress tests
        run: npm run cy:run
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        if: failure()
        with:
          name: cypress-artifacts
          path: |
            cypress/screenshots
            cypress/videos
            cypress/reports
```

## Troubleshooting

### Common Issues

**Tests timing out**

- Increase timeouts in `cypress.config.ts`
- Check if app is running
- Verify network stubs are correct

**Session not persisting**

- Check `cy.session()` validation function
- Clear cookies/localStorage in `beforeEach` if needed

**Element not found**

- Use more specific selectors
- Add `data-testid` attributes
- Check if element is in DOM with `cy.debug()`

**Network requests not stubbed**

- Verify intercept pattern matches request
- Check intercept is registered before action
- Use wildcards carefully (`**` vs `*`)

### Debugging

```typescript
// Pause test execution
cy.pause();

// Debug current subject
cy.get('.element').debug();

// Log to Cypress command log
cy.log('Custom message');

// Check state
cy.window().then(console.log);
```
