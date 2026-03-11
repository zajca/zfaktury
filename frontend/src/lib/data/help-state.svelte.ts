import type { HelpTopicId } from './help-content';

export const helpDrawer = $state({
	open: false,
	topicId: null as HelpTopicId | null
});

export function openHelp(topicId: HelpTopicId) {
	helpDrawer.topicId = topicId;
	helpDrawer.open = true;
}

export function closeHelp() {
	helpDrawer.open = false;
}
