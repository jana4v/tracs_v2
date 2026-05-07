<template>
  <div class="db-page">
    <!-- Sidebar -->
    <aside class="db-sidebar-wrapper">
      <TracsNovaDBSidebar
        :active-section="activeSection"
        @select="activeSection = $event"
        @add="handleAdd"
        @remove="handleRemove"
      />
    </aside>

    <!-- Main content -->
    <main class="db-content">
      <!-- Systems → Transmitter -->
      <TracsNovaDatabaseSystemsTransmitterPanel v-if="activeSection === 'transmitter'" />

      <!-- Systems → Receiver -->
      <TracsNovaDatabaseSystemsReceiverPanel v-else-if="activeSection === 'receiver'" />

      <!-- Systems → Transponder -->
      <TracsNovaDatabaseSystemsTransponderPanel v-else-if="activeSection === 'transponder'" />

      <!-- Specifications -->
      <TracsNovaDatabaseSpecificationsPanel
        v-else-if="selectedSpecParameter"
        :active-parameter="selectedSpecParameter"
      />

      <!-- Test Profiles → Transmitter / Spurious / Bands -->
      <SpuriousSearchBandsPanel v-else-if="activeSection === 'tp_tx_spurious_bands'" />

      <!-- Test Profiles → Transmitter / Spurious / Profile -->
      <SpuriousProfilePanel v-else-if="activeSection === 'tp_tx_spurious_profile'" />

      <!-- Test Profiles → Receiver -->
      <ReceiverTestProfilesPanel v-else-if="activeSection === 'test_profiles_receiver'" />

      <!-- Test Profiles → Transponder -->
      <TransponderTestProfilesPanel v-else-if="activeSection === 'test_profiles_transponder'" />

      <!-- Specifications → Ranging Threshold (standalone panel) -->
      <RangingThresholdPanel v-else-if="activeSection === 'specifications_ranging_threshold'" />

      <!-- Test Systems → Instruments -->
      <TestSystemsInstrumentsPanel v-else-if="activeSection === 'test_systems_instruments'" />

      <!-- Test Systems → TSM Paths -->
      <TestSystemsTsmPathsPanel v-else-if="activeSection === 'test_systems_tsm_paths'" />

      <!-- Test Systems → Power Meter -->
      <TestSystemsPowerMeterPanel v-else-if="activeSection === 'test_systems_power_meter'" />

      <!-- On Board Losses → Transmitter -->
      <OnBoardLossesTransmitterLossPanel v-else-if="activeSection === 'on_board_losses_transmitter'" />

      <!-- On Board Losses → Receiver -->
      <OnBoardLossesReceiverLossPanel v-else-if="activeSection === 'on_board_losses_receiver'" />

      <!-- On Board Losses → Transponder (coming soon) -->
      <div v-else-if="activeSection === 'on_board_losses_transponder'" class="coming-soon">
        <i class="pi pi-server" />
        <p>On Board Losses / <strong class="section-name">Transponder</strong> — coming soon</p>
      </div>

      <!-- Calibration → Transmitter -->
      <CalibrationTransmitterPanel v-else-if="activeSection === 'calibration_transmitter'" />

      <!-- Calibration → Receiver (coming soon) -->
      <div v-else-if="activeSection === 'calibration_receiver'" class="coming-soon">
        <i class="pi pi-server" />
        <p>Calibration / <strong class="section-name">Receiver</strong> — coming soon</p>
      </div>

      <!-- Calibration → Transponder (coming soon) -->
      <div v-else-if="activeSection === 'calibration_transponder'" class="coming-soon">
        <i class="pi pi-server" />
        <p>Calibration / <strong class="section-name">Transponder</strong> — coming soon</p>
      </div>

      <!-- Test Plan → Transmitter -->
      <TestPlanTransmitterPanel v-else-if="activeSection === 'test_plan_transmitter'" />

      <!-- Test Plan → Receiver -->
      <TestPlanReceiverPanel v-else-if="activeSection === 'test_plan_receiver'" />

      <!-- Test Plan → Transponder -->
      <TestPlanTransponderPanel v-else-if="activeSection === 'test_plan_transponder'" />

      <!-- ENV Data -->
      <EnvDataPanel v-else-if="activeSection === 'env_data'" />

      <!-- All other sections (placeholder) -->
      <div v-else class="coming-soon">
        <i class="pi pi-database" />
        <p>
          <strong class="section-name">{{ sectionLabel }}</strong> — coming soon
        </p>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
import { initMenu } from '@/composables/tracsNova/SideNav';
import SpuriousSearchBandsPanel from '@/components/tracsNova/Database/TestProfiles/SpuriousSearchBandsPanel.vue';
import SpuriousProfilePanel from '@/components/tracsNova/Database/TestProfiles/SpuriousProfilePanel.vue';
import ReceiverTestProfilesPanel from '@/components/tracsNova/Database/TestProfiles/ReceiverTestProfilesPanel.vue';
import TransponderTestProfilesPanel from '@/components/tracsNova/Database/TestProfiles/TransponderTestProfilesPanel.vue';
import TracsNovaDatabaseSystemsTransponderPanel from '@/components/tracsNova/Database/Systems/TransponderPanel.vue';
import TestSystemsInstrumentsPanel from '@/components/tracsNova/Database/TestSystems/InstrumentsPanel.vue';
import TestSystemsTsmPathsPanel from '@/components/tracsNova/Database/TestSystems/TsmPathsPanel.vue';
import TestSystemsPowerMeterPanel from '@/components/tracsNova/Database/TestSystems/PowerMeterPanel.vue';
import OnBoardLossesTransmitterLossPanel from '@/components/tracsNova/Database/OnBoardLosses/TransmitterLossPanel.vue';
import OnBoardLossesReceiverLossPanel from '@/components/tracsNova/Database/OnBoardLosses/ReceiverLossPanel.vue';
import CalibrationTransmitterPanel from '@/components/tracsNova/Database/Calibration/TransmitterCalibrationPanel.vue';
import TestPlanTransmitterPanel from '@/components/tracsNova/Database/TestPlan/TransmitterTestPlanPanel.vue';
import TestPlanReceiverPanel from '@/components/tracsNova/Database/TestPlan/ReceiverTestPlanPanel.vue';
import TestPlanTransponderPanel from '@/components/tracsNova/Database/TestPlan/TransponderTestPlanPanel.vue';
     import RangingThresholdPanel from '@/components/tracsNova/Database/Specifications/RangingThresholdPanel.vue';
import EnvDataPanel from '@/components/tracsNova/Database/EnvData/EnvDataPanel.vue';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

const activeSection = ref('transmitter');

const sectionLabels: Record<string, string> = {
  configurations: 'Configurations',
  specifications_power: 'Specifications / Power',
  specifications_frequency: 'Specifications / Frequency',
  specifications_modulation_index: 'Specifications / Modulation Index',
  specifications_spurious: 'Specifications / Spurious',
  specifications_command_threshold: 'Specifications / Command Threshold',
  test_systems_instruments: 'Test Systems / Instruments',
  test_systems_tsm_paths: 'Test Systems / TSM Paths',
  test_systems_power_meter: 'Test Systems / Power Meter',
  on_board_losses_transmitter: 'On Board Losses / Transmitter',
  on_board_losses_receiver: 'On Board Losses / Receiver',
  on_board_losses_transponder: 'On Board Losses / Transponder',
      specifications_ranging_threshold: 'Specifications / Ranging Threshold',
  calibration_transmitter: 'Calibration / Transmitter',
  calibration_receiver: 'Calibration / Receiver',
  calibration_transponder: 'Calibration / Transponder',
  test_plan_transmitter: 'Test Plan / Transmitter',
  test_plan_receiver: 'Test Plan / Receiver',
  test_plan_transponder: 'Test Plan / Transponder',
  tp_tx_spurious_bands: 'Test Profiles / Transmitter / Spurious / Bands',
  tp_tx_spurious_profile: 'Test Profiles / Transmitter / Spurious / Profile',
  test_profiles_receiver: 'Test Profiles / Receiver',
  test_profiles_transponder: 'Test Profiles / Transponder',
  tm_tc: 'TM TC',
  env_data: 'ENV Data',
};

const specSectionToParameter: Record<string, 'power' | 'frequency' | 'modulation_index' | 'spurious' | 'command_threshold'> = {
  specifications_power: 'power',
  specifications_frequency: 'frequency',
  specifications_modulation_index: 'modulation_index',
  specifications_spurious: 'spurious',
  specifications_command_threshold: 'command_threshold',
};

const selectedSpecParameter = computed(() => specSectionToParameter[activeSection.value]);

const sectionLabel = computed(() => sectionLabels[activeSection.value] ?? activeSection.value);

      // NOTE: 'specifications_ranging_threshold' is handled by its own panel above
initMenu(3);

// Persist active section across navigation/reloads.
const ui = useUiStatePersistence('ui_state:tracsNova:database');
ui.bindRefs({ activeSection });
onMounted(() => { void ui.load(); });

function handleAdd() {
  // Future: trigger add action in active panel via event bus or store
}

function handleRemove() {
  // Future: trigger remove action in active panel
}
</script>

<style scoped>
.db-page {
  display: flex;
  height: calc(100vh - 4rem);
  min-height: 0;
  background: #081525;
}

.db-sidebar-wrapper {
  width: 220px;
  flex-shrink: 0;
}

.db-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0;
}

/* Placeholder "coming soon" panels */
.coming-soon {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  height: 60vh;
  color: #475569;
  font-size: 1rem;
}

.coming-soon .pi {
  font-size: 3rem;
  color: #1e3a5f;
}

.section-name {
  color: #22d3ee;
}

/* ── Light theme overrides ────────────────────────────────────────────── */
html:not(.dark) .db-page {
  background: var(--p-surface-50);
}
html:not(.dark) .coming-soon {
  color: var(--p-text-muted-color);
}
html:not(.dark) .coming-soon .pi {
  color: var(--p-surface-300);
}
</style>
