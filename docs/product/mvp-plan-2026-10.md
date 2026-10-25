# ServicePro MVP plan (2026-10)

Decided 2026-10-25: the owner confirmed the app-upgrade docs drafted 2026-10-24. This file is the in-repo copy of
what the MVP is and why; **`TODO.md` is the source of truth for tasks** (MVP Launch Gate, decisions D7–D14, open
decisions). Full write-ups with mockups and sources live in Notion under Apps → All Apps → ServicePro:
MVP Proposal, Market & Competition, Technical Requirements, Decisions.

"MVP" = everything before paid launch (2027-09-07). "V1" = after launch, by checkpoint. "ServicePro" is a working
name (OD-10).

## The bet

**Re-scope, not restart.** The back office already works (customers, jobs, quotes, invoices, reminders, A/R aging,
refunds, multiple businesses per login). The MVP narrows it to one buyer and adds what that buyer can't get elsewhere
at this price, in about 292 hours.

- **Buyer (D7):** commercial cleaning and facility-maintenance companies with 1–10 techs that clean and maintain
  buildings for property managers and multi-site businesses, and bill them monthly on net terms.
- **Sample customer:** Brightline Facility Services, Columbus OH — 7 techs, an owner and an office manager, 23
  customers across 41 sites, $38,400/month in recurring agreements.
- **What it replaces:** spreadsheets or a residential field-service app, QuickBooks invoices built by hand, and a lot
  of copy-paste to bill one property manager for four buildings.
- **Goal:** five design partners each run a full month — recurring visits, proof of service, one consolidated invoice
  per customer, paid by bank transfer, synced to QuickBooks — with no manual fixes by 2027-06-30; paid launch
  2027-09-07.

## How it works

```
Service agreement (site, frequency, price)
  → visits scheduled by tech and site
  → tech completes checklist + photos, site contact signs
  → service report emailed to the property manager
  → month end: one invoice per customer (per-site lines, approved work orders, PO, Net 30)
  → paid by bank ($10 fee) or card → payout to the tenant's own Stripe account
  → invoice and payment synced to QuickBooks
  (late → reminders at 7/14/30 days, sent as the tenant, in daytime)
```

Extra work goes to the customer's named approver first; anything over the not-to-exceed limit can't be scheduled
without approval. Approved work lands on the same monthly invoice with its work-order number.

## Who uses it

| Who                           | Uses it to                                                                                   |
|-------------------------------|----------------------------------------------------------------------------------------------|
| Owner                         | Everything: pricing, agreements, payouts, plan, team                                         |
| Office manager                | Customers and sites, schedule, invoices, payments, reminders (not payouts or plan)           |
| Technician                    | Their own day on a phone: visits, checklists, photos, signatures. No prices or invoices      |
| Property manager (customer)   | Receives service reports and one monthly invoice; approves extra work; pays by bank. No login |
| Site contact                  | Signs the visit on the tech's phone                                                          |

## What happens to what's built

| Built feature                              | Verdict  | Task                    |
|--------------------------------------------|----------|-------------------------|
| Customers and contacts                     | Keep     | sites added (T186)      |
| Jobs and work orders                       | Simplify | T267                    |
| Calendar and conflict check                | Keep     | drag-and-drop later (T027) |
| Recurring jobs                             | Fix      | becomes service agreements (T167) |
| Quotes: send, revise, PDF, expiry          | Keep     | e-sign approval in V1 (T183) |
| Invoices and PDFs                          | Keep     | consolidated invoices (T188) |
| Online invoice payment                     | Fix      | off until Connect (T261, T169) |
| Refunds                                    | Keep     | —                       |
| Email payment reminders                    | Fix      | T265                    |
| SMS reminders                              | Hide     | T256 (OD-12)            |
| A/R aging, who owes what, revenue report   | Keep     | —                       |
| Customers report (segments, cohorts)       | Cut      | T266                    |
| CSV import                                 | Fix      | T241, T193              |
| Data export jobs                           | Hide     | T259                    |
| Export PDFs (wkhtmltopdf)                  | Cut      | T260 (D13)              |
| Roles page (5 fixed roles)                 | Simplify | Team page, 3 roles (T035) |
| Multiple businesses per login              | Keep     | —                       |
| Plans and billing (Free / $29 / $79)       | Fix      | Solo / Team / Pro (T210) |
| Public pricing page                        | Hide     | T255 until T210         |
| Landing, features, about pages             | Fix      | T255                    |
| Dashboard                                  | Fix      | T257                    |
| Account settings: password, notifications, 2FA | Hide | T259 until T158         |
| Business hours setting                     | Hide     | T259                    |
| Tax rates                                  | Simplify | flat per-tenant, set in onboarding (T235) |
| Payment-terms calculators                  | Simplify | keep Net terms; unused calculators go later |
| Outgoing webhooks                          | Cut      | T266                    |
| Offline app                                | Defer    | T179                    |

**New in the MVP:**

1. **Onboarding wizard** — your business, how you bill (terms, consolidated invoices, PO rule, tax), get paid (T235,
   T169).
2. **Customers & sites** — many sites per customer, each with its own agreement, contact and notes (T186).
3. **Service agreements** — recurring visits by site, frequency and price, generating the schedule (T167).
4. **Proof of service** — the technician's day on a phone, checklist, photos, signature, emailed report (T173, T174,
   T176).
5. **Consolidated monthly invoices with POs and approvals** — one invoice per customer with per-site lines (T187,
   T188).
6. **Get paid** — bank and card payments straight to the tenant, $10 bank fee, QuickBooks sync, Jobber/Housecall Pro
   import (T169, T170, T192, T193).

**Not in the MVP:** text messages, customer portal and online booking, e-signed quote approvals, deposits, late fees,
payroll, AI features, offline app, app-store apps.
**V1 (after launch, by checkpoint):** text reminders and on-my-way messages, property-manager approval inbox,
statements portal, quote approval page, custom roles.
**Later:** backflow-testing reports (first new niche), offline tech app, Zapier, payroll, financing, AI collections.

## Launch blockers

1. Customer payments land in the platform's Stripe account, not the tenant's — off until Connect (T261, T169; D9, D11).
2. The public site makes claims that aren't true (T255; D11).
3. Text reminders and file uploads silently go nowhere without a provider (T256).
4. The name collides with two live trademarks for field-service software (OD-10).
5. Dashboard shows $0 outstanding and 0 active jobs (T257).
6. Default sender addresses use `servicepro.com`, a domain owned by someone else (T258, T206).
7. Neon Free keeps only 6 hours of restore history; nightly backups aren't on (D14, T209).
8. Terms and Privacy are drafts (T207).
9. Business setup: legal entity, Hawaii GET license, employer approval (OD-11).

## Rules the product enforces

| Rule                              | What the product does                                                                                     |
|-----------------------------------|-----------------------------------------------------------------------------------------------------------|
| Bank-payment authorization        | Customers accept the ACH mandate on the pay page and get an emailed copy (Stripe, Nacha)                  |
| Card data stays out of our systems | Card and bank details only on Stripe's form (SAQ A)                                                      |
| Commercial email needs an opt-out | Invoices/reminders are transactional; follow-ups and review requests carry unsubscribe + postal address; opt-outs honoured on every send (T265) |
| Reminders come from the tenant    | "Tenant via ServicePro", tenant as reply-to, 8 a.m.–6 p.m. tenant-local (T265)                            |
| PO required                       | Customers marked "PO required" can't be invoiced without one (T187)                                       |
| Not-to-exceed limit               | Extra work over the limit can't be scheduled without the approver's OK (T187)                             |
| One customer, one bill a month    | A visit can't be billed twice (T188, T225)                                                                |
| No texts without consent          | SMS stays off until consent, STOP handling and 10DLC exist (OD-12)                                        |
| No card surcharging               | Not in the MVP (banned in CT, MA, ME; capped elsewhere)                                                   |

## Payments and pricing

- **Connect (D9):** Accounts v2, direct charges, Stripe collects its fees from the tenant; ServicePro takes an
  application fee and pays no Connect fees. Not Express, not destination charges.
- **Fees (D10):** cards Stripe's rate + 0.4%; ACH 1% capped at $10. Plans Solo $39 / Team $99 unlimited users / Pro
  $199; founding offer Team $59/mo for life for the first 25 tenants while payments stay on.
- Brightline example: about $310/month cheaper than a Jobber stack, plus 6–10 hours of month-end billing (estimate).

## Integrations

| Group                | MVP                                              | V1         | Later                                      |
|----------------------|--------------------------------------------------|------------|--------------------------------------------|
| Money & payments     | Stripe, QuickBooks Online                        | —          | Xero, Wisetack                             |
| Getting started      | Jobber import, Housecall Pro import (CSV built in) | —        | —                                          |
| Messages & calendar  | —                                                | Twilio SMS | Google Calendar, Outlook Calendar          |
| Team & workflow      | —                                                | —          | Gusto, Zapier, AppFolio export, CompanyCam |

## Stack for the MVP (D12)

Keep Gin + GORM; finish Neon Auth (T158), Resend only (T159), R2 (T160), RLS backstop (T154), stripe-go v87 (T264),
one PDF engine (gofpdf, D13). Frozen until the month-6 checkpoint: T153, T155, T156, T157, T162. The data model adds a
connected Stripe account, sites, job evidence and a QuickBooks link, and wires `recurring_schedules`; seven unused
tables are dropped by an additive migration (T263). Every new table gets `tenant_id NOT NULL`, a `(tenant_id, …)`
index and a cross-tenant test.

## Timeline and gates

Phases A–G and their tasks are in `TODO.md` → MVP Launch Gate.

| Date        | Gate                                                                                                |
|-------------|-----------------------------------------------------------------------------------------------------|
| 2026-11-08  | Honesty gate (D11): no false claims, payments off until they reach the tenant, nothing silently fails |
| 2027-01-31  | 10 interviews confirm the buyer and the $99 price, or the plan changes before phase D (T218)        |
| 2027-03-01  | New name chosen and cleared by an attorney (OD-10)                                                  |
| 2027-04-05  | Design-partner beta starts, 5 companies                                                             |
| 2027-06-30  | Beta gate: each partner runs a full month with no manual fixes; ≥3 take a bank payment              |
| 2027-09-07  | Paid launch with backups, monitoring, legal review and the support inbox in place                   |

**Success checkpoints (D8), from paid launch:** month 6 (2028-03-07) ≥10 paying tenants, ≥50% taking payments,
churn <5%/mo — else stop selling and keep it as a portfolio piece; month 12 (2028-09-07) ≥$3k MRR, churn <4%, one
repeatable channel; month 24 (2029-09-07) $5–10k MRR — keep running it or list it at about 2–3x revenue.

**Costs:** hosting $0 today; about $25–50/month from the first paying tenant (Neon Launch, Resend Pro); about $55–150
at 70 tenants (estimate). One-time: a domain for the new name plus attorney fees for trademark clearance.

## Assumptions to validate

| Assumption                                                     | Cheap test                                 | Kill criterion                                      |
|----------------------------------------------------------------|--------------------------------------------|-----------------------------------------------------|
| Commercial cleaners billing property managers feel multi-site billing pain | 10 interviews by 2027-01-31 (T218) | Fewer than 5 of 10 bill any customer for 3+ sites   |
| $99 flat with unlimited users is an easy yes                   | Ask for a $59 founding commitment          | Fewer than 3 of 10 would commit                     |
| Customers will pay large invoices by bank transfer             | Design partners' first month               | Fewer than 3 of 5 partners collect a bank payment   |
| One-way QuickBooks sync is enough at launch                    | Design partners' month-end close           | Partners still re-key more than a few invoices      |
| A solo builder can sell 4 hours a week                         | 20 outreach contacts a week, Oct–Nov 2027  | Fewer than 2 trials a month by 2027-11-30           |
| The narrow beachhead is big enough                             | Companies reached and replying             | Reply rate under 5% after 200 contacts              |

## Still open

- The new name (OD-10) and business setup (OD-11).
- Whether the ACH cap stays $10 or moves to $15 (about $30/month more per tenant) — D10 decided $10; revisit with
  interview data.
- Which tools a real Brightline-sized company uses today and how long month-end billing takes (T218).
