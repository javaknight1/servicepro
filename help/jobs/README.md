# Jobs & Scheduling

Schedule, assign, and manage service jobs.

## Guides

- [Creating Jobs](#creating-jobs)
- [Assigning Technicians](#assigning-technicians)
- [Managing Calendar](#managing-calendar)
- [Recurring Jobs](#recurring-jobs)

## Creating Jobs

1. Click **Jobs** > **+ New Job**
2. Select customer (or create new)
3. Fill in job details:

| Field       | Description                      |
| ----------- | -------------------------------- |
| Job Type    | Service Call, Installation, etc. |
| Description | Work to be performed             |
| Date & Time | When the job is scheduled        |
| Duration    | Expected time to complete        |
| Technician  | Assigned team member             |
| Priority    | Normal, High, Emergency          |

4. Click **Save Job**

## Job Statuses

| Status      | Meaning                      |
| ----------- | ---------------------------- |
| Scheduled   | On the calendar, not started |
| Dispatched  | Technician notified          |
| In Progress | Work underway                |
| Completed   | Job finished                 |
| Cancelled   | Job cancelled                |

## Assigning Technicians

### From Job Creation

Select technician in the job form dropdown.

### From Calendar

1. Open Calendar view
2. Drag job to technician's column
3. Confirm assignment

### Auto-Assignment

Enable in **Settings > Jobs > Auto-Dispatch** to automatically assign based on:

- Technician availability
- Location proximity
- Skills/certifications

## Managing Calendar

### Views

- **Day**: Hourly view of one day
- **Week**: 7-day overview
- **Month**: Monthly calendar
- **Technician**: Side-by-side technician schedules

### Actions

- **Drag & Drop**: Move jobs to new times/technicians
- **Click**: Open job details
- **Double-Click**: Quick edit

### Keyboard Shortcuts

| Action   | Shortcut     |
| -------- | ------------ |
| New Job  | Ctrl/Cmd + J |
| Today    | T            |
| Previous | Left Arrow   |
| Next     | Right Arrow  |

## Recurring Jobs

For regular service visits:

1. Create job as normal
2. Click **Make Recurring**
3. Set pattern:
   - Daily, Weekly, Monthly
   - Repeat interval
   - End date (or never)
4. Save

### Managing Recurring Jobs

- Edit single instance: Changes only that occurrence
- Edit series: Changes all future occurrences
- Skip instance: Mark as skipped
- Cancel series: Ends all future jobs

## Job Completion

When work is done:

1. Open job details
2. Add completion notes
3. Add photos (optional)
4. Click **Complete Job**
5. Create invoice (optional)

## Related

- [Creating Invoices](../billing/README.md)
- [Customer Management](../customers/README.md)
