import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../../services/quoteService', () => ({
  quoteService: {
    sendQuote: vi.fn(),
    acceptQuote: vi.fn(),
    rejectQuote: vi.fn(),
    downloadQuotePDF: vi.fn(),
  },
}));

import { quoteService } from '../../services/quoteService';
import { QuoteActionsMenu } from './QuoteActionsMenu';

describe('QuoteActionsMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reports the API's reason when accepting fails", async () => {
    const user = userEvent.setup();
    const onError = vi.fn();
    const onStatusChange = vi.fn();
    vi.mocked(quoteService.acceptQuote).mockRejectedValue({
      response: {
        status: 403,
        data: {
          error: 'cannot_accept',
          message: 'Quote cannot be accepted in current status',
        },
      },
    });

    render(
      <MemoryRouter>
        <QuoteActionsMenu
          quoteId="quote-1"
          status="sent"
          onStatusChange={onStatusChange}
          onError={onError}
        />
      </MemoryRouter>
    );

    await user.click(screen.getByRole('button', { name: 'Quote actions' }));
    await user.click(await screen.findByText('Accept Quote'));

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(
        'Quote cannot be accepted in current status'
      );
    });
    expect(onStatusChange).not.toHaveBeenCalled();
  });
});
