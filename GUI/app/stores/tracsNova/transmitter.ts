import { defineStore } from 'pinia';
import { useTransmitterApi, type Transmitter, type TransmitterSavePayload } from '@/composables/tracsNova/useTransmitterApi';

export const useTransmitterStore = defineStore('TracsNova/transmitter', () => {
  const api = useTransmitterApi();

  const list = ref<Transmitter[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAll() {
    loading.value = true;
    error.value = null;
    try {
      const res = await api.getTransmitters();
      if (res.error.value) {
        error.value = 'Failed to load transmitters';
      } else {
        list.value = (res.data.value as Transmitter[]) ?? [];
      }
    } finally {
      loading.value = false;
    }
  }

  async function save(payload: TransmitterSavePayload): Promise<boolean> {
    const res = await api.saveTransmitter(payload);
    if (res.error.value) return false;
    await fetchAll();
    return true;
  }

  async function remove(code: string): Promise<boolean> {
    const res = await api.deleteTransmitter(code);
    if (res.error.value) return false;
    list.value = list.value.filter((t) => t.code !== code);
    return true;
  }

  const codeExists = (code: string, excludeCode?: string) =>
    list.value.some((t) => t.code === code && t.code !== excludeCode);

  const nameExists = (name: string, excludeCode?: string) =>
    list.value.some((t) => t.name === name && t.code !== excludeCode);

  return { list, loading, error, fetchAll, save, remove, codeExists, nameExists };
});
