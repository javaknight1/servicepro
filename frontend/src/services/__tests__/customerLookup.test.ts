import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@services/customerService', () => ({
  customerService: { getCustomers: vi.fn() },
}));

import { customerService } from '@services/customerService';
import type { Customer } from '@services/customerService';
import { fetchAllCustomers } from '../customerLookup';

const makeCustomers = (start: number, count: number): Customer[] =>
  Array.from(
    { length: count },
    (_, i) => ({ id: `c-${start + i}` }) as unknown as Customer
  );

describe('fetchAllCustomers', () => {
  beforeEach(() => {
    vi.mocked(customerService.getCustomers).mockReset();
  });

  it('pages past the 100-customer cap', async () => {
    const total = 250;
    vi.mocked(customerService.getCustomers).mockImplementation(
      async (filters = {}) => {
        const page = filters.page ?? 1;
        const size = filters.page_size ?? 100;
        const start = (page - 1) * size;
        return {
          customers: makeCustomers(
            start,
            Math.max(0, Math.min(size, total - start))
          ),
          total,
          page,
          page_size: size,
        };
      }
    );

    const customers = await fetchAllCustomers();

    expect(customers).toHaveLength(250);
    expect(customers[249].id).toBe('c-249');
    expect(customerService.getCustomers).toHaveBeenCalledTimes(3);
    expect(customerService.getCustomers).toHaveBeenCalledWith(
      expect.objectContaining({ page: 3, page_size: 100 })
    );
  });

  it('makes a single request for small tenants', async () => {
    vi.mocked(customerService.getCustomers).mockResolvedValue({
      customers: makeCustomers(0, 3),
      total: 3,
      page: 1,
      page_size: 100,
    });

    expect(await fetchAllCustomers()).toHaveLength(3);
    expect(customerService.getCustomers).toHaveBeenCalledTimes(1);
  });
});
