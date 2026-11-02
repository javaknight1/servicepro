import { describe, it, expect, vi } from 'vitest';

// The global test setup mocks this module; test the real implementation.
vi.unmock('../sentry');

const init = vi.fn();
vi.mock('@sentry/react', () => ({
  init: (options: unknown) => init(options),
  browserTracingIntegration: () => ({}),
  replayIntegration: () => ({}),
}));

// T262: visitors are not recorded routinely; replay only around an error.
describe('Sentry replay sampling', () => {
  it('records no sessions and every error session', async () => {
    const { initErrorTracking } = await import('../sentry');
    initErrorTracking({
      dsn: 'https://key@example.ingest.sentry.io/1',
      environment: 'test',
    });

    expect(init).toHaveBeenCalledTimes(1);
    expect(init.mock.calls[0][0]).toMatchObject({
      replaysSessionSampleRate: 0,
      replaysOnErrorSampleRate: 1.0,
    });
  });
});
