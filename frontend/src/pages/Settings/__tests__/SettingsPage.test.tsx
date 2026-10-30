import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock('@components/settings/ProfilePictureSection', () => ({
  ProfilePictureSection: () => <div />,
}));

const authState = {
  user: {
    id: 'user-1',
    email: 'owner@example.com',
    first_name: 'Pat',
    last_name: 'Lee',
  },
  setUser: vi.fn(),
  fetchCurrentUser: vi.fn(),
};

vi.mock('@store', () => ({
  useAuthStore: <T,>(selector: (state: typeof authState) => T): T =>
    selector(authState),
}));

vi.mock('@services/userService', () => ({
  userService: { updateProfile: vi.fn() },
}));

import { SettingsPage } from '../SettingsPage';

// T259: password, 2FA and notification-preference controls did nothing, so
// they are hidden until Neon Auth supplies them (T158). Profile still works.
describe('SettingsPage', () => {
  it('shows the profile form and no dead security or notification controls', () => {
    render(<SettingsPage />);

    expect(screen.getByLabelText('First Name')).toHaveValue('Pat');
    expect(
      screen.getByRole('button', { name: /save changes/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole('button', { name: /security/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /notifications/i })
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/password/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/two-factor/i)).not.toBeInTheDocument();
  });
});
