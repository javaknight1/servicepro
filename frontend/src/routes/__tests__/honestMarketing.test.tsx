import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { AppRoutes } from '../index';
import { useAuthStore } from '@store';
import { BETA_ACCESS_MAILTO } from '@/config/support';

// T255 / D11: the public site only says true things while ServicePro is an
// invite-only beta.
const FALSE_CLAIMS =
  /thousands|99\.9%|free trial|get started free|create free account|audit trails/i;

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AppRoutes />
    </MemoryRouter>
  );
}

describe('public marketing pages (T255)', () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: false, user: null });
  });

  it.each(['/', '/features', '/about'])(
    '%s makes no false claims and offers beta access',
    async (path) => {
      renderAt(path);
      const cta = await screen.findAllByRole(
        'link',
        { name: /request (beta )?access/i },
        { timeout: 3000 }
      );
      expect(cta[0]).toHaveAttribute('href', BETA_ACCESS_MAILTO);
      expect(document.body.textContent ?? '').not.toMatch(FALSE_CLAIMS);
      expect(screen.queryByRole('link', { name: /^pricing$/i })).toBeNull();
    }
  );

  it('redirects /pricing home while public pricing is hidden', async () => {
    renderAt('/pricing');
    const matches = await screen.findAllByText(
      /invite-only beta/i,
      {},
      { timeout: 3000 }
    );
    expect(matches.length).toBeGreaterThan(0);
  });

  it('says ServicePro is a working name', async () => {
    renderAt('/');
    expect(
      await screen.findByText(/working name/i, {}, { timeout: 3000 })
    ).toBeInTheDocument();
  });
});
