import { defineStore } from 'pinia'

export interface DialogInputOption {
  type?: 'text' | 'number' | 'password' // extend as needed
  placeholder?: string
  default?: string | number
}

export interface DialogRequest {
  app_name: string
  dialogTitle: string
  dialogMessage: string
  dialogOptions: string[]
  dialogInput?: DialogInputOption
}

interface DialogRequestState {
  dialogRequest: DialogRequest | null
}

export const useDialogRequestStore = defineStore('dialogRequest', {
  state: (): DialogRequestState => ({
    dialogRequest: null,
  }),
  actions: {
    setDialogRequest(request: DialogRequest) {
      this.dialogRequest = request
    },
    clearDialogRequest() {
      this.dialogRequest = null
    },
  },
})
