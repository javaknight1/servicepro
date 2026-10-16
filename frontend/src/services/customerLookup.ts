import { customerService } from '@services/customerService';
import type { Customer, CustomerFilters } from '@services/customerService';

/** The API caps page_size at 100. */
const PAGE_SIZE = 100;
/** Safety cap so a runaway total cannot loop forever (100 × 100 = 10k). */
const MAX_PAGES = 100;

/**
 * Load every customer for pickers (invoice/quote/job forms) by paging
 * through the list endpoint, instead of silently truncating at 100.
 */
export async function fetchAllCustomers(
  filters: Omit<CustomerFilters, 'page' | 'page_size'> = {}
): Promise<Customer[]> {
  const all: Customer[] = [];
  for (let page = 1; page <= MAX_PAGES; page++) {
    const response = await customerService.getCustomers({
      ...filters,
      page,
      page_size: PAGE_SIZE,
    });
    all.push(...response.customers);
    if (
      response.customers.length < PAGE_SIZE ||
      all.length >= (response.total ?? all.length)
    ) {
      break;
    }
  }
  return all;
}
