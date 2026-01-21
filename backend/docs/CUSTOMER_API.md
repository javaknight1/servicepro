# Customer Management API Documentation

## Overview

Complete RESTful API for customer management with JWT authentication, permission-based access control, and comprehensive validation.

## Authentication

All customer endpoints require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

Get a token by logging in via `/api/v1/auth/login`.

## Permissions Required

| Endpoint                             | Permission Required |
| ------------------------------------ | ------------------- |
| `POST /api/v1/customers`             | `customers.create`  |
| `GET /api/v1/customers`              | `customers.list`    |
| `GET /api/v1/customers/search`       | `customers.list`    |
| `GET /api/v1/customers/:id`          | `customers.read`    |
| `GET /api/v1/customers/email/:email` | `customers.read`    |
| `PUT /api/v1/customers/:id`          | `customers.update`  |
| `DELETE /api/v1/customers/:id`       | `customers.delete`  |

## Base URL

```
http://localhost:8080/api/v1
```

## Endpoints

### 1. Create Customer

Create a new customer record.

**Endpoint:** `POST /api/v1/customers`

**Headers:**

```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body:**

```json
{
  "first_name": "John",
  "last_name": "Doe",
  "company_name": "Acme Corp",
  "email": "john.doe@example.com",
  "phone_primary": "555-123-4567",
  "phone_secondary": "555-987-6543",
  "billing_address_street": "123 Main Street",
  "billing_address_city": "Springfield",
  "billing_address_state": "IL",
  "billing_address_zip": "62701",
  "service_address_street": "456 Service Ave",
  "service_address_city": "Chicago",
  "service_address_state": "IL",
  "service_address_zip": "60601",
  "customer_type": "commercial",
  "status": "active",
  "notes": "Preferred customer"
}
```

**Field Validation:**

- `first_name`: Required, max 50 characters
- `last_name`: Required, max 50 characters
- `company_name`: Optional, max 100 characters
- `email`: Required, valid email format, max 255 characters, must be unique
- `phone_primary`: Required, US phone format, max 20 characters
- `phone_secondary`: Optional, US phone format, max 20 characters
- `billing_address_*`: All required
- `service_address_*`: All optional (but if one provided, all must be provided)
- `customer_type`: Required, enum: `residential` or `commercial`
- `status`: Optional, enum: `active`, `inactive`, or `prospect` (defaults to `prospect`)
- `notes`: Optional

**Success Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "first_name": "John",
  "last_name": "Doe",
  "display_name": "Acme Corp (John Doe)",
  "company_name": "Acme Corp",
  "email": "john.doe@example.com",
  "phone_primary": "555-123-4567",
  "phone_secondary": "555-987-6543",
  "billing_address_street": "123 Main Street",
  "billing_address_city": "Springfield",
  "billing_address_state": "IL",
  "billing_address_zip": "62701",
  "billing_address_full": "123 Main Street, Springfield, IL 62701",
  "service_address_street": "456 Service Ave",
  "service_address_city": "Chicago",
  "service_address_state": "IL",
  "service_address_zip": "60601",
  "service_address_full": "456 Service Ave, Chicago, IL 60601",
  "customer_type": "commercial",
  "status": "active",
  "notes": "Preferred customer",
  "created_at": "2024-01-20T12:00:00Z",
  "updated_at": "2024-01-20T12:00:00Z"
}
```

**Error Responses:**

400 Bad Request - Invalid input:

```json
{
  "error": "invalid_request",
  "message": "Validation error message"
}
```

409 Conflict - Email already exists:

```json
{
  "error": "email_exists",
  "message": "A customer with this email already exists"
}
```

---

### 2. Get Customer by ID

Retrieve a specific customer by their UUID.

**Endpoint:** `GET /api/v1/customers/:id`

**Headers:**

```
Authorization: Bearer <jwt_token>
```

**URL Parameters:**

- `id` (UUID): Customer ID

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "first_name": "John",
  "last_name": "Doe",
  "display_name": "John Doe",
  "email": "john.doe@example.com",
  ...
}
```

**Error Responses:**

400 Bad Request - Invalid UUID:

```json
{
  "error": "invalid_id",
  "message": "Invalid customer ID format"
}
```

404 Not Found:

```json
{
  "error": "customer_not_found",
  "message": "Customer not found"
}
```

---

### 3. Get Customer by Email

Retrieve a customer by their email address.

**Endpoint:** `GET /api/v1/customers/email/:email`

**Headers:**

```
Authorization: Bearer <jwt_token>
```

**URL Parameters:**

- `email` (string): Customer email address

**Success Response (200 OK):**
Same as Get Customer by ID

**Error Responses:**

404 Not Found:

```json
{
  "error": "customer_not_found",
  "message": "Customer not found"
}
```

---

### 4. List Customers

Get a paginated list of customers with optional filtering and sorting.

**Endpoint:** `GET /api/v1/customers`

**Headers:**

```
Authorization: Bearer <jwt_token>
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `search` | string | - | Search term for name, email, company, phone |
| `customer_type` | enum | - | Filter by type: `residential` or `commercial` |
| `status` | enum | - | Filter by status: `active`, `inactive`, `prospect` |
| `state` | string | - | Filter by billing state (2-letter code) |
| `city` | string | - | Filter by billing city |
| `page` | int | 1 | Page number (starts at 1) |
| `page_size` | int | 20 | Items per page (max 100) |
| `sort_by` | string | created_at | Sort field |
| `sort_order` | enum | DESC | Sort order: `ASC` or `DESC` |

**Example Request:**

```
GET /api/v1/customers?search=john&status=active&page=1&page_size=20&sort_by=last_name&sort_order=ASC
```

**Success Response (200 OK):**

```json
{
  "customers": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "first_name": "John",
      "last_name": "Doe",
      ...
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "first_name": "Jane",
      "last_name": "Smith",
      ...
    }
  ],
  "total": 45,
  "page": 1,
  "page_size": 20,
  "total_pages": 3
}
```

---

### 5. Search Customers

Quick search for customers (returns up to 50 results).

**Endpoint:** `GET /api/v1/customers/search`

**Headers:**

```
Authorization: Bearer <jwt_token>
```

**Query Parameters:**

- `q` (required): Search query

**Example Request:**

```
GET /api/v1/customers/search?q=john
```

**Success Response (200 OK):**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "first_name": "John",
    "last_name": "Doe",
    ...
  }
]
```

**Error Responses:**

400 Bad Request - Missing query:

```json
{
  "error": "invalid_request",
  "message": "Search query is required"
}
```

---

### 6. Update Customer

Update an existing customer's information.

**Endpoint:** `PUT /api/v1/customers/:id`

**Headers:**

```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**URL Parameters:**

- `id` (UUID): Customer ID

**Request Body (all fields optional):**

```json
{
  "first_name": "Jane",
  "last_name": "Smith",
  "email": "jane.smith@example.com",
  "phone_primary": "555-999-8888",
  "status": "inactive",
  "notes": "Updated notes"
}
```

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "first_name": "Jane",
  "last_name": "Smith",
  ...
}
```

**Error Responses:**

400 Bad Request - Invalid input:

```json
{
  "error": "invalid_request",
  "message": "Validation error"
}
```

404 Not Found:

```json
{
  "error": "customer_not_found",
  "message": "Customer not found"
}
```

409 Conflict - Email already exists (if updating email):

```json
{
  "error": "email_exists",
  "message": "A customer with this email already exists"
}
```

---

### 7. Delete Customer

Soft delete a customer (sets `deleted_at` timestamp).

**Endpoint:** `DELETE /api/v1/customers/:id`

**Headers:**

```
Authorization: Bearer <jwt_token>
```

**URL Parameters:**

- `id` (UUID): Customer ID

**Success Response (204 No Content):**
No response body

**Error Responses:**

400 Bad Request - Invalid UUID:

```json
{
  "error": "invalid_id",
  "message": "Invalid customer ID format"
}
```

404 Not Found:

```json
{
  "error": "customer_not_found",
  "message": "Customer not found"
}
```

---

## Data Models

### Customer Object

```json
{
  "id": "uuid",
  "first_name": "string",
  "last_name": "string",
  "display_name": "string (computed)",
  "company_name": "string | null",
  "email": "string",
  "phone_primary": "string",
  "phone_secondary": "string | null",
  "billing_address_street": "string",
  "billing_address_city": "string",
  "billing_address_state": "string (2 chars)",
  "billing_address_zip": "string",
  "billing_address_full": "string (computed)",
  "service_address_street": "string | null",
  "service_address_city": "string | null",
  "service_address_state": "string (2 chars) | null",
  "service_address_zip": "string | null",
  "service_address_full": "string (computed)",
  "customer_type": "residential | commercial",
  "status": "active | inactive | prospect",
  "notes": "string | null",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Error Response

```json
{
  "error": "string (error code)",
  "message": "string (human-readable message)"
}
```

## Phone Number Formats

Accepted US phone formats:

- `(123) 456-7890`
- `123-456-7890`
- `1234567890`
- `+1-123-456-7890`

## ZIP Code Formats

Accepted US ZIP code formats:

- `12345`
- `12345-6789`

## State Codes

Must be valid 2-letter US state codes:

- All 50 states (AL, AK, AZ, AR, CA, CO, CT, DE, FL, GA, HI, ID, IL, IN, IA, KS, KY, LA, ME, MD, MA, MI, MN, MS, MO, MT, NE, NV, NH, NJ, NM, NY, NC, ND, OH, OK, OR, PA, RI, SC, SD, TN, TX, UT, VT, VA, WA, WV, WI, WY)
- DC (District of Columbia)
- Territories: AS, GU, MP, PR, VI

## Usage Examples

### cURL Examples

**Create Customer:**

```bash
curl -X POST http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john.doe@example.com",
    "phone_primary": "555-123-4567",
    "billing_address_street": "123 Main St",
    "billing_address_city": "Springfield",
    "billing_address_state": "IL",
    "billing_address_zip": "62701",
    "customer_type": "residential",
    "status": "active"
  }'
```

**List Customers with Filters:**

```bash
curl -X GET "http://localhost:8080/api/v1/customers?status=active&page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Update Customer:**

```bash
curl -X PUT http://localhost:8080/api/v1/customers/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "inactive",
    "notes": "Customer moved"
  }'
```

**Delete Customer:**

```bash
curl -X DELETE http://localhost:8080/api/v1/customers/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### JavaScript/Fetch Examples

```javascript
// Create customer
const createCustomer = async (customerData) => {
  const response = await fetch('http://localhost:8080/api/v1/customers', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(customerData),
  });
  return await response.json();
};

// Get customer
const getCustomer = async (customerId) => {
  const response = await fetch(
    `http://localhost:8080/api/v1/customers/${customerId}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  return await response.json();
};

// List customers with filters
const listCustomers = async (filters = {}) => {
  const params = new URLSearchParams(filters);
  const response = await fetch(
    `http://localhost:8080/api/v1/customers?${params}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  return await response.json();
};
```

## Database Validation

The database performs additional validation via triggers and constraints:

- Email addresses are automatically lowercased and trimmed
- State codes are automatically uppercased
- All text fields are automatically trimmed
- Service address must be all NULL or all filled (consistency check)

## Performance Notes

1. **Email lookups** are very fast due to unique index
2. **Name searches** are optimized with composite index
3. **Phone searches** use dedicated indexes
4. **Full-text search** uses the `/search` endpoint for best performance
5. **Geographic queries** (by state, city, ZIP) are indexed
6. **Pagination** is recommended for large result sets (max page_size is 100)

## Security Notes

1. All endpoints require valid JWT authentication
2. Permission checks are performed before any data access
3. Soft deletes prevent accidental data loss
4. Email uniqueness is enforced at database level
5. Input validation prevents SQL injection and XSS attacks
6. Rate limiting is applied at the middleware level

## Testing

Run handler tests:

```bash
go test ./internal/api/handlers -v -run TestCustomer
```

All 17 handler tests cover:

- ✅ Create customer (success, email exists, invalid request, repository error)
- ✅ Get customer by ID (success, not found, invalid ID)
- ✅ Get customer by email (success, not found)
- ✅ Update customer (success, not found)
- ✅ Delete customer (success, not found)
- ✅ List customers (success, with filters)
- ✅ Search customers (success, missing query)
