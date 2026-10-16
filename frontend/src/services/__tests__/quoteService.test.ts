import { describe, it, expect, vi, beforeEach } from 'vitest';
import { quoteService } from '../quoteService';
import api from '../api';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
  },
}));

const sentParams = (): URLSearchParams => {
  const config = vi.mocked(api.get).mock.calls[0][1] as {
    params: URLSearchParams;
  };
  return config.params;
};

describe('quoteService.getQuotes (T228)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.get).mockResolvedValue({
      data: { quotes: [], total: 0, page: 1, page_size: 10 },
    });
  });

  it('sends q, sort_by and sort_order', async () => {
    await quoteService.getQuotes({
      q: '  acme ',
      sort_by: 'total',
      sort_order: 'asc',
      page: 2,
      page_size: 10,
    });

    expect(api.get).toHaveBeenCalledWith('/v1/quotes', expect.anything());
    const params = sentParams();
    expect(params.get('q')).toBe('acme');
    expect(params.get('sort_by')).toBe('total');
    expect(params.get('sort_order')).toBe('asc');
    expect(params.get('page')).toBe('2');
    expect(params.has('search')).toBe(false);
  });

  it('omits a blank search', async () => {
    await quoteService.getQuotes({ q: '   ', page: 1 });
    expect(sentParams().has('q')).toBe(false);
  });
});
