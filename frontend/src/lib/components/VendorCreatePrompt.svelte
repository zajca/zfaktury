<script lang="ts">
	import { contactsApi, type Contact } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		initialIco: string;
		initialName: string;
		initialDic?: string;
		oncreate: (contact: Contact) => void;
		onskip: () => void;
		oncancel: () => void;
	}

	let { initialIco, initialName, initialDic = '', oncreate, onskip, oncancel }: Props = $props();

	// Snapshot prop values once when the dialog mounts; users can edit the form freely afterwards.
	function initialForm() {
		return {
			name: initialName,
			ico: initialIco,
			dic: initialDic,
			street: '',
			city: '',
			zip: '',
			country: 'CZ'
		};
	}
	let form = $state(initialForm());

	let aresLoading = $state(false);
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

	const inputClass =
		'w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';

	async function handleAres() {
		if (!form.ico.trim()) {
			errorMsg = 'Pro načtení z ARES je nutné zadat IČO.';
			return;
		}
		aresLoading = true;
		errorMsg = null;
		try {
			const ares = await contactsApi.lookupAres(form.ico.trim());
			form.name = ares.name || form.name;
			form.dic = ares.dic || form.dic;
			form.street = ares.street || form.street;
			form.city = ares.city || form.city;
			form.zip = ares.zip || form.zip;
			form.country = ares.country || form.country;
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Načtení z ARES se nezdařilo.';
		} finally {
			aresLoading = false;
		}
	}

	async function handleCreate() {
		if (!form.name.trim()) {
			errorMsg = 'Název dodavatele je povinný.';
			return;
		}
		saving = true;
		errorMsg = null;
		try {
			const contact = await contactsApi.create({
				type: 'company',
				name: form.name.trim(),
				ico: form.ico.trim(),
				dic: form.dic.trim(),
				street: form.street.trim(),
				city: form.city.trim(),
				zip: form.zip.trim(),
				country: form.country.trim() || 'CZ'
			});
			oncreate(contact);
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Vytvoření dodavatele se nezdařilo.';
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') oncancel();
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center" onkeydown={handleKeydown}>
	<div class="fixed inset-0 bg-overlay" role="presentation" onclick={oncancel}></div>

	<div
		class="relative z-50 w-full max-w-xl bg-surface rounded-xl border border-border shadow-xl p-6 max-h-[90vh] overflow-y-auto"
		role="dialog"
		aria-modal="true"
		aria-labelledby="vendor-create-title"
	>
		<h2 id="vendor-create-title" class="text-lg font-semibold text-primary mb-2">
			Dodavatel není v kontaktech
		</h2>
		<p class="text-sm text-secondary mb-5">
			Dodavatel z OCR nebyl nalezen v adresáři kontaktů. Můžete ho rovnou vytvořit (volitelně s
			doplněním údajů z ARES) nebo pokračovat bez přiřazení dodavatele.
		</p>

		{#if errorMsg}
			<div
				role="alert"
				class="mb-4 rounded-lg bg-danger/10 border border-danger/30 px-3 py-2 text-sm text-danger"
			>
				{errorMsg}
			</div>
		{/if}

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleCreate();
			}}
			class="space-y-4"
		>
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="vendor-ico" class="block text-sm font-medium text-secondary mb-1">IČO</label>
					<input id="vendor-ico" type="text" bind:value={form.ico} class={inputClass} />
				</div>
				<div class="flex items-end">
					<Button
						variant="secondary"
						onclick={handleAres}
						disabled={aresLoading || !form.ico.trim()}
					>
						{aresLoading ? 'Načítání…' : 'Načíst z ARES'}
					</Button>
				</div>
			</div>

			<div>
				<label for="vendor-name" class="block text-sm font-medium text-secondary mb-1"
					>Název *</label
				>
				<input id="vendor-name" type="text" bind:value={form.name} class={inputClass} required />
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="vendor-dic" class="block text-sm font-medium text-secondary mb-1">DIČ</label>
					<input id="vendor-dic" type="text" bind:value={form.dic} class={inputClass} />
				</div>
				<div>
					<label for="vendor-country" class="block text-sm font-medium text-secondary mb-1"
						>Stát</label
					>
					<input id="vendor-country" type="text" bind:value={form.country} class={inputClass} />
				</div>
			</div>

			<div>
				<label for="vendor-street" class="block text-sm font-medium text-secondary mb-1"
					>Ulice</label
				>
				<input id="vendor-street" type="text" bind:value={form.street} class={inputClass} />
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="vendor-city" class="block text-sm font-medium text-secondary mb-1"
						>Město</label
					>
					<input id="vendor-city" type="text" bind:value={form.city} class={inputClass} />
				</div>
				<div>
					<label for="vendor-zip" class="block text-sm font-medium text-secondary mb-1">PSČ</label>
					<input id="vendor-zip" type="text" bind:value={form.zip} class={inputClass} />
				</div>
			</div>

			<div class="flex justify-between items-center gap-3 pt-4 border-t border-border">
				<Button variant="secondary" onclick={onskip} disabled={saving}>Přeskočit</Button>
				<div class="flex gap-3">
					<Button variant="secondary" onclick={oncancel} disabled={saving}>Zrušit</Button>
					<Button variant="primary" type="submit" disabled={saving}>
						{saving ? 'Ukládání…' : 'Vytvořit a použít'}
					</Button>
				</div>
			</div>
		</form>
	</div>
</div>
