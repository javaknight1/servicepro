# ServicePro: Competitive Research & Positioning

Research date: 2026-10-16. All "Fetched" prices came from the vendor's own pricing page on 2026-10-16. Prices marked **[3rd-party]** come from review or aggregator sites and were not confirmed with the vendor. Prices marked **[unverified]** were contradictory or quote-only.

---

## 0. Where ServicePro stands today (from the repo)

**Built:** multi-tenant CRM (search, import/export), jobs and work orders with status history, a calendar with conflict detection, technician assignment, a recurrence engine (`backend/internal/services/recurring`), quotes with templates and a status machine, invoices with payment terms, PO number, T&Cs and PDFs, Stripe Checkout invoice payments, payment reminders by email and SMS (an SMS client exists), A/R aging, a "who owes what" report, revenue and customer reports, roles and permissions, business hours and timezone, outgoing webhooks, a PWA service worker, Sentry, PostHog and Grafana, and SaaS subscription plans/billing.

**Gaps that matter commercially:**
- **Invoice payments go through Stripe Checkout on the platform account.** There is no Stripe Connect, so tenants can't receive the money themselves. This blocks launch.
- **Missing workflow links:** quote→job (T028), job→invoice (T029/T030), a Technician role (T035), confirmations, reminders and on-my-way messages (T048–T050), deposits (T061/T063), e-signature (T081), and drag-and-drop scheduling (T027).
- **No QuickBooks sync, no customer portal, no online booking, no two-way SMS inbox, no tech mobile experience** (job photos, checklists, time clock), and no review requests.
- **One service address per customer.** There are no multi-property or site records, which matters for commercial and property-manager customers.

The back office and A/R side is unusually strong for a pre-launch product. The field side, the customer-facing side and the integrations are thin.

---

## 1. Competitors

### 1.1 Pricing summary

| Vendor | Segment | Pricing model | Current price (source / date) |
|---|---|---|---|
| **Jobber** | Residential home services, solo to 15 users (lawn, cleaning, HVAC, plumbing, handyman) | Tier + included users; **extra user $29/mo** | Core $49/mo (1 user, no commitment; $39 with 1-year commit; $29 billed annually). Connect $139 (1u) / $199 (5u) / $299 (10u) / $399 (15u). Grow $199 (1u) to $499 (15u). Plus $499 (5u) to $699 (15u). AI Receptionist add-on $29/mo. Pipeline add-on $49, Marketing Suite $99. — Fetched https://www.getjobber.com/pricing/ 2026-10-16 |
| **Housecall Pro** | Residential home services, 1–10+ techs | Tier + included users | Basic $79/mo monthly, $59 annual (1 user). Essentials $189 / $149 (5 users). Max $329 / $299 (8 users, +$35/user). Add-ons have no published price: CSR AI, Pipeline, Payroll, Voice, Tap to Pay and more. Card rate "as low as 2.59%". — Fetched https://www.housecallpro.com/pricing/ 2026-10-16 |
| **ServiceTitan** | Mid-market and enterprise residential and commercial HVAC/plumbing/electrical, roughly 10+ techs | Per technician, quote-only, annual contracts | **[3rd-party]** about $245–$500 per tech per month, plus $5k–$50k implementation (https://projul.com/blog/servicetitan-pricing-analysis-2026, https://www.getonecrew.com/post/servicetitan-pricing) |
| **Workiz** | Small and mid-size trades: locksmith, garage door, appliance, junk removal | Quote-only ("Request pricing") | Tiers are Standard, Pro and Ultimate, with no published prices. Genius Answering, Genius Phone and Genius Marketing are sold separately. — Fetched https://www.workiz.com/pricing/ 2026-10-16 |
| **FieldEdge** | HVAC/plumbing/electrical, 5–50 techs | Quote-only, per user | **[3rd-party / unverified]** about $100 per office user plus $125 per tech per month, plus a $500–$2k setup fee (https://www.fieldproxy.ai/fsm-software-pricing/fieldedge, https://costbench.com/software/field-service-management/fieldedge/negotiation/) |
| **ServiceM8** | Small trades. Strong in Australia, NZ and the UK, smaller in the US | **Per job volume, unlimited users** | Free $0 (30 jobs). Starter $29 (50 jobs). Growing $79 (150 jobs). Premium $149 (500 jobs). Premium Plus $349 (1,500+ jobs, then $0.20/job). Phone add-on from $19. — Fetched https://www.servicem8.com/us/pricing 2026-10-16 |
| **Service Fusion** | HVAC/plumbing/electrical, 5–30 techs | **Flat, unlimited users** | Starter $245 monthly / $208 annual. Plus $382 / $325. Pro $627 / $533. ServiceCall.ai and GPS are add-ons. — Fetched https://www.servicefusion.com/pricing 2026-10-16 |
| **Kickserv** | Budget small shops | Tier + user cap | Start $60 (5 users). Run $119 (10 users). Scale $199 (20 users). 20% off annual. QuickBooks Desktop add-on $50. "Kickback" 5% discount if online payment volume exceeds a threshold. — Fetched https://www.kickserv.com/pricing 2026-10-16 |
| **GorillaDesk** | Pest control, lawn, pool route businesses | Flat per route; **admin users unlimited** | Basic $49/mo ($539/yr). Pro $99 ($1,089/yr). Growth $149 ($1,639/yr, though trial links show $299, a discrepancy on the page). SMS line $5/mo plus credit packs. — Fetched https://gorilladesk.com/pricing/ 2026-10-16 |
| **FieldPulse** | HVAC/plumbing/electrical, 3–50 techs | Seat-based, quote-only | No published prices. Maverick AI is included. Operator AI and Engage phone are add-ons. — Fetched https://www.fieldpulse.com/pricing 2026-10-16 |
| **Zuper** | Mid-market: roofing, HVAC, commercial field service | Per user, annual | **[3rd-party / unverified]** $65–$105 per user per month (https://toolradar.com/tools/zuper/pricing). Other sources say from $35–$55. The vendor pricing page returned 404. |
| **Markate** | Owner-operators and small teams | Base fee + **$5 per employee** + à-la-carte add-ons | Owner Operator $49.95/mo ($45.95 annual). Team: base fee + $5 per employee. Add-ons cost $10/mo each (online booking, portal, review requests, phone number, etc.). Kate AI Receptionist $1 per call. — Fetched https://www.markate.com/pricing 2026-10-16 |
| **Joist** | Solo contractors and remodelers: estimates and invoices | Tiered, cheap | Basics $12/mo. Pro $17. Elite $32. Run $100 ($58.33 annual). AI Receptionist from $49/mo. — Fetched https://www.joist.com/pricing/ 2026-10-16 |
| **Invoice Simple** | Solo invoicing (not true FSM) | Tier by invoice count | **[partially verified, search snippet of the official page; the direct fetch returned 403]** Essentials $6.99/mo (3 invoices), Premium $21.99/mo (unlimited). Plus is about $14.99 per https://pricingsaas.com/companies/invoicesimple/diffs/2026Q2 |
| **Tradify** | Small trades, Australia/NZ/UK/US | **Per user** | Lite $47, Pro $51, Plus $61 per user per month (USD). The SmartTools AI is Plus only. — Fetched https://www.tradifyhq.com/pricing 2026-10-16 |
| **simPRO** | Mid-size commercial contractors: electrical, security, fire, HVAC | Per user, quote, multi-year contracts | **[3rd-party / unverified]** $60–$140 per user per month (https://fieldservicesoftware.io/software/simpro/). One review describes a push toward 3- to 5-year lock-in contracts. |
| **Fergus** | Plumbers and electricians in NZ, Australia and the UK, expanding into the US | Tier + per-user add-ons | Essentials $31/mo, Professional $44/mo (a "90% off until December" promotion shows $5.30/$7.50). Contractor users $2.50/day. Card processing 2.95% + $0.30. — Fetched https://www.fergus.com/us/pricing 2026-10-16 |
| **Commusoft** | UK-centric commercial HVAC, gas, plumbing | Quote | **[unverified]** No public price could be retrieved (the pricing URL returned 404). Rated 4.7/5 from about 209 reviews on Capterra. |
| **QuoteIQ** (newer, AI-heavy) | Exterior services: pressure washing, roofing, lawn, concrete | Tier + users + **AI credits** | Essentials $29.99 (1 user). Beginner $99.99 (2 users). Pro $199.99 (4 users). Elite $299.99 (10 users). Max $699.99 (20 users). +$25 per extra user. — Fetched https://myquoteiq.com/pricing/ 2026-10-16 |
| **Fieldproxy** (AI-native) | SMB to mid-market | Annual plan + $1 AI credits, unlimited users | Quote-only, with an implementation fee and no refunds. Its blog cites "from $649/mo" **[unverified]**. — Fetched https://www.fieldproxy.ai/pricing 2026-10-16 |
| **Vertical references** | Pool: **Skimmer** | Per serviced location, unlimited techs | $1 per location ($49 minimum), $2 per location (Scaling Up). AI Phone $99/mo. — Fetched https://www.getskimmer.com/pricing |
| | Residential cleaning: **ZenMaid** | Flat | Starter $19, Pro $39, Pro Max $49. — Fetched https://get.zenmaid.com/pricing |

### 1.2 AI-native and AI-layer entrants (2025–2026)

- **Avoca.** AI front office for HVAC, plumbing, roofing and electrical: 24/7 call answering, booking into ServiceTitan, Jobber or Housecall Pro, unsold-estimate follow-up, and CSR coaching. It raised more than $125M at a **$1B valuation** (Series B co-led by Meritech and General Catalyst, announced 2026-04-27) and has 800+ customers. Pricing is custom; third parties cite $1k+/mo. Sources: https://ai2.work/blog/avoca-hits-1b-valuation-with-125m-raise-to-bring-ai-to-home-services, https://www.withallo.com/blog/best-ai-receptionists-for-contractors (updated 2026-08-03).
- **Sameday AI.** AI answering that books into Service Fusion, ServiceTitan, Jobber, FieldRoutes and Housecall Pro. Launch $449/mo (500 min), Scale $789/mo **[3rd-party, Allo blog]**.
- **Rosie.** AI answering from $49/mo, with no native FSM integration **[3rd-party, Allo blog]**.
- **Netic.** Tagline "Autonomous AI revenue engine for essential home services." No details were verified.
- **Incumbent response:**
  - **ServiceTitan Max / Atlas.** An "agentic operating system" announced at Pantheon on 2026-10-06. It includes the Atlas mobile app for asking business questions, AI CSRs using Adaptive Capacity, and **Homh**, which makes contractors bookable through ChatGPT, Gemini and Claude (https://www.servicetitan.com/press/servicetitan-announcing-new-and-expanded-capabilities-at-pantheon-2026).
  - **Jobber Receptionist.** Generally available since August 2025, now $29/mo, and included in Plus. Jobber AI Voice and Chat are also live.
  - **Housecall Pro CSR AI** and "AI team members."
  - **Workiz Genius** suite.
  - **FieldPulse** Maverick and Operator AI.

**Takeaway:** an AI receptionist went from differentiator to a **$29/mo commodity add-on in about 12 months**. The VC-funded money is going into an AI layer that sits on top of existing FSMs rather than into new FSMs.

### 1.3 Standout features and complaints by competitor

Review sources are aggregators and some competitor blogs, flagged where relevant.

| Vendor | Standout features | Common complaints |
|---|---|---|
| **Jobber** | Best-in-class ease of use, client hub, online booking, a free website, 100+ integrations, quote upsell options, AI receptionist/voice/chat | **Costs rise fast with users and tiers.** Key features (QuickBooks, two-way SMS, automations) are gated behind higher tiers. Limited quote/invoice customization. **App doesn't work offline.** Limited control over automated messages. Weak job costing and reporting. Post-sale support. Source: https://www.getonecrew.com/post/jobber-reviews (2026-02-16; a competitor blog citing Capterra and Trustpilot). Price history tracker: Core went from $39 to $49 and Connect from $119 to $139 in 2026 (https://costbench.com/software/field-service-management/jobber/price-history/). |
| **Housecall Pro** | Consumer financing, price book, GPS, route optimization, a large add-on ecosystem (payroll, phones, dashcams), CSR AI | Rigid invoicing. Glitches. Calendar sync. Price for small teams. Weak keyword search. Trustpilot is about 3.0 versus Capterra 4.7 **[3rd-party aggregate, https://www.rfp.wiki/vendors/housecall-pro]**. Capterra: https://www.capterra.com/p/140363/housecall-pro/ |
| **ServiceTitan** | Deepest feature set: call booking, price books, memberships, marketing attribution, payroll, inventory, AI | **Long implementations (2–12 months)**, $5k–$50k setup, **annual or multi-year lock-in**, 15–20% renewal hikes, hard data export and cancellation, "too much for a small company." Sources: https://projul.com/blog/servicetitan-pricing-analysis-2026 (competitor blog), Capterra. |
| **Workiz** | Built-in phone, call tracking, Genius AI answering, strong for lead-driven trades | **Hidden pricing**, unauthorized annual renewal charges, hard cancellation and number porting, slow support, account-manager churn. Source: https://www.rfp.wiki/vendors/workiz, Trustpilot via search. |
| **ServiceM8** | Unlimited users, per-job pricing, cheap, strong iOS app, AI quote and invoice helpers | Historically **iOS-only**, no batch invoicing (requested since 2012), slowdowns after updates. Source: Capterra https://www.capterra.com/p/110711/ServiceM8/reviews/ (older reviews). |
| **Service Fusion** | Unlimited users at a flat price | **High entry price ($208+)**, and photos and e-sign are gated to Plus or Pro |
| **GorillaDesk** | Cheap route businesses, unlimited office users, subscription billing | Limited depth outside route-based trades **[no independent reviews retrieved]** |
| **Kickserv** | Cheap, generous user caps | Dated UX, no AI **[general reputation, unverified]** |
| **Tradify** | Simple job management with Xero/QuickBooks | Per-user pricing is expensive for crews |
| **simPRO** | Strong commercial and project features | Multi-year lock-ins and annual escalators **[single anecdote]** |
| **Markate** | Cheapest per-employee model ($5) | "Nickel-and-dime" à-la-carte add-ons at $10 each |

**Cross-cutting complaint themes (ranked):**
1. Price scales with headcount or tier gating.
2. Contracts, renewals and hard cancellation.
3. Slow or poor support after the sale.
4. Offline and mobile reliability.
5. Rigid invoice and quote customization.
6. Reporting and job-costing gaps.
7. Painful data migration and export.

---

## 2. Table stakes: minimum to be considered

Status key: ✅ have · 🟡 partial · ❌ missing. "S/M/L" is rough build size.

| # | Feature | ServicePro | Notes |
|---|---|---|---|
| 1 | Customer CRM with service and billing address, notes, history | ✅ | Timeline and comm-log still to do (T054/T055) |
| 2 | Quotes/estimates with line items, templates, PDF and email | ✅ | |
| 3 | Online quote approval by the customer | 🟡 | Status machine exists. Check for a public approve link. |
| 4 | Quote deposits | ❌ | T061/T063 (M) |
| 5 | E-signature on quote approval | ❌ | T081 (S, typed or drawn signature plus IP and timestamp) |
| 6 | Quote → Job → Invoice one-click conversion | ❌ | T028/T029/T030 (S). This is the core loop and must exist. |
| 7 | Scheduling calendar plus dispatch board, drag-and-drop | 🟡 | Calendar and conflicts exist. Drag-and-drop is T027 (M). |
| 8 | Recurring jobs / service agreements | 🟡 | Recurrence engine exists. Confirm UI and job generation are wired end to end. |
| 9 | **Online payments that go to the business** (card plus ACH) | ❌ | **Needs Stripe Connect (Express)**. Today money lands in the platform account. (M) |
| 10 | Tap-to-pay / in-person card | ❌ | Stripe Terminal Tap to Pay on iPhone/Android via the mobile app (M–L) |
| 11 | Automated payment reminders | ✅ | A strength |
| 12 | Appointment confirmations and reminders (email plus SMS) | ❌ | T048/T049 (S) |
| 13 | "On my way" text | ❌ | T050 (S) |
| 14 | Two-way SMS inbox (business number) | ❌ | Twilio plus 10DLC registration (M) |
| 15 | **Technician mobile experience**: today's jobs, map/directions, status updates, photos, notes, checklists, signature, collect payment | ❌ | PWA shell exists. A mobile-first tech view is needed (L). Offline is a known Jobber complaint, so an offline queue is a plus. |
| 16 | Job photos and attachments | ❌ | S3/R2 upload (S–M) |
| 17 | Job checklists / forms | ❌ | M |
| 18 | Time tracking / clock in-out per job | ❌ | M |
| 19 | Technician role with restricted view | ❌ | T035 (S) |
| 20 | Customer portal / client hub: view and approve quotes, pay invoices, request work, history | ❌ | M |
| 21 | Online booking / request form embeddable on a website | ❌ | M |
| 22 | QuickBooks Online sync (customers, invoices, payments) | ❌ | **Gate for most buyers** (M–L). Xero if targeting AU/UK. |
| 23 | Review requests (Google) after job or payment | ❌ | S |
| 24 | Map/route view of the day's jobs | ❌ | S–M (Mapbox/Google) |
| 25 | Basic reports: revenue, A/R aging, jobs | ✅ | A/R is a strength |
| 26 | Data import from CSV / other FSMs | 🟡 | Generic CSV import exists. Jobber and HCP export mappers are needed (S each). |
| 27 | Roles/permissions, multi-user | ✅ | |
| 28 | Native-feel mobile app (store-listed) | ❌ | Buyers search the App Store. A PWA wrapper such as Capacitor is acceptable (M). |

**Current table-stakes score:** about 6 ✅, 4 🟡, 18 ❌. Back-office depth is good. Field, customer-facing and payments-to-merchant are the gaps.

---

## 3. "Extremely competitive" extended list

What Housecall Pro Max, Jobber Plus and ServiceTitan offer, plus items that could beat them:

| Feature | Who has it | Value / notes |
|---|---|---|
| AI receptionist (calls and texts, books jobs) | Jobber ($29), HCP CSR AI, Workiz Genius, Markate ($1/call), Joist ($49), Skimmer ($99), Avoca/Sameday (standalone) | Now a commodity. Integrating with Avoca/Sameday-style vendors may beat building one. |
| AI quote builder from photos, voice or notes | Markate Kate Estimator, QuoteIQ, ServiceM8 AI Quote, Jobber AI Voice | Strong demo value. LLM plus price book makes it feasible for a solo developer. |
| Price book / flat-rate pricing, good-better-best options | HCP Essentials+, ServiceTitan, Workiz Ultimate, Jobber optional line items | T060 |
| Consumer financing (Wisetack, Sunbit) | HCP, Jobber, Skimmer, QuoteIQ | Wisetack: about 3.9% merchant fee, embeddable via partner API **[3rd-party]**. Matters for HVAC replacements. |
| Membership / maintenance plans with auto-billing | HCP Max, ServiceTitan, Workiz Ultimate | Recurring revenue for the contractor, so it is sticky |
| Marketing automation (campaigns, reactivation, review funnels) | Jobber Marketing Suite ($99), HCP Campaigns | |
| Pipeline / CRM sales stages | Jobber ($49 add-on), HCP Pipeline, GorillaDesk Growth | |
| GPS fleet / dashcams | HCP VGPS, Service Fusion | Partner, don't build |
| Route optimization | HCP Max, GorillaDesk, QuoteIQ Elite, Skimmer | Google Route Optimization API |
| Inventory, POs, truck stock | Service Fusion Plus, Workiz Ultimate, Tradify Plus | |
| Job costing / profit per job | Jobber Grow, HCP, Service Fusion | |
| Payroll / commissions | HCP add-on, ServiceTitan | Partner (Gusto) |
| Multi-location / franchise | Workiz Ultimate, Jobber partnerships, GorillaDesk multi-branch | |
| Built-in phone / VoIP with call tracking | Workiz, HCP Voice, ServiceM8 Phone, FieldPulse Engage | |
| AI "ask your business" analytics | ServiceTitan Atlas, Jobber AI Chat, Fergus Assistant, QuoteIQ "Ask AI Reports" | ServicePro already has Grafana views and clean SQL, so this is cheap to add |
| **MCP server / agent-accessible API** | QuoteIQ, Fieldproxy | Cheap for a solo developer, a forward-looking differentiator |
| Discoverable and bookable by AI assistants | ServiceTitan Homh (Oct 2026) | Watch this |
| AI collections agent | ServiceTitan "coming soon" | **ServicePro's natural wedge** (see below) |

---

## 4. Differentiation and moat

### 4.1 What will NOT be a moat (be honest)

- **AI features in general.** Jobber, HCP, ServiceTitan and Workiz all ship them, and the receptionist is a $29 add-on. AI is table stakes by 2027, not a moat.
- **Features.** Every feature listed above is copyable, and incumbents have 100+ engineers.
- **Low price alone.** Joist ($12), ServiceM8 (free), ZenMaid ($19) and Invoice Simple ($7) already exist below you. Low price wins switchers, it doesn't hold them.
- **Data network effects.** With fewer than 1,000 tenants you get no cross-customer learning advantage, and pooling tenant data raises trust issues. Don't plan on this.
- **Tech stack quality** (Go, observability). Customers can't see it.

### 4.2 What can realistically be defensible for a solo builder

1. **Switching costs, the real FSM moat.** Once customer history, recurring agreements, stored payment methods (Stripe Connect) and the QuickBooks mapping live in the system, churn drops sharply. Every FSM's moat is this. You get it by holding the **money flow** and **recurring schedule**, so prioritize Connect, saved cards/autopay and service agreements.
2. **Vertical depth in a niche the big players ignore.** Jobber and HCP are built around residential "pay at completion." Commercial and B2B service shops (net terms, POs, multi-site customers, property managers, compliance reports) are served by expensive mid-market tools like ServiceTitan, simPRO and Commusoft, or by spreadsheets. Niche-specific workflows, templates and compliance outputs are hard for horizontal tools to prioritize.
3. **Founder-led trust and positioning**: month-to-month, flat price, one-click export, "we'll migrate you for free." The top complaints are contracts, renewals, cancellation and support. A solo founder can credibly promise "no contract, your data leaves with one click, you email me directly." This is a brand moat that incumbents structurally can't copy without hurting their revenue.
4. **Embedded payments margin.** This is not a moat, but it is the business model that makes flat pricing viable. Add a 0.3–0.5% platform fee on Stripe Connect volume. Example: a 3-tech shop doing $40k/mo, 70% paid online, gives about $28k × 0.4% ≈ $112/mo, which is comparable to the subscription itself. Kickserv's "Kickback" shows vendors value this volume.

### 4.3 Beachhead recommendation

**Primary: small commercial and B2B service contractors who invoice on terms.** That means commercial cleaning/janitorial, facility and property-maintenance or handyman companies serving property managers, and small commercial HVAC/refrigeration shops (1–10 techs).

Why:
- **It fits what's already built.** A/R aging, "who owes what," payment terms, PO numbers, automated reminders and collections (T044, T064, T065) are exactly the pain of B2B net-30 billing. Residential-first tools (Jobber, HCP) treat these as afterthoughts.
- **Commercial customers need multi-site records, PO matching, consolidated monthly invoicing, approval portals for property managers, and service reports.** These are moderate features a solo developer can build, and horizontal residential tools do them poorly.
- **Higher LTV and lower churn.** Contracts are recurring, the books are larger, and these businesses are less price-shopping than solo residential operators.
- **Less AI-receptionist arms race.** Commercial work comes from relationships and recurring contracts, not inbound homeowner calls. You avoid competing head-on with the $1B Avoca and Jobber AI.

**Secondary (evaluate after 10 customers): compliance-recurring inspection trades**, such as backflow testing, fire extinguisher and sprinkler inspection, and kitchen hood cleaning. Recurring annual or semiannual schedules plus certificates or reports for authorities create strong lock-in, and the software options are fragmented and dated (no strong 2026 pricing data found, so **this is a hypothesis to validate by interviews**).

**Avoid as a beachhead:** residential HVAC/plumbing (ServiceTitan, HCP and FieldEdge are entrenched and AI-funded), lawn and residential cleaning (Jobber, ZenMaid at $19), and pool (Skimmer).

### 4.4 Positioning statement

> **For small commercial service contractors (1–10 techs) who bill businesses and property managers on terms, ServicePro is the field service platform that gets you paid. It handles scheduling, work orders, multi-site customers and automated collections in one flat monthly price with unlimited users and no contract. Unlike Jobber and Housecall Pro, which are built for homeowner pay-on-the-spot jobs and charge per seat, ServicePro is built around net-30 invoicing, POs and A/R, and you can leave with all your data any time.**

Tagline options: "Field service software that chases the money for you." / "Do the work. We'll collect."

### 4.5 Pricing recommendation

Flat, with unlimited users, no contract, and a published price. This directly attacks complaint #1.

| Plan | Price | Who | Includes |
|---|---|---|---|
| **Solo** | $39/mo ($32 annual) | Owner-operator | Everything core, 1 office user + 1 tech |
| **Team** | **$99/mo** ($84 annual) | 2–10 techs | **Unlimited users**, QuickBooks sync, SMS (fair-use allotment), portal, recurring agreements, collections automation |
| **Pro** | $199/mo ($169 annual) | 10–25 techs or multi-site-heavy | Multi-location customers, advanced reports, API/MCP, priority support, AI collections and AI quote drafting with a generous allotment |

- **Payments:** pass through Stripe rates plus a 0.4% platform fee. Lower ACH (for example 1%, capped at $10) to push B2B customers to ACH. Optional surcharge pass-through where legal.
- **Usage add-ons:** AI phone answering at cost plus margin, or partner with an AI receptionist vendor. Extra SMS per message.
- **Founding offer:** the first 25 customers lock $59/mo for life for Team, plus free white-glove migration.
- **Benchmark logic:** $99 unlimited-users undercuts Jobber Connect 5-user ($199), HCP Essentials ($189 for 5), Kickserv Run ($119 for 10) and Service Fusion Starter ($245). It sits near GorillaDesk Pro ($99) and ServiceM8 Growing ($79).

### 4.6 What the moat actually is (summary)

**Lock-in through money flow plus recurring schedules in an under-served B2B niche, reinforced by a trust brand (flat, no contract, easy exit) that per-seat incumbents can't match without cannibalizing themselves.** AI is an accelerator and a marketing hook, not the moat.

---

## 5. Prioritized roadmap

### MVP: launch to the first 10 paying customers

| # | Item | Size | Rationale |
|---|---|---|---|
| 1 | **Stripe Connect Express onboarding** plus invoice payments routed to the tenant, card and ACH, platform fee | M | Without it, no one can actually get paid. This blocks everything. |
| 2 | Quote→Job→Invoice conversion (T028, T029, T030) | S | The core loop every competitor has |
| 3 | Technician role (T035) plus mobile-first "My Day" PWA view: job list, directions link, status buttons, notes, **photos**, customer signature | L | Techs must use it in the field, or owners won't buy |
| 4 | Customer-facing quote approval page with e-sign (T081) plus optional deposit (T061/T063) | M | Gets quotes approved and secures deposits |
| 5 | Appointment confirmation, reminder and on-my-way email/SMS (T048–T050) | S | Cheapest visible "wow" for customers |
| 6 | QuickBooks Online one-way sync (customers, invoices, payments) | M–L | The #1 buying gate for B2B shops |
| 7 | Recurring jobs / service agreements UI wired to the recurrence engine | M | Commercial cleaning and maintenance are recurring by nature |
| 8 | Multi-site customers (a customer has many service locations, invoice per site or consolidated) | M | Beachhead-specific must-have |
| 9 | Import mappers for Jobber, HCP and generic CSV exports, plus a "we migrate you" service | S | Switching friction is the main barrier |
| 10 | Production deploy, billing for ServicePro plans (exists), Terms/Privacy, a support inbox | M | Launch hygiene |

### v1: competitive (customers 10–100)

| Item | Size | Rationale |
|---|---|---|
| Customer portal: view and pay invoices, approve quotes, request service, statement of account | M | Property managers expect self-serve, and it reduces A/R calls |
| Collections suite: dashboard (T044), promise-to-pay (T065), last-contact (T064), statements, late fees (T083) | M | Double down on the wedge |
| Two-way SMS inbox (Twilio plus 10DLC) | M | Expected by small shops |
| Drag-and-drop dispatch board plus map view of the day (T027) | M | Daily driver for office staff |
| Job checklists and service report PDF sent on completion | M | Commercial clients need proof of service |
| Time clock per job, plus timesheet export | M | Payroll and job costing basics |
| Review request automation | S | Cheap growth lever for customers |
| Online booking/request widget | M | Lead capture |
| Tap to Pay (Stripe Terminal) via a Capacitor-wrapped app in the stores | M–L | In-person payments and App Store presence |
| Consolidated monthly invoicing and PO matching | S–M | B2B-specific |
| Quote follow-up automation (T041, T042, T051) | S | Lifts win rates |

### v2: differentiators

| Item | Size | Rationale |
|---|---|---|
| **AI collections agent**: drafts and sends escalating reminders, reads replies ("paid," "send W-9," "dispute"), logs promise-to-pay, suggests calls | M | Aligned with the wedge. ServiceTitan only lists it as "coming soon." |
| AI quote drafting from photos, voice or notes against the tenant's price book | M | Matches Markate, QuoteIQ and Jobber, with strong demo appeal |
| "Ask your business" chat over the Grafana/SQL views, plus an **MCP server** so owners can use ServicePro from Claude or ChatGPT | S–M | Cheap given the existing data layer. A forward-looking differentiator. |
| Property-manager approval workflow (NTE limits, multi-approver, work-order intake from PM email) | M | Deep B2B moat feature |
| Compliance and inspection module (recurring inspections, certificates, authority reports) for the secondary beachhead | L | Opens backflow, fire and hood-cleaning verticals |
| Maintenance agreements with autopay plus renewal reminders | M | Recurring revenue for customers, which makes them sticky |
| Integrations rather than builds: AI receptionist partner (or Twilio voice plus an LLM), Wisetack financing, Gusto payroll, Zapier | M | Cover the "extremely competitive" checklist without building it all |
| Offline-capable tech app (queue and sync) | L | Fixes a known Jobber complaint, so it can be a marketing point |

---

## Sources (fetched 2026-10-16 unless noted)

- Jobber pricing: https://www.getjobber.com/pricing/
- Housecall Pro pricing: https://www.housecallpro.com/pricing/
- Workiz pricing: https://www.workiz.com/pricing/
- ServiceM8 pricing: https://www.servicem8.com/us/pricing
- FieldPulse pricing: https://www.fieldpulse.com/pricing
- GorillaDesk pricing: https://gorilladesk.com/pricing/
- Kickserv pricing: https://www.kickserv.com/pricing
- Service Fusion pricing: https://www.servicefusion.com/pricing
- Markate pricing: https://www.markate.com/pricing
- Joist pricing: https://www.joist.com/pricing/
- Tradify pricing: https://www.tradifyhq.com/pricing
- Fergus pricing: https://www.fergus.com/us/pricing
- QuoteIQ pricing: https://myquoteiq.com/pricing/
- Fieldproxy pricing: https://www.fieldproxy.ai/pricing
- Skimmer pricing: https://www.getskimmer.com/pricing
- ZenMaid pricing: https://get.zenmaid.com/pricing
- ServiceTitan Pantheon 2026 press release: https://www.servicetitan.com/press/servicetitan-announcing-new-and-expanded-capabilities-at-pantheon-2026
- Avoca funding: https://ai2.work/blog/avoca-hits-1b-valuation-with-125m-raise-to-bring-ai-to-home-services
- AI receptionist comparison (updated 2026-08-03): https://www.withallo.com/blog/best-ai-receptionists-for-contractors
- Jobber reviews summary (competitor blog, 2026-02-16): https://www.getonecrew.com/post/jobber-reviews
- Jobber price history: https://costbench.com/software/field-service-management/jobber/price-history/
- **[3rd-party]** ServiceTitan pricing: https://projul.com/blog/servicetitan-pricing-analysis-2026, https://www.getonecrew.com/post/servicetitan-pricing
- **[3rd-party]** FieldEdge pricing: https://www.fieldproxy.ai/fsm-software-pricing/fieldedge
- **[3rd-party]** Zuper pricing: https://toolradar.com/tools/zuper/pricing
- **[3rd-party]** simPRO pricing: https://fieldservicesoftware.io/software/simpro/
- **[3rd-party]** Invoice Simple price changes: https://pricingsaas.com/companies/invoicesimple/diffs/2026Q2
- **[3rd-party]** Housecall Pro and Workiz review aggregates: https://www.rfp.wiki/vendors/housecall-pro, https://www.rfp.wiki/vendors/workiz
- Capterra listings: Housecall Pro https://www.capterra.com/p/140363/housecall-pro/, ServiceM8 https://www.capterra.com/p/110711/ServiceM8/reviews/, Commusoft https://capterra.com/p/149651/Commusoft/
- **[3rd-party]** Wisetack fees: https://servicemag.org/software/wisetack, https://contractortoolstack.com/categories/financing-tools/
- **[3rd-party]** Jobber payment rates (2.9% + 30¢ card, 1% ACH): https://pipelineon.com/blog/jobber-pricing/
