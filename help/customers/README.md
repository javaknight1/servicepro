# Managing Customers

Create, edit, and manage customer records.

## Guides

- [Creating Customers](#creating-customers)
- [Editing Customers](#editing-customers)
- [Search & Filters](#search-and-filters)
- [Importing Customers](#importing-customers)

## Creating Customers

1. Click **Customers** > **+ Add Customer**
2. Fill in required fields:

   - Customer Name
   - Type (Commercial/Residential)
   - Primary Contact
   - Email and Phone
   - Service Address

3. Optional fields:

   - Billing Address (if different)
   - Notes
   - Custom Fields
   - Tax Exempt Status

4. Click **Save Customer**

## Editing Customers

1. Find the customer (search or browse)
2. Click customer name to open profile
3. Click **Edit**
4. Update fields as needed
5. Click **Save Changes**

## Search and Filters

### Quick Search

Use `Ctrl/Cmd + K` to open global search, type customer name.

### Advanced Filters

- **Status**: Active, Inactive
- **Type**: Commercial, Residential
- **Location**: City, State, ZIP
- **Created Date**: Date range

## Importing Customers

Bulk import from CSV:

1. Go to **Customers** > **Import**
2. Download the template CSV
3. Fill in customer data
4. Upload completed CSV
5. Map columns to fields
6. Review and confirm import

### CSV Format

```
name,type,contact_name,email,phone,address,city,state,zip
ABC Corp,commercial,John Smith,john@abc.com,555-1234,123 Main St,Austin,TX,78701
```

## Customer Profile

Each customer profile shows:

| Tab      | Content                       |
| -------- | ----------------------------- |
| Overview | Contact info, recent activity |
| Jobs     | Job history, scheduled jobs   |
| Invoices | Invoice history, balance      |
| Notes    | Internal notes, attachments   |

## Related

- [Scheduling Jobs](../jobs/README.md)
- [Creating Invoices](../billing/README.md)
