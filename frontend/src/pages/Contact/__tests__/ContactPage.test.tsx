import { describe, it, expect, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { ContactPage } from '../ContactPage';
import { SUPPORT_EMAIL, buildMailto } from '@/config/support';

describe('ContactPage', () => {
  const originalLocation = window.location;

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    });
  });

  it('opens a mailto link instead of faking a successful send', async () => {
    let href = '';
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...originalLocation,
        set href(value: string) {
          href = value;
        },
        get href() {
          return href;
        },
      },
    });
    const user = userEvent.setup({ delay: null });

    render(
      <MemoryRouter>
        <ContactPage />
      </MemoryRouter>
    );

    await user.type(screen.getByLabelText(/^name/i), 'Ada');
    await user.type(screen.getByLabelText(/^email/i), 'ada@example.com');
    await user.type(screen.getByLabelText(/^subject/i), 'Hello there');
    await user.type(screen.getByLabelText(/^message/i), 'Question');
    await user.click(
      screen.getByRole('button', { name: /open in email app/i })
    );

    expect(href.startsWith(`mailto:${SUPPORT_EMAIL}?`)).toBe(true);
    expect(href).toContain('subject=Hello%20there');
    expect(screen.queryByText(/thank you for your message/i)).toBeNull();
    expect(screen.getByRole('status')).toHaveTextContent(SUPPORT_EMAIL);
  });

  it('builds mailto URLs with encoded subject and body', () => {
    expect(buildMailto('a@b.c', { subject: 'Hi you', body: 'x&y' })).toBe(
      'mailto:a@b.c?subject=Hi%20you&body=x%26y'
    );
    expect(buildMailto('a@b.c')).toBe('mailto:a@b.c');
  });
});
