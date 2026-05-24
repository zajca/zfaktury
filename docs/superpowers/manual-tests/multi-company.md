# Multi-Company Manual Smoke Test

Run before merging the multi-company branch. Exercises the full feature
end-to-end through the UI.

## Setup

1. Fresh DB: `rm -f ~/.zfaktury/zfaktury.db` (or set `ZFAKTURY_DATA_DIR=/tmp/zft-smoke` to a fresh path)
2. `make dev` (Vite + Go server)
3. Open http://localhost:5173

## Checklist

- [ ] **Empty state**: app routes to `/companies/new` (no companies exist yet)
- [ ] **Create first company**: enter IČO `12345678` (or any valid 8-digit), click "Načíst z ARES" — name/address pre-fill from ARES. Save. App routes to `/` dashboard with "Manas OSVČ" (or whatever ARES returned) shown in the header.
- [ ] **Add a contact**: go to `/contacts/new`, create "Keboola s.r.o." (ICO 25620916). Verify it appears in the contacts list.
- [ ] **Add an invoice**: go to `/invoices/new`, customer = the contact, add one line item. Save. Verify it appears in `/invoices` with status "Vystaveno".
- [ ] **Add second company**: click the dropdown header → "+ Přidat firmu". Create "Test s.r.o." with a different IČO. The dropdown should now list both companies.
- [ ] **Switch via dropdown**: click "Test s.r.o." in the dropdown. Verify:
  - The header label changes to "Test s.r.o."
  - The contacts list becomes empty (different company)
  - The invoice list becomes empty
  - The dashboard totals change to 0
- [ ] **Create contact in s.r.o.**: add "Acme spol. s r.o." (only visible while s.r.o. is active)
- [ ] **Sequence collision check**: in s.r.o., create an invoice. The next invoice number should be FV20260001 (NOT FV20260002) — each company has its own sequence.
- [ ] **Switch back to OSVČ**: original contact and invoice should reappear.
- [ ] **Cross-company contact rejection**: in OSVČ, try to create an invoice via the API directly using s.r.o.'s contact ID:
  ```bash
  curl -sS -X POST http://localhost:5173/api/v1/companies/1/invoices \
    -H 'content-type: application/json' \
    -d '{"customer_id": <s.r.o.-contact-id>, ...}'
  ```
  Should return 4xx with a not-found-style error (the contact doesn't belong to OSVČ).
- [ ] **In-use delete protection**: from `/companies`, click "Smazat" on OSVČ (which has an invoice). Should error with "cannot delete: still in use".
- [ ] **Last-company delete protection**: soft-delete all data in s.r.o. (or just remember it's the only OTHER company). Delete s.r.o. successfully. Then try to delete OSVČ — should error with "cannot delete the last company".
- [ ] **Reload persistence**: after switching to OSVČ, hit browser refresh. The header should still show OSVČ (localStorage restored).

## What to look for

- Header dropdown closes when clicking outside
- Switching companies shows a brief loading state (no flicker of stale data)
- Error messages are in Czech
- Forms validate before submitting
- `X-Company-Id` response header is present on every per-company response (open browser devtools Network panel)

## If anything fails

Document in the PR description as a regression to address before merge.
