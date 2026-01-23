# Billing & Payments

Create invoices, quotes, and process payments.

## Guides

- [Creating Quotes](#creating-quotes)
- [Creating Invoices](#creating-invoices)
- [Sending Invoices](#sending-invoices)
- [Recording Payments](#recording-payments)

## Creating Quotes

1. Click **Quotes** > **+ New Quote**
2. Select customer
3. Add line items:
   - Description
   - Quantity
   - Unit Price
4. Add notes/terms
5. Click **Save Quote**

### Quote Statuses

| Status   | Meaning               |
| -------- | --------------------- |
| Draft    | Not sent, in progress |
| Sent     | Sent to customer      |
| Accepted | Customer approved     |
| Declined | Customer declined     |
| Expired  | Past expiration date  |

### Converting to Job

1. Open accepted quote
2. Click **Convert to Job**
3. Schedule date/time
4. Assign technician
5. Confirm

## Creating Invoices

### From a Job

1. Open completed job
2. Click **Create Invoice**
3. Line items auto-populate
4. Review and adjust
5. Save or Send

### Standalone Invoice

1. Click **Invoices** > **+ New Invoice**
2. Select customer
3. Add line items
4. Set due date
5. Save or Send

### Invoice Fields

| Field      | Description              |
| ---------- | ------------------------ |
| Invoice #  | Auto-generated           |
| Customer   | Bill-to customer         |
| Line Items | Services/products        |
| Tax        | Calculated automatically |
| Due Date   | Payment deadline         |
| Notes      | Terms, instructions      |

## Sending Invoices

### Email Invoice

1. Open invoice
2. Click **Send**
3. Review email preview
4. Click **Send Invoice**

### Download PDF

1. Open invoice
2. Click **Download PDF**
3. Print or save

## Invoice Statuses

| Status  | Meaning                  |
| ------- | ------------------------ |
| Draft   | Not sent                 |
| Sent    | Emailed to customer      |
| Viewed  | Customer opened          |
| Paid    | Full payment received    |
| Partial | Partial payment received |
| Overdue | Past due date            |

## Recording Payments

### Manual Payment

1. Open invoice
2. Click **Record Payment**
3. Enter:
   - Amount
   - Payment method (Cash, Check, Card)
   - Date received
   - Reference/check number
4. Click **Save Payment**

### Online Payments

With Stripe integration:

1. Customer clicks **Pay Now** in email
2. Enters card details
3. Payment processes automatically
4. Invoice marked as Paid

## Payment Methods

| Method        | Setup               |
| ------------- | ------------------- |
| Cash          | No setup needed     |
| Check         | Record check number |
| Credit Card   | Requires Stripe     |
| Bank Transfer | Record reference    |

## Overdue Invoices

### Automatic Reminders

Enable in **Settings > Invoices > Reminders**:

- 3 days before due
- On due date
- 7 days overdue
- 14 days overdue

### Manual Follow-up

1. Go to **Invoices** > Filter by **Overdue**
2. Click invoice
3. Click **Send Reminder**

## Related

- [Customer Management](../customers/README.md)
- [Job Scheduling](../jobs/README.md)
