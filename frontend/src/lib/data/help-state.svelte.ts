import type { TaxConstants } from '$lib/api/client';
import type { HelpTopicId } from './help-content';

export const helpDrawer = $state({
	open: false,
	topicId: null as HelpTopicId | null,
	taxConstants: null as TaxConstants | null
});

export function openHelp(topicId: HelpTopicId, taxConstants?: TaxConstants | null) {
	helpDrawer.topicId = topicId;
	helpDrawer.taxConstants = taxConstants ?? null;
	helpDrawer.open = true;
}

export function closeHelp() {
	helpDrawer.open = false;
}
