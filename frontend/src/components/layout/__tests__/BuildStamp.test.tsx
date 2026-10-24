import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { UseQueryResult } from '@tanstack/react-query';
import { BuildStamp } from '../BuildStamp';
import { useVersion } from '@hooks/useVersion';
import type { VersionInfo } from '@services/versionService';
import { resolveVersionUrl } from '@config/endpoints';

vi.mock('@hooks/useVersion', () => ({
  useVersion: vi.fn(),
}));

function mockApi(data: VersionInfo | undefined) {
  vi.mocked(useVersion).mockReturnValue({
    data,
  } as unknown as UseQueryResult<VersionInfo>);
}

const SHA_A = 'a1b2c3d4e5f60718293a4b5c6d7e8f9012345678';
const SHA_B = 'f0e1d2c3b4a5968778695a4b3c2d1e0f98765432';

describe('BuildStamp', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubEnv('VITE_APP_VERSION', '');
    vi.stubEnv('VITE_GIT_SHA', '');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('renders nothing when neither version is known', () => {
    mockApi(undefined);
    const { container } = render(<BuildStamp />);
    expect(container).toBeEmptyDOMElement();
  });

  it("treats the API's dev/unknown defaults as unknown", () => {
    mockApi({ app_version: 'dev', git_sha: 'unknown' });
    const { container } = render(<BuildStamp />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows an em dash, never a bare V, for the unknown half', () => {
    mockApi({ app_version: '0.9.0', git_sha: SHA_A });
    render(<BuildStamp />);
    expect(screen.getByText('WEB —')).toBeInTheDocument();
    expect(screen.getByText('API V0.9.0')).toBeInTheDocument();
    expect(screen.queryByText(/V$/)).not.toBeInTheDocument();
  });

  it('shows both versions with short-SHA tooltips when they match', () => {
    vi.stubEnv('VITE_APP_VERSION', '0.9.0');
    vi.stubEnv('VITE_GIT_SHA', SHA_A);
    mockApi({ app_version: '0.9.0', git_sha: SHA_A });
    render(<BuildStamp />);

    const web = screen.getByText('WEB V0.9.0');
    const api = screen.getByText('API V0.9.0');
    expect(web).toHaveAttribute('title', 'Web build V0.9.0 (a1b2c3d)');
    expect(api).toHaveAttribute('title', 'API build V0.9.0 (a1b2c3d)');
    expect(api).not.toHaveAttribute('data-drift');
  });

  it('flags drift when the API serves a different version', () => {
    vi.stubEnv('VITE_APP_VERSION', '0.10.0');
    vi.stubEnv('VITE_GIT_SHA', SHA_B);
    mockApi({ app_version: '0.9.0', git_sha: SHA_A });
    render(<BuildStamp />);

    const api = screen.getByText('API V0.9.0');
    expect(screen.getByText('WEB V0.10.0')).toBeInTheDocument();
    expect(api).toHaveAttribute('data-drift', 'true');
    expect(api).toHaveAttribute(
      'title',
      'API is serving V0.9.0 (a1b2c3d), which differs from this web build V0.10.0'
    );
  });

  it('shows the web version alone when the API is unreachable', () => {
    vi.stubEnv('VITE_APP_VERSION', '0.9.0');
    mockApi(undefined);
    render(<BuildStamp />);
    expect(screen.getByText('WEB V0.9.0')).toBeInTheDocument();
    expect(screen.getByText('API —')).not.toHaveAttribute('data-drift');
  });
});

describe('resolveVersionUrl', () => {
  it.each([
    ['/api', '/version'],
    ['/api/', '/version'],
    ['http://localhost:8080/api', 'http://localhost:8080/version'],
  ])('%s -> %s', (base, expected) => {
    expect(resolveVersionUrl(base)).toBe(expected);
  });
});
