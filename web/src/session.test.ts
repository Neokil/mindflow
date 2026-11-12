import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { PageSession } from './session';
import type { Transport } from './session';
import { emptyDocument } from './model';
import { APIError } from './api';
let sessions: PageSession[] = [];
beforeEach(() => { vi.stubGlobal('navigator', { onLine: true }); });
afterEach(() => { sessions.forEach(s => s.dispose()); sessions = []; vi.unstubAllGlobals(); });
function make(transport: Transport) { const s = new PageSession('page', transport); s.loaded = true; s.editable = true; s.epoch = 1; sessions.push(s); return s; }
it('does not mark an edit made during an in-flight save as saved', async () => {
 let resolve!: (v: unknown) => void; const transport = vi.fn(() => new Promise(r => { resolve = r; })) as unknown as Transport; const s = make(transport);
 s.edit({ ...emptyDocument(), settings: { background: 'plain', gridSize: 20, snap: false } }); const first = s.flush(); await Promise.resolve(); await Promise.resolve(); s.edit({ ...s.doc, settings: { ...s.doc.settings, gridSize: 40 } }); resolve({ revision: 1 }); await first; expect(s.dirty).toBe(true); expect(s.revision).toBe(1);
});
it('retains dirty local work when another editor takes control', async () => {
 const transport = vi.fn(async (_path: string, method?: string) => { if (method === 'PUT') throw new APIError(409, 'lease_lost', 'Lost control'); return { document: emptyDocument(), revision: 2 }; }) as unknown as Transport; const s = make(transport); s.edit({ ...emptyDocument(), settings: { background: 'plain', gridSize: 33, snap: true } }); await s.flush(); expect(s.editable).toBe(false); expect(s.recovery?.settings.gridSize).toBe(33); expect(s.doc.settings.gridSize).toBe(20); expect(s.unsaved).toBe(true);
});
