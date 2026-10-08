import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { I18nProvider } from '../i18n';
import { ToastProvider } from '../components/Toast';
import { api } from '../api/client';
import ExtrasPage from './ExtrasPage';

vi.mock('../context/AppContext', () => ({ useAppContext: () => ({ isProjectMode: true }) }));
vi.mock('../api/client', async (load) => {
  const actual = await load<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      ...actual.api,
      listExtras: vi.fn().mockResolvedValue({
        extras: [
          { name: 'team-rules', file: 'AGENTS.md', source_dir: '/p/.skillshare/extras/team-rules', source_type: 'per-extra', file_count: 1, source_exists: true,
            targets: [{ path: '.', mode: 'symlink', flatten: false, status: 'synced' }] },
          { name: 'rules', source_dir: '/p/.skillshare/extras/rules', source_type: 'per-extra', file_count: 2, source_exists: true,
            targets: [{ path: '.claude/rules', mode: 'merge', flatten: false, status: 'synced' }] },
          { name: 'conventions', file: 'CONVENTIONS.md', source_dir: '/p/.skillshare/extras/conventions', source_type: 'per-extra', file_count: 1, source_exists: true,
            targets: [{ path: '.cursor', mode: 'copy', flatten: false, as: 'rules.md', status: 'synced' }] },
          { name: 'pi-prompt', file: 'system.md', source_dir: String.raw`C:\Users\me\prompts`, source_type: 'custom', file_count: 1, source_exists: true,
            targets: [{ path: String.raw`C:\Users\me\.pi\agent`, mode: 'copy', flatten: false, as: 'APPEND_SYSTEM.md', status: 'synced' }] },
        ],
      }),
      createExtra: vi.fn().mockResolvedValue({ success: true }),
      previewExtraFilter: vi.fn().mockResolvedValue({ files: [{ file: 'index.md', status: 'synced' }, { file: 'draft.md', status: 'synced' }], unmatched: null }),
      editExtraTarget: vi.fn().mockResolvedValue({ success: true }),
      editExtra: vi.fn().mockResolvedValue({ success: true }),
      listExtraExtensions: vi.fn().mockResolvedValue({ extensions: [] }),
      availableTargets: vi.fn().mockResolvedValue({ targets: [] }),
      getOverview: vi.fn().mockResolvedValue({}),
    },
  };
});

const renderPage = (url = '/extras') => render(
  <QueryClientProvider client={new QueryClient()}>
    <I18nProvider><ToastProvider><MemoryRouter initialEntries={[url]}><ExtrasPage /></MemoryRouter></ToastProvider></I18nProvider>
  </QueryClientProvider>,
);

describe('Extras page in a project', () => {
  it('lists folder extras on the first tab', async () => {
    renderPage();
    expect(await screen.findByText('rules')).toBeInTheDocument();
  });

  // Shared AGENTS.md files are on the AGENTS.md tab.
  it('leaves AGENTS.md extras off the first tab', async () => {
    renderPage();
    await screen.findByText('rules');
    expect(screen.queryByText('team-rules')).not.toBeInTheDocument();
  });

  it('shows other single-file extras with the full target file path', async () => {
    renderPage();
    expect(await screen.findByText('conventions')).toBeInTheDocument();
    expect(screen.getByText('.cursor/rules.md')).toBeInTheDocument();
  });

  // A Windows folder keeps its backslashes up to the file name.
  it('joins a Windows target folder and file name with a backslash', async () => {
    renderPage();
    expect(await screen.findByText(String.raw`~\.pi\agent\APPEND_SYSTEM.md`)).toBeInTheDocument();
    // The full path is the row's tooltip.
    expect(screen.getByTitle(String.raw`C:\Users\me\.pi\agent\APPEND_SYSTEM.md`)).toBeInTheDocument();
  });

  it('creates a single-file extra with its file and target file name', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add extra' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Name'), 'notes');
    await user.click(within(dialog).getByRole('radio', { name: 'Single file' }));
    const [fileInput, asInput] = within(dialog).getAllByRole('textbox', { name: 'File name' });
    await user.type(fileInput, 'NOTES.md');
    await user.type(within(dialog).getByRole('textbox', { name: 'Folder' }), '.claude');
    await user.type(asInput, 'CLAUDE-notes.md');
    await user.click(within(dialog).getByRole('button', { name: 'Create' }));

    expect(api.createExtra).toHaveBeenCalledWith({
      name: 'notes',
      file: 'NOTES.md',
      targets: [{ path: '.claude', mode: 'merge', as: 'CLAUDE-notes.md' }],
    });
  });

  // Issue #300: a single file's name defaults to its file name without the extension.
  it('names a single-file extra after its file until a name is typed', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add extra' }));
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('radio', { name: 'Single file' }));
    await user.type(within(dialog).getAllByRole('textbox', { name: 'File name' })[0], 'APPEND_SYSTEM.md');
    expect(within(dialog).getByLabelText('Name')).toHaveValue('APPEND_SYSTEM');
    expect(within(dialog).getByText('Shown in the list. The file name is a good choice.')).toBeInTheDocument();
  });

  // Issue #300: several single files can share one folder of the shared extras folder.
  it('creates a single-file extra in a source folder named differently from the extra', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add extra' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Name'), 'review');
    await user.click(within(dialog).getByRole('radio', { name: 'Single file' }));
    const [fileInput] = within(dialog).getAllByRole('textbox', { name: 'File name' });
    await user.type(fileInput, 'review.md');
    await user.type(within(dialog).getByRole('textbox', { name: 'Source folder' }), 'prompts');
    await user.type(within(dialog).getByRole('textbox', { name: 'Folder' }), '.claude/commands');
    await user.click(within(dialog).getByRole('button', { name: 'Create' }));

    expect(api.createExtra).toHaveBeenCalledWith({
      name: 'review',
      folder: 'prompts',
      file: 'review.md',
      targets: [{ path: '.claude/commands', mode: 'merge' }],
    });
  });

  it('fills the source folder from a folder another single-file extra uses', async () => {
    vi.mocked(api.getOverview).mockResolvedValue({ extrasSource: '/p/.skillshare/extras' } as never);
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add extra' }));
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('radio', { name: 'Single file' }));
    await user.click(await within(dialog).findByRole('button', { name: 'conventions' }));
    expect(within(dialog).getByRole('textbox', { name: 'Source folder' })).toHaveValue('conventions');
    vi.mocked(api.getOverview).mockResolvedValue({} as never);
  });

  // Issue #435: a click on a preview row excludes that file from the target.
  it('saves a file excluded from a target in Edit target', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'More actions for .claude/rules' }));
    await user.click(screen.getByRole('menuitem', { name: 'Edit target…' }));
    const dialog = screen.getByRole('dialog');
    await user.click(await within(dialog).findByRole('button', { name: /draft\.md/ }));
    await user.click(within(dialog).getByRole('button', { name: 'Save' }));

    expect(api.editExtraTarget).toHaveBeenCalledWith('rules', '.claude/rules', {
      path: '.claude/rules', mode: 'merge', flatten: false, extension: '', as: '', include: [], exclude: ['draft.md'],
    });
  });

  it('renames an extra in Edit extra', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'More actions for rules' }));
    await user.click(screen.getByRole('menuitem', { name: 'Edit extra…' }));
    const name = within(screen.getByRole('dialog')).getByLabelText('Name');
    await user.clear(name);
    await user.type(name, 'docs');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(api.editExtra).toHaveBeenCalledWith('rules', { name: 'docs' });
  });

  // A target page's "Share with Extras" link.
  it('opens Add extra ready to share a target file', async () => {
    renderPage('/extras?add=file&target=%2Fhome%2Fme%2F.pi%2Fagent&file=APPEND_SYSTEM.md');
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByRole('radio', { name: 'Single file' })).toHaveAttribute('aria-checked', 'true');
    expect(within(dialog).getByLabelText('Name')).toHaveValue('APPEND_SYSTEM');
    expect(within(dialog).getAllByRole('textbox', { name: 'File name' })[0]).toHaveValue('APPEND_SYSTEM.md');
    expect(within(dialog).getByRole('textbox', { name: 'Folder' })).toHaveValue('/home/me/.pi/agent');
  });
});
