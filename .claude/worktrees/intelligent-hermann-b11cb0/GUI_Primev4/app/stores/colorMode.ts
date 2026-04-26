// stores/colorMode.ts
import { defineStore } from 'pinia';

// Define the type for the state
interface ColorModeState {
  currentMode: string; // You can use a more specific type if needed (e.g., 'light' | 'dark' | 'auto')
}

export const useColorModeStore = defineStore('colorMode', {
  // State with type annotation
  state: (): ColorModeState => ({
    currentMode: 'auto', // Default mode
  }),

  // Actions with type-safe methods
  actions: {
    setCurrentMode(mode: string): void {
      this.currentMode = mode;
    },
  },

  // Optional: Getters for derived state (if needed)
  getters: {
    getCurrentMode(state): string {
      return state.currentMode;
    },
  },
});