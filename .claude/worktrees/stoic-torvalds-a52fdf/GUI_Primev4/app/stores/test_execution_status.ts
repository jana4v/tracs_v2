// stores/testExecutionStatus.ts
import { defineStore } from 'pinia';

export const useTestExecutionStatusStore = defineStore('myTest_execution_statusStore', {
  // State definition with TypeScript interface
  state: () => ({
    summary: '' as string,
    status: [] as string[],
    progress: 0 as number,
  }),

  // Actions to modify the state
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
        this.status.push(payload.status);
        // Keep only the last 20 statuses
        if (this.status.length > 20) {
          this.status.shift();
        }
      }

      if (payload.progress !== undefined) {
        this.progress = parseInt(payload.progress.toString(), 10);
      }
    },

    /**
     * Set the summary value.
     * @param summary - The new summary string.
     */
    setSummary(summary: string) {
      this.summary = summary;
    },

    /**
     * Set the status array.
     * @param status - The new status array.
     */
    setStatus(status: string[]) {
      this.status = status;
    },

    /**
     * Set the progress value.
     * @param progress - The new progress number.
     */
    setProgress(progress: number) {
      this.progress = progress;
    },
  },
});