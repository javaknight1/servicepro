// Avatar tints from the MVP mockups: warm orange, teal and grey, soft
// backgrounds with darker initials.
const AVATAR_COLORS = [
  { bg: 'bg-primary-50', text: 'text-primary-600' },
  { bg: 'bg-secondary-50', text: 'text-secondary-700' },
  { bg: 'bg-neutral-100', text: 'text-neutral-600' },
];

/**
 * Simple hash function to convert a string to a number
 */
function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
}

/**
 * Get a deterministic color for an avatar based on email
 * The same email will always get the same color
 */
export function getAvatarColor(email: string): { bg: string; text: string } {
  const hash = hashString(email.toLowerCase());
  const index = hash % AVATAR_COLORS.length;
  return AVATAR_COLORS[index];
}

/**
 * Get initials from first/last name or email address
 * Priority:
 * 1. First name + Last name → First letter of each (e.g., "JD" for John Doe)
 * 2. Only first name → First 2 letters (e.g., "JO" for John)
 * 3. Only last name → First 2 letters (e.g., "DO" for Doe)
 * 4. No name → First 2 letters of email (before @)
 */
export function getInitials(
  email: string,
  firstName?: string | null,
  lastName?: string | null
): string {
  const first = firstName?.trim();
  const last = lastName?.trim();

  // Both first and last name
  if (first && last) {
    return (first[0] + last[0]).toUpperCase();
  }

  // Only first name - take first 2 letters
  if (first) {
    return first.slice(0, 2).toUpperCase();
  }

  // Only last name - take first 2 letters
  if (last) {
    return last.slice(0, 2).toUpperCase();
  }

  // Fallback to email
  if (!email) return '??';

  const localPart = email.split('@')[0];
  if (!localPart) return '??';

  // Take first 2 characters and uppercase
  return localPart.slice(0, 2).toUpperCase();
}

/**
 * Get display name from first/last name or email
 * Priority:
 * 1. First name + Last name → "First Last"
 * 2. Only first name → "First"
 * 3. Only last name → "Last"
 * 4. No name → email
 */
export function getDisplayName(
  email: string,
  firstName?: string | null,
  lastName?: string | null
): string {
  const first = firstName?.trim();
  const last = lastName?.trim();

  if (first && last) {
    return `${first} ${last}`;
  }

  if (first) {
    return first;
  }

  if (last) {
    return last;
  }

  return email;
}
