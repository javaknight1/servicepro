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

Bring your customers (and their sites, jobs and unpaid invoices) over from
Jobber, Housecall Pro or any spreadsheet:

1. Go to **Customers** > **Import** (or **Settings** > **Import data**)
2. Pick where the file comes from: **Jobber**, **Housecall Pro** or **CSV or Excel**
3. Pick what it holds: customers first, then sites, jobs and open invoices
4. Upload the export (CSV or .xlsx, up to 10 MB and 50,000 rows)
5. Check the preview: the first rows as they will be saved, and any rows with
   problems. For a CSV from anywhere else, match its columns to ServicePro's fields
6. Click **Import** and wait for the result (you can leave the page; it keeps going)

Importing the same file again won't create duplicates. Customers need an email,
a US phone number and a full US billing address. Only invoices with a balance due
are imported, for that balance.

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
