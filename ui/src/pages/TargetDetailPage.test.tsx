import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api, type Target } from '../api/client';
import { ToastProvider } from '../components/Toast';
import { I18nProvider } from '../i18n';
import TargetDetailPage from './TargetDetailPage';

const offline = vi.hoisted(() => () => Promise.reject(new Error('offline')));
vi.mock('../api/client', async (load) => ({
  ...await load<typeof import('../api/client')>(),
  api: {
    listTargets: vi.fn(), previewSyncMatrix: vi.fn(() => Promise.resolve({ entries: [] })), availableTargets: vi.fn(offline),
    getTargetInstructions: vi.fn(offline), listTargetFiles: vi.fn(offline), listExtraExtensions: vi.fn(offline), updateTarget: vi.fn(),
  },
}));
vi.mock('../api/mcp', async (load) => ({ ...await load<typeof import('../api/mcp')>(), mcpApi: { list: vi.fn(offline) } }));
vi.mock('../api/hooks', async (load) => ({ ...await load<typeof import('../api/hooks')>(), hooksApi: { list: vi.fn(offline) } }));

const target = (name: string, over: Partial<Target> = {}) => ({
  name, path: '/home/me/.agents/skills', mode: 'merge', targetNaming: 'flat', status: 'merged', linkedCount: 3, localCount: 0,
  include: [], exclude: [], expectedSkillCount: 3, skillsEnabled: true, ...over,
}) as Target;

function renderPage(codex: Partial<Target>, universal: Partial<Target> = {}) {
  vi.mocked(api.listTargets).mockResolvedValue({
    targets: [target('codex', { skillsSharedWith: ['universal'], ...codex }), target('universal', { skillsSharedWith: ['codex'], ...universal })],
    sourceSkillCount: 3,
  });
  render(
    <MemoryRouter initialEntries={['/targets/codex']}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <I18nProvider><ToastProvider><Routes><Route path="/targets/:name" element={<TargetDetailPage />} /></Routes></ToastProvider></I18nProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('Target page shared skills folder', () => {
  beforeEach(() => vi.clearAllMocks());

  it('warns under the mode picker and sets the draft to the other target\'s mode', async () => {
    const user = userEvent.setup();
    renderPage({ mode: 'copy' });
    expect(await screen.findByText(/universal also syncs skills to .*\.agents\/skills, with merge\. With copy, each sync would undo the other\./)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Use merge like universal' }));
    expect(screen.queryByText(/each sync would undo the other/)).toBeNull();
  });

  it('warns under the naming control, and only once the modes agree', async () => {
    const user = userEvent.setup();
    renderPage({ targetNaming: 'standard' });
    expect(await screen.findByText(/universal also syncs skills to .*\.agents\/skills, with flat naming\. With standard, each sync would undo the other\./)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Use flat like universal' }));
    expect(screen.queryByText(/each sync would undo the other/)).toBeNull();
  });

  it('warns under the filters when only they differ, and copies the other target\'s filters', async () => {
    const user = userEvent.setup();
    renderPage({ exclude: ['feature-radar*'] });
    expect(await screen.findByText(/universal also syncs skills to .*\.agents\/skills, with different filters\. Each sync would add back or remove what the other filters out\./)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Use the same filters as universal' }));
    expect(screen.queryByText(/with different filters/)).toBeNull();
  });

  it('stays quiet when both resolve to symlink, whatever the naming', async () => {
    renderPage({ mode: 'symlink', targetNaming: 'standard' }, { mode: 'symlink' });
    await screen.findByText('Sync mode');
    expect(screen.queryByText(/each sync would undo the other/)).toBeNull();
  });
});
