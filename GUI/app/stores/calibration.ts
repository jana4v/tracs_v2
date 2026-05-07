import { defineStore } from 'pinia'

export const useMyCalibrationStore = defineStore({
  id: 'myCalibrationStore',
  state: () => ({
    selected_cal_id: '',
    selected_cal_type: '',
    selected_up_link_channels: [],
    selected_down_link_channels: [],
    selected_tvac_ref_cable_channels: [],
  }),
  actions: {
    set_selected_cal_id(cal_id: string) {
      this.selected_cal_id = cal_id
    },
    set_selected_cal_type(cal_type: string) {
      this.selected_cal_type = cal_type
    },
    set_selected_up_link_channels(up_link_channels: any) {
      this.selected_up_link_channels = up_link_channels
    },
    set_selected_down_link_channels(down_link_channels: any) {
      this.selected_down_link_channels = down_link_channels
    },
    set_selected_tvac_ref_cable_channels(tvac_ref_cable_channels: any) {
      this.selected_tvac_ref_cable_channels = tvac_ref_cable_channels
    },
  },
})
