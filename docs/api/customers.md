# Customers API

REST API for customer management with full CRUD operations, search, and filtering.

## Overview

The Customers API provides:

- Complete customer lifecycle management
- Residential and commercial customer types
- Separate billing and service addresses
- Advanced search and filtering
- Permission-based access control

## Endpoints

| Method | Endpoint                  | Description      | Permission       |
| ------ | ------------------------- | ---------------- | ---------------- |
| GET    | `/customers`              | List customers   | customers.list   |
| GET    | `/customers/:id`          | Get customer     | customers.read   |
| GET    | `/customers/email/:email` | Get by email     | customers.read   |
| GET    | `/customers/search`       | Search customers | customers.list   |
| POST   | `/customers`              | Create customer  | customers.create |
| PUT    | `/customers/:id`          | Update customer  | customers.update |
| DELETE | `/customers/:id`          | Delete customer  | customers.delete |

## List Customers

Get a paginated list of customers with filters.

```http
GET /api/v1/customers
```

### Query Parameters

| Parameter  | Type    | Description                                  |
| ---------- | ------- | -------------------------------------------- |
| page       | integer | Page number (default: 1)                     |
| page_size  | integer | Items per page (default: 20, max: 100)       |
| type       | string  | Filter by type: residential, commercial      |
| status     | string  | Filter by status: active, inactive, prospect |
| state      | string  | Filter by state code (e.g., CA)              |
| city       | string  | Filter by city                               |
| sort_by    | string  | Sort field (default: created_at)             |
| sort_order | string  | asc or desc (default: desc)                  |

### Response

```json
{
  "customers": [
    {
      "id": "uuid",
      "first_name": "John",
      "last_name": "Doe",
      "email": "john.doe@example.com",
      "phone_primary": "555-123-4567",
      "customer_type": "residential",
      "status": "active",
      "billing_address_city": "Springfield",
      "billing_address_state": "IL",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

## Get Customer

Get a single customer by ID.

```http
GET /api/v1/customers/:id
```

### Response

```json
{
  "id": "uuid",
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@example.com",
  "phone_primary": "555-123-4567",
  "phone_secondary": "555-987-6543",
  "company_name": "Doe Enterprises",
  "customer_type": "commercial",
  "status": "active",
  "billing_address_street": "123 Main St",
  "billing_address_city": "Springfield",
  "billing_address_state": "IL",
  "billing_address_zip": "62701",
  "service_address_street": "456 Oak Ave",
  "service_address_city": "Springfield",
  "service_address_state": "IL",
  "service_address_zip": "62702",
  "notes": "Preferred contact method: email",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

## Get Customer by Email

Look up a customer by their email address.

```http
GET /api/v1/customers/email/:email
```

## Search Customers

Quick search across name, email, company, and phone.

```http
GET /api/v1/customers/search?q=john
```

### Query Parameters

| Parameter | Type   | Required | Description                     |
| --------- | ------ | -------- | ------------------------------- |
| q         | string | Yes      | Search query (min 2 characters) |

### Response

Returns an array of matching customers.

## Create Customer

Create a new customer.

```http
POST /api/v1/customers
```

### Request Body

```json
{
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@example.com",
  "phone_primary": "555-123-4567",
  "phone_secondary": "555-987-6543",
  "company_name": "Doe Enterprises",
  "customer_type": "commercial",
  "status": "active",
  "billing_address_street": "123 Main St",
  "billing_address_city": "Springfield",
  "billing_address_state": "IL",
  "billing_address_zip": "62701",
  "service_address_street": "456 Oak Ave",
  "service_address_city": "Springfield",
  "service_address_state": "IL",
  "service_address_zip": "62702",
  "notes": "Preferred contact method: email"
}
```

### Required Fields

- `first_name` (1-100 characters)
- `last_name` (1-100 characters)
- `email` (valid email format)
- `phone_primary` (valid phone format)
- `billing_address_street`
- `billing_address_city`
- `billing_address_state` (2-letter code)
- `billing_address_zip` (5 or 9 digit)

### Customer Types

- `residential` - Individual homeowner
- `commercial` - Business customer

### Customer Status

- `active` - Active customer
- `inactive` - Inactive customer
- `prospect` - Potential customer

### Response (201 Created)

Returns the created customer object.

## Update Customer

Update an existing customer.

```http
PUT /api/v1/customers/:id
```

### Request Body

All fields are optional. Only include fields to update.

```json
{
  "phone_primary": "555-111-2222",
  "status": "inactive",
  "notes": "Updated notes"
}
```

### Response

Returns the updated customer object.

## Delete Customer

Soft delete a customer (sets `deleted_at` timestamp).

```http
DELETE /api/v1/customers/:id
```

### Response

```
HTTP 204 No Content
```

## Validation Rules

### Email

- Must be valid email format
- Must be unique across all customers

### Phone Numbers

- Accepts multiple formats: (555) 123-4567, 555-123-4567, 5551234567
- Minimum 10 digits

### State Codes

- Must be valid 2-letter US state/territory code
- Automatically converted to uppercase

### ZIP Codes

- 5-digit format: 12345
- ZIP+4 format: 12345-6789

### Service Address

- If any service address field is provided, all are required
- Must be complete or empty (no partial addresses)

## Error Responses

### Validation Error (400)

```json
{
  "error": "validation_error",
  "message": "Validation failed",
  "fields": {
    "email": "invalid email format",
    "billing_address_zip": "invalid ZIP code format"
  }
}
```

### Email Already Exists (409)

```json
{
  "error": "email_already_exists",
  "message": "A customer with this email already exists"
}
```

### Customer Not Found (404)

```json
{
  "error": "customer_not_found",
  "message": "Customer not found"
}
```

## Examples

### Create Residential Customer

```bash
curl -X POST http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jane",
    "last_name": "Smith",
    "email": "jane.smith@example.com",
    "phone_primary": "555-987-6543",
    "customer_type": "residential",
    "status": "active",
    "billing_address_street": "789 Elm St",
    "billing_address_city": "Chicago",
    "billing_address_state": "IL",
    "billing_address_zip": "60601"
  }'
```

### List Active Commercial Customers

```bash
curl "http://localhost:8080/api/v1/customers?type=commercial&status=active&page=1" \
  -H "Authorization: Bearer <token>"
```

### Search Customers

```bash
curl "http://localhost:8080/api/v1/customers/search?q=smith" \
  -H "Authorization: Bearer <token>"
```

## Database Schema

### customers table

| Column             | Type         | Description                |
| ------------------ | ------------ | -------------------------- |
| id                 | UUID         | Primary key                |
| first_name         | VARCHAR(100) | Customer first name        |
| last_name          | VARCHAR(100) | Customer last name         |
| email              | VARCHAR(255) | Unique email address       |
| phone_primary      | VARCHAR(20)  | Primary phone              |
| phone_secondary    | VARCHAR(20)  | Secondary phone            |
| company_name       | VARCHAR(200) | Company name (commercial)  |
| customer_type      | ENUM         | residential, commercial    |
| status             | ENUM         | active, inactive, prospect |
| billing*address*\* | VARCHAR      | Billing address fields     |
| service*address*\* | VARCHAR      | Service address fields     |
| notes              | TEXT         | Customer notes             |
| created_at         | TIMESTAMP    | Creation timestamp         |
| updated_at         | TIMESTAMP    | Last update timestamp      |
| deleted_at         | TIMESTAMP    | Soft delete timestamp      |

### Indexes

- Unique index on email
- Index on customer_type
- Index on status
- Composite index on state, city
- Full-text search index on name, email, company
