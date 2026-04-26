
import { defineStore } from 'pinia'

export const useMyHomeStore = defineStore({
  id: 'myHomeStore',
  state: () => ({
    is_tx_plan_selected: true,
    is_rx_plan_selected: true,
    is_tp_plan_selected: true,
    selected_test_phase: '', 
    selected_sub_test_phase: '',
    selected_test_plan: '',
    tx_plan: [] as any,
    tx_plan_selected_count: 0, 
    tx_plan_estimated_execution_time: 0,
    rx_plan: [] as any,
    rx_plan_selected_count: 0, 
    rx_plan_estimated_execution_time: 0,
    tp_plan: [] as any,
    tp_plan_selected_count: 0,
    tp_plan_estimated_execution_time: 0,
    test_execution_mode:'Normal'
  }),
  actions: {
    set_is_tx_plan_selected(is_tx_plan_selected: boolean) {
      this.is_tx_plan_selected = is_tx_plan_selected;
    },
    set_is_rx_plan_selected(is_rx_plan_selected: boolean) {
      this.is_rx_plan_selected = is_rx_plan_selected;
    },
    set_is_tp_plan_selected(is_tp_plan_selected: boolean) {
      this.is_tp_plan_selected = is_tp_plan_selected;
    },
       
    set_tx_plan(tx_plan: any[]) {
      this.tx_plan = tx_plan;
    },
    set_tx_plan_selected_count(tx_plan_selected_count: number) { 
      this.tx_plan_selected_count = tx_plan_selected_count;
    },
    set_rx_plan(rx_plan: any[]) {
      this.rx_plan = rx_plan;
    },
    set_rx_plan_selected_count(rx_plan_selected_count: number) { 
      this.rx_plan_selected_count = rx_plan_selected_count;
    },
    set_tp_plan(tp_plan: any[]) {
      this.tp_plan = tp_plan;
    },
    set_tp_plan_selected_count(tp_plan_selected_count: number) { 
      this.tp_plan_selected_count = tp_plan_selected_count;
    },
    set_selected_test_phase(selected_test_phase: string) { 
      this.selected_test_phase = selected_test_phase;
    },
    set_selected_sub_test_phase(selected_sub_test_phase: string) {
      this.selected_sub_test_phase = selected_sub_test_phase;
    },
    set_selected_test_plan(selected_test_plan: string) {
      this.selected_test_plan = selected_test_plan;
    },
    set_test_execution_mode(test_execution_mode: string) {
      this.test_execution_mode = test_execution_mode;
    }
  },
})
