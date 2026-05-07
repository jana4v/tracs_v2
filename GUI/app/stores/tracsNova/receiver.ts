import { defineStore } from 'pinia';
import { useTransmitterApi, type Receiver, type TransmitterSavePayload } from '@/composables/tracsNova/useTransmitterApi';

export const useReceiverStore = defineStore('TracsNova/receiver', () => {
  const api = useTransmitterApi();

  const list = ref<Receiver[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAll() {
    loading.value = true;
    error.value = null;
    try {
      const res = await api.getReceivers();
      if (res.error.value) {
        error.value = 'Failed to load receivers';
      } else {
        list.value = (res.data.value as Receiver[]) ?? [];
      }
    } finally {
      loading.value = false;
    }
  }

  async function save(payload: TransmitterSavePayload): Promise<boolean> {
    const res = await api.saveReceiver({
      ...payload,
      system_type: 'Receiver',
    });
    if (res.error.value) return false;
    await fetchAll();
    return true;
  }

  async function remove(code: string): Promise<boolean> {
    const res = await api.deleteReceiver(code);
    if (res.error.value) return false;
    list.value = list.value.filter((r) => r.code !== code);
    return true;
  }

  const codeExists = (code: string, excludeCode?: string) =>
    list.value.some((r) => r.code === code && r.code !== excludeCode);

  const nameExists = (name: string, excludeCode?: string) =>
    list.value.some((r) => r.name === name && r.code !== excludeCode);

  return { list, loading, error, fetchAll, save, remove, codeExists, nameExists };
});
