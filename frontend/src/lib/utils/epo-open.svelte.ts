import {
	ApiError,
	epoApi,
	epoFindingsFromError,
	type EPOFinding,
	type EPOKind
} from '$lib/api/client';
import { detectDesktop } from '$lib/utils/download';
import { isEPOFormUrl } from '$lib/utils/epo';
import { prepareExternalOpen } from '$lib/utils/external';

const MSG_NETWORK = 'Nepodařilo se spojit se serverem.';
const MSG_OPEN_FAILED = 'Formulář se nepodařilo otevřít v prohlížeči.';
const MSG_BAD_URL = 'Server vrátil neplatný odkaz na formulář EPO, formulář nebyl otevřen.';

/** State of one "open in EPO" action, shared by the button and its result panel. */
export class EPOOpenState {
	desktop = $state(false);
	desktopChecked = $state(false);
	loading = $state(false);
	errorMessage = $state<string | null>(null);
	blockingFindings = $state<EPOFinding[]>([]);
	warnings = $state<EPOFinding[]>([]);
	opened = $state(false);
	fallbackUrl = $state<string | null>(null);

	async init(): Promise<void> {
		this.desktop = await detectDesktop();
		this.desktopChecked = true;
	}

	async open(kind: EPOKind, id: number): Promise<void> {
		if (this.loading) return;
		// Must run before the first await so the new tab keeps the click gesture.
		const pending = prepareExternalOpen(this.desktop);

		this.loading = true;
		this.errorMessage = null;
		this.blockingFindings = [];
		this.warnings = [];
		this.opened = false;
		this.fallbackUrl = null;
		try {
			let result;
			try {
				result = await epoApi.openForm(kind, id);
			} catch (e) {
				pending.cancel();
				if (e instanceof ApiError) {
					this.errorMessage = e.message;
					this.blockingFindings = epoFindingsFromError(e);
				} else {
					this.errorMessage = MSG_NETWORK;
				}
				return;
			}

			if (!isEPOFormUrl(result.url)) {
				pending.cancel();
				this.errorMessage = MSG_BAD_URL;
				return;
			}

			this.warnings = result.warnings;
			try {
				if (await pending.open(result.url)) {
					this.opened = true;
				} else {
					this.fallbackUrl = result.url;
				}
			} catch {
				pending.cancel();
				this.warnings = [];
				this.errorMessage = MSG_OPEN_FAILED;
			}
		} finally {
			this.loading = false;
		}
	}
}
