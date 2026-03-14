import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

function emptyResponse(status = 204) {
	return new Response(null, { status, statusText: 'No Content' });
}

const sampleBackup = {
	id: 1,
	filename: 'backup-2026-03-13-120000.zip',
	status: 'completed',
	trigger: 'manual',
	destination: 'local',
	size_bytes: 1048576,
	file_count: 42,
	db_migration_version: 15,
	duration_ms: 2300,
	created_at: '2026-03-13T12:00:00Z',
	completed_at: '2026-03-13T12:00:02Z'
};

const runningBackup = {
	id: 2,
	filename: 'backup-2026-03-13-140000.zip',
	status: 'running',
	trigger: 'manual',
	destination: 'local',
	size_bytes: 0,
	file_count: 0,
	db_migration_version: 15,
	duration_ms: 0,
	created_at: '2026-03-13T14:00:00Z'
};

const failedBackup = {
	id: 3,
	filename: 'backup-2026-03-13-100000.zip',
	status: 'failed',
	trigger: 'scheduled',
	destination: 'local',
	size_bytes: 0,
	file_count: 0,
	db_migration_version: 15,
	duration_ms: 500,
	error_message: 'Disk full',
	created_at: '2026-03-13T10:00:00Z'
};

const sampleStatus = {
	is_running: false,
	last_backup: sampleBackup,
	next_scheduled: '2026-03-14T03:00:00Z'
};

const idleStatusNoBackups = {
	is_running: false,
	last_backup: null,
	next_scheduled: ''
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Backup settings page', () => {
	it('shows loading spinner initially', () => {
		mockFetch.mockReturnValue(new Promise(() => {}));
		render(Page);

		const spinner = document.querySelector('[role="status"]');
		expect(spinner).toBeInTheDocument();
	});

	it('loads and displays backup history', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([sampleBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('backup-2026-03-13-120000.zip')).toBeInTheDocument();
		});

		expect(screen.getByText('1.0 MB')).toBeInTheDocument();
		expect(screen.getByText('2.3 s')).toBeInTheDocument();
		const badges = screen.getAllByText('Dokonceno');
		expect(badges.length).toBeGreaterThanOrEqual(1);
	});

	it('shows empty state when no backups exist', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([]))
			.mockResolvedValueOnce(jsonResponse(idleStatusNoBackups));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zadne zalohy')).toBeInTheDocument();
		});
	});

	it('shows status card with correct info', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([sampleBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Necinna')).toBeInTheDocument();
		});

		expect(screen.getByText('Posledni zaloha')).toBeInTheDocument();
		expect(screen.getByText('Dalsi planovana')).toBeInTheDocument();
	});

	it('shows running status when backup is in progress', async () => {
		const runningStatus = {
			is_running: true,
			last_backup: runningBackup,
			next_scheduled: ''
		};

		mockFetch
			.mockResolvedValueOnce(jsonResponse([runningBackup]))
			.mockResolvedValueOnce(jsonResponse(runningStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Probiha zalohovani')).toBeInTheDocument();
		});
	});

	it('shows error message for failed backups', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([failedBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Chyba')).toBeInTheDocument();
		});

		expect(screen.getByText('Disk full')).toBeInTheDocument();
	});

	it('create backup button triggers API call', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([]))
			.mockResolvedValueOnce(jsonResponse(idleStatusNoBackups));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vytvorit zalohu')).toBeInTheDocument();
		});

		// Mock the create call + subsequent reload
		mockFetch
			.mockResolvedValueOnce(jsonResponse(sampleBackup))
			.mockResolvedValueOnce(jsonResponse([sampleBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		await fireEvent.click(screen.getByText('Vytvorit zalohu'));

		await waitFor(() => {
			const calls = mockFetch.mock.calls;
			const createCall = calls.find(
				(c: unknown[]) =>
					(c[0] as string).includes('/api/v1/backups') && (c[1] as RequestInit)?.method === 'POST'
			);
			expect(createCall).toBeTruthy();
		});
	});

	it('delete backup shows confirmation dialog and calls API', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([sampleBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('backup-2026-03-13-120000.zip')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Smazat'));

		await waitFor(() => {
			expect(screen.getByText('Smazat zalohu')).toBeInTheDocument();
			expect(
				screen.getByText('Opravdu chcete smazat tuto zalohu? Soubor bude trvale odstranen.')
			).toBeInTheDocument();
		});

		// Mock the delete call + subsequent reload
		mockFetch
			.mockResolvedValueOnce(emptyResponse())
			.mockResolvedValueOnce(jsonResponse([]))
			.mockResolvedValueOnce(jsonResponse(idleStatusNoBackups));

		// Find and click the confirm button inside the dialog
		const confirmButtons = screen.getAllByText('Smazat');
		const dialogConfirmBtn = confirmButtons[confirmButtons.length - 1];
		await fireEvent.click(dialogConfirmBtn);

		await waitFor(() => {
			const calls = mockFetch.mock.calls;
			const deleteCall = calls.find(
				(c: unknown[]) =>
					(c[0] as string).includes('/api/v1/backups/1') &&
					(c[1] as RequestInit)?.method === 'DELETE'
			);
			expect(deleteCall).toBeTruthy();
		});
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});

	it('shows download link only for completed backups', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([sampleBackup, failedBackup]))
			.mockResolvedValueOnce(jsonResponse(sampleStatus));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('backup-2026-03-13-120000.zip')).toBeInTheDocument();
		});

		const downloadLinks = screen.getAllByText('Stahnout');
		expect(downloadLinks).toHaveLength(1);

		const link = downloadLinks[0] as HTMLAnchorElement;
		expect(link.getAttribute('href')).toContain('/api/v1/backups/1/download');
	});

	it('shows sync info section', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse([]))
			.mockResolvedValueOnce(jsonResponse(idleStatusNoBackups));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Synchronizace dat')).toBeInTheDocument();
		});

		expect(screen.getByText(/ZFAKTURY_DATA_DIR/)).toBeInTheDocument();
		expect(screen.getByText(/journal_mode/)).toBeInTheDocument();
	});
});
