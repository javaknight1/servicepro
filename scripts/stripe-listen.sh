#!/bin/sh
# Stripe CLI webhook forwarding script for local development

WEBHOOK_SECRET_FILE="/shared/stripe-webhook-secret"
BACKEND_URL="${STRIPE_FORWARD_URL:-http://backend:8080/api/v1/stripe/webhooks}"

echo "[stripe-cli] Starting Stripe webhook forwarding to: $BACKEND_URL"

# Check if API key is provided (support both STRIPE_SECRET_KEY and STRIPE_API_KEY)
API_KEY="${STRIPE_SECRET_KEY:-$STRIPE_API_KEY}"

if [ -z "$API_KEY" ]; then
    echo "[stripe-cli] ERROR: STRIPE_SECRET_KEY environment variable is required"
    echo "[stripe-cli] Set STRIPE_SECRET_KEY in your backend/.env file"
    sleep 10
    exit 1
fi

echo "[stripe-cli] API key configured (${#API_KEY} chars)"

# Get the webhook signing secret first (this just prints and exits)
echo "[stripe-cli] Fetching webhook signing secret..."
SECRET=$(stripe listen --api-key "$API_KEY" --print-secret 2>&1)

if echo "$SECRET" | grep -q "whsec_"; then
    # Extract just the secret
    SECRET=$(echo "$SECRET" | grep -o 'whsec_[a-zA-Z0-9_]*' | head -1)
    echo "$SECRET" > "$WEBHOOK_SECRET_FILE"
    echo "[stripe-cli] Webhook secret: $SECRET"
    echo "[stripe-cli] Saved to $WEBHOOK_SECRET_FILE"
else
    echo "[stripe-cli] WARNING: Could not get webhook secret"
    echo "[stripe-cli] Output was: $SECRET"
fi

# Now start the actual listener (this blocks and forwards webhooks)
echo "[stripe-cli] Starting webhook listener..."
exec stripe listen \
    --api-key "$API_KEY" \
    --forward-to "$BACKEND_URL"
