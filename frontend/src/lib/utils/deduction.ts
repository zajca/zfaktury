// Shared labels for tax deduction categories.
//
// Categories correspond to the "nezdanitelné části základu daně" recognised by
// the Czech income tax act -- mortgage interest, life insurance, pension
// contributions, donations and union dues.

export const categoryLabels: Record<string, string> = {
	mortgage: 'Úroky z hypotéky',
	life_insurance: 'Životní pojištění',
	pension: 'Penzijní spoření',
	donation: 'Dary',
	union_dues: 'Odborové příspěvky'
};

/**
 * Translate a category key into its Czech label. Unknown keys fall back to the
 * raw key so the UI never renders an empty string.
 */
export function formatCategory(category: string): string {
	return categoryLabels[category] ?? category;
}
