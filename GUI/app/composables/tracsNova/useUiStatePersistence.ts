import { ref, watch, type Ref } from 'vue';
import { useTransmitterApi } from '@/composables/tracsNova/useTransmitterApi';

/**
 * Persists arbitrary UI state (toolbar selections, active tabs, AG-Grid layout
 * + expanded groups) to the backend Configuration table under a single
 * parameter key so the page restores its state across navigation/reloads.
 *
 * Storage shape (JSON-stringified into the Configuration row's `value`):
 *   {
 *     simple: { <key>: <jsonable>, ... },
 *     grids:  { <gridKind>: { columnState: [...], expandedKeys: [...] } }
 *   }
 *
 * Usage:
 *   const ui = useUiStatePersistence('ui_state:tracsNova:database');
 *   ui.bindRefs({ activeSection });
 *   ui.registerGrid('myGrid');
 *   // template: @grid-ready="(e) => ui.onGridReady('myGrid', e)"
 *   //           @column-moved="() => ui.notifyGridChanged('myGrid')" ...
 *   onMounted(async () => { await ui.load(); });
 */
export function useUiStatePersistence(parameterKey: string, debounceMs = 500) {
  const api = useTransmitterApi();

  const loaded = ref(false);
  const simpleRefs: Record<string, Ref<any>> = {};
  const gridApis: Record<string, any> = {};
  const pendingGridState: Record<string, { columnState?: any[]; expandedKeys?: string[] }> = {};
  let saveTimer: ReturnType<typeof setTimeout> | null = null;

  function bindRefs(refs: Record<string, Ref<any>>) {
    for (const [k, r] of Object.entries(refs)) {
      simpleRefs[k] = r;
      // Watch each ref so changes (post-load) trigger a save.
      watch(r, () => scheduleSave(), { deep: true });
    }
  }

  function registerGrid(kind: string) {
    if (!(kind in gridApis)) gridApis[kind] = null;
  }

  function onGridReady(kind: string, event: any) {
    gridApis[kind] = event?.api ?? null;
    // Auto-subscribe to AG-Grid's catch-all state event so callers don't have
    // to wire eight template event handlers per grid. Falls back silently if
    // the event name isn't supported by the installed AG-Grid version.
    try {
      event?.api?.addEventListener?.('stateUpdated', () => scheduleSave());
      event?.api?.addEventListener?.('rowGroupOpened', () => scheduleSave());
    } catch { /* noop */ }
    const pending = pendingGridState[kind];
    if (pending) {
      applyGridStateInternal(kind, pending);
      delete pendingGridState[kind];
    }
  }

  function notifyGridChanged(_kind: string) {
    scheduleSave();
  }

  function captureGridState(kind: string): { columnState: any[]; expandedKeys: string[] } | null {
    const grid = gridApis[kind];
    if (!grid) return null;
    let columnState: any[] = [];
    try { columnState = grid.getColumnState ? grid.getColumnState() : []; } catch { /* noop */ }
    const expandedKeys: string[] = [];
    try {
      grid.forEachNode?.((node: any) => {
        if (node?.group && node.expanded) {
          const path: string[] = [];
          let cur: any = node;
          while (cur && cur.group) {
            path.unshift(`${String(cur.field ?? '')}=${String(cur.key ?? '')}`);
            cur = cur.parent;
          }
          if (path.length > 0) expandedKeys.push(path.join('/'));
        }
      });
    } catch { /* noop */ }
    return { columnState, expandedKeys };
  }

  function applyGridStateInternal(kind: string, state: { columnState?: any[]; expandedKeys?: string[] } | undefined) {
    if (!state) return;
    const grid = gridApis[kind];
    if (!grid) {
      pendingGridState[kind] = state;
      return;
    }
    if (Array.isArray(state.columnState) && state.columnState.length > 0) {
      try { grid.applyColumnState?.({ state: state.columnState, applyOrder: true }); } catch { /* noop */ }
    }
    const expanded = new Set(state.expandedKeys ?? []);
    if (expanded.size > 0) {
      try {
        grid.forEachNode?.((node: any) => {
          if (!node?.group) return;
          const path: string[] = [];
          let cur: any = node;
          while (cur && cur.group) {
            path.unshift(`${String(cur.field ?? '')}=${String(cur.key ?? '')}`);
            cur = cur.parent;
          }
          if (path.length > 0 && expanded.has(path.join('/'))) {
            node.setExpanded?.(true);
          }
        });
      } catch { /* noop */ }
    }
  }

  /**
   * Re-apply the most recently captured grid state for `kind`. Useful to call
   * from a `watch` on the row data so that AG-Grid's group expansion isn't
   * lost when rowData is rebuilt.
   */
  function reapplyGridState(kind: string) {
    if (!loaded.value) return;
    const s = captureGridState(kind);
    if (s) applyGridStateInternal(kind, s);
  }

  function captureState() {
    const simple: Record<string, any> = {};
    for (const [k, r] of Object.entries(simpleRefs)) {
      try {
        // Drop functions / undefined; rely on JSON to skip non-serialisable.
        simple[k] = JSON.parse(JSON.stringify(r.value));
      } catch {
        simple[k] = null;
      }
    }
    const grids: Record<string, any> = {};
    for (const kind of Object.keys(gridApis)) {
      const g = captureGridState(kind);
      if (g) grids[kind] = g;
    }
    return { simple, grids };
  }

  function scheduleSave() {
    if (!loaded.value) return;
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(async () => {
      saveTimer = null;
      try {
        const blob = JSON.stringify(captureState());
        await api.saveConfigurationValue(parameterKey, { value: blob });
      } catch { /* best-effort */ }
    }, debounceMs);
  }

  async function load() {
    try {
      const res = await api.getConfigurationValue(parameterKey);
      if (res.error.value || !res.data.value) return;
      const payload: any = res.data.value;
      const raw = String(payload?.value ?? '').trim();
      if (!raw) return;
      let state: any = null;
      try { state = JSON.parse(raw); } catch { state = null; }
      if (!state || typeof state !== 'object') return;

      const simple = state.simple ?? {};
      for (const [k, r] of Object.entries(simpleRefs)) {
        if (Object.prototype.hasOwnProperty.call(simple, k)) {
          try { r.value = simple[k]; } catch { /* noop */ }
        }
      }
      const grids = state.grids ?? {};
      for (const kind of Object.keys(gridApis)) {
        if (grids[kind]) applyGridStateInternal(kind, grids[kind]);
      }
    } finally {
      loaded.value = true;
    }
  }

  return {
    loaded,
    bindRefs,
    registerGrid,
    onGridReady,
    notifyGridChanged,
    reapplyGridState,
    load,
    /** Force-save now (skips debounce). */
    saveNow: async () => {
      if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; }
      try {
        const blob = JSON.stringify(captureState());
        await api.saveConfigurationValue(parameterKey, { value: blob });
      } catch { /* best-effort */ }
    },
  };
}
