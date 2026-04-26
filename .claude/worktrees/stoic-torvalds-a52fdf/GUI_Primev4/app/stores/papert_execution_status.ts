// stores/testExecutionStatus.ts
import { defineStore } from 'pinia';

export const usePapertExecutionStatusStore = defineStore('papert_execution_statusStore', {
  state: () => ({
    summary: '' as string,
    status: [] as string[], // latest first
    progress: 0 as number,
  }),

  actions: {
    /**
     * Update the store with new values for summary, status, and progress.
     * @param payload - An object containing optional `summary`, `status`, and `progress`.
     */
    setStore(payload: { summary?: string; status?: string; progress?: number }) {
      if (payload.summary && payload.summary.length > 0) {
        this.summary = payload.summary;
      }

      if (payload.status && payload.status.length > 0) {
        this.status.unshift(payload.status); // <-- Add to start
        // Keep only the last 20 statuses (latest first)
        if (this.status.length > 20) {
          this.status.pop(); // <-- Remove oldest from end
        }
      }

      if (payload.progress !== undefined) {
        this.progress = parseInt(payload.progress.toString(), 10);
      }
    },

    setSummary(summary: string) {
      this.summary = summary;
    },

    setStatus(status: string[]) {
      this.status = status;
    },

    setProgress(progress: number) {
      this.progress = progress;
    },
  },
});
