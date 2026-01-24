// Predefined avatar colors - visually distinct and accessible
const AVATAR_COLORS = [
  { bg: 'bg-red-100', text: 'text-red-700' },
  { bg: 'bg-orange-100', text: 'text-orange-700' },
  { bg: 'bg-amber-100', text: 'text-amber-700' },
  { bg: 'bg-lime-100', text: 'text-lime-700' },
  { bg: 'bg-green-100', text: 'text-green-700' },
  { bg: 'bg-teal-100', text: 'text-teal-700' },
  { bg: 'bg-cyan-100', text: 'text-cyan-700' },
  { bg: 'bg-blue-100', text: 'text-blue-700' },
  { bg: 'bg-indigo-100', text: 'text-indigo-700' },
  { bg: 'bg-violet-100', text: 'text-violet-700' },
  { bg: 'bg-purple-100', text: 'text-purple-700' },
  { bg: 'bg-pink-100', text: 'text-pink-700' },
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
 * Get initials from an email address
 * Takes the first two characters of the email (before @)
 */
export function getInitials(email: string): string {
  if (!email) return '??';

  const localPart = email.split('@')[0];
  if (!localPart) return '??';

  // Take first 2 characters and uppercase
  return localPart.slice(0, 2).toUpperCase();
}
