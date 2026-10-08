import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import type { Target } from '../api/client';
import { ToastProvider } from '../components/Toast';
import { I18nProvider } from '../i18n';
import TargetDetailPage from './TargetDetailPage';

vi.mock('../api/client', async (load) => {
  const actual = await load<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      ...actual.api,
      listTargets: vi.fn(), getTargetInstructions: vi.fn(), listTargetFiles: vi.fn(),
      previewSyncMatrix: vi.fn(), listExtraExtensions: vi.fn(),
    },
  };
});
vi.mock('../hooks/useSharedQueries', () => {
  const idle = () => ({ data: undefined });
  return { useAvailableTargetsQuery: idle, useHooksQuery: idle, useMcpQuery: idle };
});

const never = () => new Promise<never>(() => {});

function renderPage(mode: string, targetNaming: string) {
  const target = { name: 'claude', path: '/home/dev/.claude/skills', mode, targetNaming, skillsEnabled: true, linkedCount: 0, localCount: 0, include: [], exclude: [] } as unknown as Target;
  vi.mocked(api.listTargets).mockResolvedValue({ targets: [target] } as Awaited<ReturnType<typeof api.listTargets>>);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={['/targets/claude']}>
      <QueryClientProvider client={client}><I18nProvider><ToastProvider>
        <Routes><Route path="/targets/:name" element={<TargetDetailPage />} /></Routes>
      </ToastProvider></I18nProvider></QueryClientProvider>
    </MemoryRouter>,
  );
}

const needsCopy = 'prefixed needs copy mode, because it rewrites the name inside each copy.';

describe('TargetDetailPage naming', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const fn of [api.getTargetInstructions, api.listTargetFiles, api.previewSyncMatrix, api.listExtraExtensions]) vi.mocked(fn).mockImplementation(never);
  });

  it('keeps prefixed visible but disabled outside copy mode, and says why', async () => {
    renderPage('merge', 'flat');

    const prefixed = await screen.findByRole('button', { name: 'prefixed' });
    expect(prefixed).toBeDisabled();
    expect(prefixed).toHaveAttribute('title', 'Needs copy mode');
    expect(screen.getByText(needsCopy)).toBeInTheDocument();
  });

  it('enables prefixed in copy mode and marks the unsaved naming as pending', async () => {
    const user = userEvent.setup();
    renderPage('copy', 'flat');

    const prefixed = await screen.findByRole('button', { name: 'prefixed' });
    expect(prefixed).toBeEnabled();
    expect(screen.queryByText(needsCopy)).not.toBeInTheDocument();
    expect(screen.queryByText('pending')).not.toBeInTheDocument();

    await user.click(prefixed);
    expect(prefixed).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('pending')).toBeInTheDocument();
  });

  it('puts the saved naming back when leaving copy mode with prefixed drafted', async () => {
    const user = userEvent.setup();
    renderPage('copy', 'standard');

    await user.click(await screen.findByRole('button', { name: 'prefixed' }));
    await user.click(screen.getByRole('radio', { name: /merge/ }));

    expect(screen.getByRole('button', { name: 'standard' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.queryByText('pending')).not.toBeInTheDocument();
  });

  it('falls back to flat when leaving copy mode with prefixed already saved', async () => {
    const user = userEvent.setup();
    renderPage('copy', 'prefixed');

    await user.click(await screen.findByRole('radio', { name: /merge/ }));

    expect(screen.getByRole('button', { name: 'flat' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('pending')).toBeInTheDocument();
  });
});
