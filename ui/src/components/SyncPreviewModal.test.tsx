import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { api, type Target } from '../api/client';
import { I18nProvider } from '../i18n';
import SyncPreviewModal from './SyncPreviewModal';

vi.mock('../api/client', async (load) => ({ ...await load<typeof import('../api/client')>(), api: { diff: vi.fn(), listTargets: vi.fn(), sync: vi.fn() } }));

const targets = [{ name: 'claude', path: '/c/skills', mode: 'merge', agentPath: '/c/agents' }, { name: 'codex', path: '/x/skills', mode: 'merge' }] as Target[];
const show = (kind: 'skill' | 'agent') => render(<MemoryRouter><QueryClientProvider client={new QueryClient()}><I18nProvider><SyncPreviewModal open onClose={vi.fn()} kind={kind} /></I18nProvider></QueryClientProvider></MemoryRouter>);

describe('Sync preview modal', () => {
  // Refs: #289. A dry-run counts existing links again; the diff counts only what sync would write.
  it('counts only the pending changes of the given kind', async () => {
    vi.mocked(api.listTargets).mockResolvedValue({ targets } as never);
    vi.mocked(api.diff).mockResolvedValue({ diffs: [
      { target: 'claude', items: [{ skill: 'pdf', action: 'link' }, { skill: 'git', action: 'link' }, { skill: 'reviewer.md', action: 'link', kind: 'agent' }] },
      { target: 'codex', items: [] },
    ] } as never);
    show('skill');
    expect(await screen.findByText('2 linked')).toBeInTheDocument();
    expect(screen.getByText('1 target already up to date')).toBeInTheDocument();
    expect(api.sync).not.toHaveBeenCalled();
  });

  it('says agents are up to date when only skills are pending', async () => {
    vi.mocked(api.listTargets).mockResolvedValue({ targets } as never);
    vi.mocked(api.diff).mockResolvedValue({ diffs: [{ target: 'claude', items: [{ skill: 'pdf', action: 'link' }] }] } as never);
    show('agent');
    expect(screen.getByText('Only agents are written. Skills, extras and MCP stay as they are.')).toBeInTheDocument();
    expect(await screen.findByText('Everything is up to date. No sync needed.')).toBeInTheDocument();
  });

  it('explains an include filter that selects no skill after syncing', async () => {
    vi.mocked(api.listTargets).mockResolvedValue({ targets } as never);
    vi.mocked(api.diff).mockResolvedValue({ diffs: [{ target: 'claude', items: [{ skill: 'pdf', action: 'link' }] }] } as never);
    vi.mocked(api.sync).mockResolvedValue({ results: [], warnings: [], unmatched: [{ target: 'codex', patterns: ['*review*'], all: true }] } as never);
    const user = userEvent.setup();
    show('skill');
    await user.click(await screen.findByRole('button', { name: 'Sync Now' }));
    expect(await screen.findByRole('link', { name: 'Edit filter' })).toHaveAttribute('href', '/targets/codex');
  });

  it('opens with focus on Cancel, not the secondary Sync page link', () => {
    vi.mocked(api.listTargets).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.diff).mockReturnValue(new Promise(() => {}));
    show('skill');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });
});
