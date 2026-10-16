package models

// QuoteStatus represents the possible states of a quote
type QuoteStatus string

const (
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusSent     QuoteStatus = "sent"
	QuoteStatusViewed   QuoteStatus = "viewed"
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusDeclined QuoteStatus = "declined"
	QuoteStatusExpired  QuoteStatus = "expired"
)

// QuoteStatusEvent represents an event type for quote status changes
type QuoteStatusEvent string

const (
	EventQuoteCreated  QuoteStatusEvent = "quote.created"
	EventQuoteSent     QuoteStatusEvent = "quote.sent"
	EventQuoteViewed   QuoteStatusEvent = "quote.viewed"
	EventQuoteAccepted QuoteStatusEvent = "quote.accepted"
	EventQuoteDeclined QuoteStatusEvent = "quote.declined"
	EventQuoteExpired  QuoteStatusEvent = "quote.expired"
	EventQuoteUpdated  QuoteStatusEvent = "quote.updated"
)
