
import { defineStore } from 'pinia'

export const usePhaseCalStatusStore = defineStore({
  id: 'phase_cal_status_store',
  state: () => ({ 
    summary: "",
    status: [""],
    progress: 0,
    
  }),
  actions: {
    set_store(status: any) {
      if(status.summary.length > 0){
        this.summary = status.summary;
        }
        if(status.status.length > 0){
        this.status.push(status.status);
        if(this.status.length > 20){
          this.status.shift()
        }
      }
      if(status.progress){
        this.progress = status.progress;
      }

    },
    set_summary(summary: string) {
      this.summary = summary;
    },
    set_status(status: string[]) {
      this.status = status;
    },
    set_progress(progress: number) {
      this.progress = progress;
    },
  },
})
