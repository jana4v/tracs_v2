
import { defineStore } from 'pinia'

export const useMyLoginStore = defineStore({
  id: 'myLoginStore',
  state: () => ({ 
    token:"",
  }),
  getters: {
    get_access_token(state) {
      if(state.token == ""){
        return localStorage.getItem('token')
      }
      return state.token
    },
  },
  actions: {
    set_access_token(token: string) {
      // you can directly mutate the state
      this.token = token;
      localStorage.setItem("token",token);
    },
  },
})
