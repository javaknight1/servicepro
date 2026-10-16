-- ============================================================================
-- 000004_invite_codes (T106 / audit F-6)
--
-- Platform invite codes for REGISTRATION_MODE=invite_only. Codes are stored
-- as SHA-256 hashes (the plaintext is shown once, when the code is created
-- via POST /api/v1/admin/invite-codes or `go run ./cmd/invite create`).
-- max_uses = 0 means unlimited; expires_at NULL means no expiry.
-- ============================================================================

CREATE TABLE IF NOT EXISTS invite_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code_hash VARCHAR(64) NOT NULL UNIQUE,
    max_uses INTEGER NOT NULL DEFAULT 1 CHECK (max_uses >= 0),
    uses INTEGER NOT NULL DEFAULT 0 CHECK (uses >= 0),
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    note VARCHAR(255),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_invite_codes_created_at ON invite_codes(created_at DESC);
