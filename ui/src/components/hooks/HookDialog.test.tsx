import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { I18nProvider } from '../../i18n';
import { hooksApi } from '../../api/hooks';
import type { HookCandidate, HookInventory } from '../../api/hooks';
import HookDialog from './HookDialog';
import HooksImportDialog from './HooksImportDialog';
import HooksRemoveDialog from './HooksRemoveDialog';

vi.mock('../CodeEditor', () => ({
  default: ({ value, onChange, ariaLabel }: { value: string; onChange: (v: string) => void; ariaLabel: string }) => <textarea aria-label={ariaLabel} value={value} onChange={(e) => onChange(e.target.value)} />,
}));
vi.mock('../../api/hooks', async (load) => ({
  ...await load<typeof import('../../api/hooks')>(),
  hooksApi: { catalog: vi.fn(), preview: vi.fn(), save: vi.fn(), configure: vi.fn(), import: vi.fn() },
}));

const catalog = { claude: { timeoutUnit: 'seconds' as const, events: [
  { name: 'PostToolUse', description: 'After a tool succeeds', matcher: true },
  { name: 'Stop', description: 'When the main agent finishes', matcher: false },
] } };

const wrap = (ui: React.ReactNode) => render(<MemoryRouter><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><I18nProvider>{ui}</I18nProvider></QueryClientProvider></MemoryRouter>);

const openClaude = async (user: ReturnType<typeof userEvent.setup>) => {
  wrap(<HookDialog existingNames={[]} onClose={vi.fn()} onSaved={vi.fn()} />);
  await user.type(screen.getByLabelText('Name'), 'lint');
  await user.click(screen.getByRole('checkbox', { name: /Claude/ }));
};

describe('Git raw YAML dialog', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(hooksApi.catalog).mockResolvedValue(catalog);
    vi.mocked(hooksApi.save).mockResolvedValue({ applied: [], backupIds: [] });
  });
  it('edits and saves commands and files together without Agent event fields', async () => {
    const user = userEvent.setup();
    const binding = { commands: { 'tool.check': { events: ['pre-commit'], command: '{files}/check.sh', parallel: false } }, files: { 'check.sh': '#!/bin/sh\nexit 0\n' } };
    wrap(<HookDialog initial={{ name: 'guard', entry: { bindings: { git: binding } } }} existingNames={['guard']} onClose={vi.fn()} onSaved={vi.fn()} />);
    const editor = screen.getByLabelText('Git Binding YAML');
    expect((editor as HTMLTextAreaElement).value).toContain('commands:');
    expect(screen.queryByLabelText('Event 1')).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Fields' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(hooksApi.save).toHaveBeenCalledWith({ name: 'guard', entry: { bindings: { git: binding } } }));
  });
});

describe('hook dialog', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(hooksApi.catalog).mockResolvedValue(catalog);
    // jsdom has no scrollIntoView, which the dropdown calls on its focused option.
    HTMLElement.prototype.scrollIntoView = vi.fn();
  });

  it('keeps invalid JSON and says why when switching back to fields', async () => {
    const user = userEvent.setup();
    await openClaude(user);
    await user.click(screen.getByRole('tab', { name: 'Native JSON' }));
    const editor = screen.getByLabelText('Claude Native JSON');
    const broken = '{\n  "Stop": [\n    { "hooks": [] }\n    { "hooks": [] }\n  ]\n}';
    await user.clear(editor);
    await user.click(editor);
    await user.paste(broken);
    expect(screen.getByRole('tab', { name: /Native JSON/ })).toContainElement(screen.getByLabelText('1 errors'));
    await user.click(screen.getByRole('tab', { name: 'Fields' }));
    expect(screen.getByRole('alert')).toHaveTextContent('Fix the JSON before switching to fields. Line 4:');
    expect(screen.getByLabelText('Claude Native JSON')).toHaveValue(broken);
  });
});

describe('hooks import', () => {
  beforeEach(() => vi.resetAllMocks());

  const candidate = (name: string, command: string): HookCandidate => ({
    name, problems: [], warnings: [],
    entry: { bindings: { claude: { events: { Stop: [{ hooks: [{ type: 'command', command }] }] } } } },
  });
  const data = {
    source: { path: '/s.yaml', configPath: '/s.yaml', entries: {} }, targets: [], backups: [], plan: null, previewError: '',
    paths: { claude: '/home/u/.claude/settings.json' },
    unmanaged: [{ target: 'claude', path: '/home/u/.claude/settings.json', names: ['notify', 'format'] }],
  } as unknown as HookInventory;

  it('saves only the checked candidates, one hook each, taking over their registrations', async () => {
    const user = userEvent.setup();
    vi.mocked(hooksApi.import).mockResolvedValue([candidate('notify', './notify.sh'), candidate('format', './format.sh')]);
    vi.mocked(hooksApi.save).mockResolvedValue({ applied: [], backupIds: [] });
    const onImported = vi.fn();
    wrap(<HooksImportDialog data={data} onClose={vi.fn()} onImported={onImported} />);
    const group = await screen.findByRole('region', { name: 'Claude' });
    expect(within(group).getByTitle('/home/u/.claude/settings.json')).toBeInTheDocument();
    await user.click(within(group).getByRole('checkbox', { name: 'format' }));
    expect(screen.getByText('1 selected')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Import 1' }));
    await waitFor(() => expect(onImported).toHaveBeenCalledWith(1));
    expect(hooksApi.import).toHaveBeenCalledWith({ from: 'claude' });
    expect(hooksApi.save).toHaveBeenCalledTimes(1);
    expect(hooksApi.save).toHaveBeenCalledWith(expect.objectContaining({ entry: candidate('notify', './notify.sh').entry, adopt: true }));
  });

  it("keeps an Antigravity block's own name, since the block is the hook", async () => {
    const user = userEvent.setup();
    const block: HookCandidate = { name: 'lint', problems: [], warnings: [], entry: { bindings: { antigravity: { events: { Stop: [{ command: './run-lint.sh' }] } } } } };
    vi.mocked(hooksApi.import).mockResolvedValue([block]);
    vi.mocked(hooksApi.save).mockResolvedValue({ applied: [], backupIds: [] });
    const agy = { ...data, paths: { antigravity: '/home/u/.gemini/config/hooks.json' }, unmanaged: [{ target: 'antigravity', path: '/home/u/.gemini/config/hooks.json', names: ['lint'] }] } as unknown as HookInventory;
    wrap(<HooksImportDialog data={agy} onClose={vi.fn()} onImported={vi.fn()} />);
    const name = await screen.findByRole('textbox', { name: 'Name for lint' });
    expect(name).toHaveValue('lint');
    expect(name).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Import 1' }));
    await waitFor(() => expect(hooksApi.save).toHaveBeenCalledWith(expect.objectContaining({ name: 'lint', adopt: true })));
  });
});

describe('hooks remove', () => {
  beforeEach(() => vi.resetAllMocks());

  it('stops managing without syncing, and says the source-only choice still deletes on the next sync', async () => {
    const user = userEvent.setup();
    const path = '/home/u/.claude/settings.json';
    vi.mocked(hooksApi.preview).mockImplementation(async (m) => ({ revision: m?.unmanage ? 'kept' : 'r', fingerprint: 'fp', sourcePath: '/s.yaml', blocked: false, changes: m?.unmanage ? [] : [{ target: 'claude', path, name: 'guard', action: 'remove' }] }));
    vi.mocked(hooksApi.configure).mockResolvedValue({ applied: [], backupIds: [] });
    const onSaved = vi.fn();
    wrap(<HooksRemoveDialog name="guard" onClose={vi.fn()} onSaved={onSaved} />);
    const sourceOnly = screen.getByRole('button', { name: 'Remove from source only' });
    await waitFor(() => expect(sourceOnly).toHaveAccessibleDescription('Leaves the files for now, but the next sync also deletes it from the Claude settings files.'));
    await user.hover(sourceOnly);
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Leaves the files for now, but the next sync also deletes it from the Claude settings files.');
    await user.click(screen.getByRole('button', { name: 'Stop managing' }));
    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(true));
    expect(hooksApi.configure).toHaveBeenCalledWith({ name: 'guard', remove: true, unmanage: true }, 'kept', false);
  });

});

describe('account hook bindings', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(hooksApi.catalog).mockResolvedValue({ codex: { timeoutUnit: 'seconds', events: [{ name: 'SessionStart', description: 'When a session starts', matcher: false }] } });
    vi.mocked(hooksApi.save).mockResolvedValue({ applied: [], backupIds: [] });
    HTMLElement.prototype.scrollIntoView = vi.fn();
  });

  it('offers the account with its Agent catalog and saves under the account key', async () => {
    const user = userEvent.setup();
    wrap(<HookDialog accounts={{ 'codex-2': 'codex' }} existingNames={[]} onClose={vi.fn()} onSaved={vi.fn()} />);
    await user.type(screen.getByLabelText('Name'), 'account');
    await user.click(screen.getByRole('checkbox', { name: 'codex-2 (Codex)' }));
    await user.click(screen.getByRole('combobox', { name: 'Event 1' }));
    await user.click(await screen.findByRole('option', { name: /^SessionStart/ }));
    await user.type(screen.getByLabelText('Command 1'), 'echo account');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(hooksApi.save).toHaveBeenCalledWith({ name: 'account', entry: { bindings: { 'codex-2': { events: { SessionStart: [{ hooks: [{ type: 'command', command: 'echo account' }] }] } } } } }));
  });

});
