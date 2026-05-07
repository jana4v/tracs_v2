<template>
  <div class="db-sidebar">
    <!-- Top control buttons -->
    <div class="sidebar-controls">
      <button class="ctrl-btn" title="Expand all" @click="expandAll">+</button>
      <button class="ctrl-btn" title="Collapse all" @click="collapseAll">−</button>
    </div>

    <!-- Menu -->
    <nav class="sidebar-nav">
      <!-- Systems group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: systemsExpanded }"
          @click="systemsExpanded = !systemsExpanded"
        >
          <span class="nav-icon"><i class="pi pi-database" /></span>
          <span class="nav-label">Systems</span>
          <i class="pi nav-chevron" :class="systemsExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="systemsExpanded" class="nav-children">
            <div
              v-for="sub in systemSubItems"
              :key="sub.key"
              class="nav-child-item"
              :class="{ 'child-active': activeSection === sub.key }"
              @click="select(sub.key)"
            >
              <i class="pi pi-external-link child-icon" />
              <span>{{ sub.label }}</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- Specifications group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: specificationsExpanded || isSpecificationsActive }"
          @click="specificationsExpanded = !specificationsExpanded"
        >
          <span class="nav-icon"><i class="pi pi-database" /></span>
          <span class="nav-label">Specifications</span>
          <i class="pi nav-chevron" :class="specificationsExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="specificationsExpanded" class="nav-children">
            <div
              v-for="sub in specificationSubItems"
              :key="sub.key"
              class="nav-child-item"
              :class="{ 'child-active': activeSection === sub.key }"
              @click="select(sub.key)"
            >
              <i class="pi pi-external-link child-icon" />
              <span>{{ sub.label }}</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- Test Profiles group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: testProfilesExpanded || isTestProfilesActive }"
          @click="testProfilesExpanded = !testProfilesExpanded"
        >
          <span class="nav-icon"><i class="pi pi-chart-line" /></span>
          <span class="nav-label">Test Profiles</span>
          <i class="pi nav-chevron" :class="testProfilesExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="testProfilesExpanded" class="nav-children">

            <!-- Transmitter — expandable sub-group -->
            <div
              class="nav-child-item nav-child-expandable"
              :class="{ 'child-active': tpTxExpanded }"
              @click="tpTxExpanded = !tpTxExpanded"
            >
              <i class="pi pi-external-link child-icon" />
              <span class="flex-1">Transmitter</span>
              <i class="pi nav-chevron" :class="tpTxExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
            </div>

            <transition name="expand">
              <div v-if="tpTxExpanded" class="nav-subchildren">

                <!-- Spurious — expandable sub-sub-group -->
                <div
                  class="nav-subchild-item nav-child-expandable"
                  :class="{ 'subchild-active': tpTxSpuriousExpanded }"
                  @click="tpTxSpuriousExpanded = !tpTxSpuriousExpanded"
                >
                  <i class="pi pi-sliders-h child-icon" />
                  <span class="flex-1">Spurious</span>
                  <i class="pi nav-chevron" :class="tpTxSpuriousExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
                </div>

                <transition name="expand">
                  <div v-if="tpTxSpuriousExpanded" class="nav-grandchildren">
                    <div
                      class="nav-grandchild-item"
                      :class="{ 'grandchild-active': activeSection === 'tp_tx_spurious_bands' }"
                      @click="select('tp_tx_spurious_bands')"
                    >
                      <i class="pi pi-minus grandchild-icon" />
                      <span>Bands</span>
                    </div>
                    <div
                      class="nav-grandchild-item"
                      :class="{ 'grandchild-active': activeSection === 'tp_tx_spurious_profile' }"
                      @click="select('tp_tx_spurious_profile')"
                    >
                      <i class="pi pi-minus grandchild-icon" />
                      <span>Profile</span>
                    </div>
                  </div>
                </transition>

              </div>
            </transition>

            <!-- Receiver leaf -->
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_profiles_receiver' }"
              @click="select('test_profiles_receiver')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Receiver</span>
            </div>

            <!-- Transponder leaf -->
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_profiles_transponder' }"
              @click="select('test_profiles_transponder')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transponder</span>
            </div>

          </div>
        </transition>
      </div>

      <!-- Test Systems group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: testSystemsExpanded || isTestSystemsActive }"
          @click="testSystemsExpanded = !testSystemsExpanded"
        >
          <span class="nav-icon"><i class="pi pi-sitemap" /></span>
          <span class="nav-label">Test Systems</span>
          <i class="pi nav-chevron" :class="testSystemsExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="testSystemsExpanded" class="nav-children">
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_systems_instruments' }"
              @click="select('test_systems_instruments')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Instruments</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_systems_tsm_paths' }"
              @click="select('test_systems_tsm_paths')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>TSM Paths</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_systems_power_meter' }"
              @click="select('test_systems_power_meter')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Power Meter</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- On Board Losses group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: onBoardLossesExpanded || isOnBoardLossesActive }"
          @click="onBoardLossesExpanded = !onBoardLossesExpanded"
        >
          <span class="nav-icon"><i class="pi pi-wave-pulse" /></span>
          <span class="nav-label">On Board Losses</span>
          <i class="pi nav-chevron" :class="onBoardLossesExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="onBoardLossesExpanded" class="nav-children">
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'on_board_losses_transmitter' }"
              @click="select('on_board_losses_transmitter')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transmitter</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'on_board_losses_receiver' }"
              @click="select('on_board_losses_receiver')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Receiver</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- Calibration group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: calibrationExpanded || isCalibrationActive }"
          @click="calibrationExpanded = !calibrationExpanded"
        >
          <span class="nav-icon"><i class="pi pi-sliders-h" /></span>
          <span class="nav-label">Calibration</span>
          <i class="pi nav-chevron" :class="calibrationExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="calibrationExpanded" class="nav-children">
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'calibration_transmitter' }"
              @click="select('calibration_transmitter')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transmitter</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'calibration_receiver' }"
              @click="select('calibration_receiver')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Receiver</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'calibration_transponder' }"
              @click="select('calibration_transponder')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transponder</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- Test Plan group (expandable) -->
      <div class="nav-group">
        <div
          class="nav-item nav-group-header"
          :class="{ active: testPlanExpanded || isTestPlanActive }"
          @click="testPlanExpanded = !testPlanExpanded"
        >
          <span class="nav-icon"><i class="pi pi-list-check" /></span>
          <span class="nav-label">Test Plan</span>
          <i class="pi nav-chevron" :class="testPlanExpanded ? 'pi-chevron-down' : 'pi-chevron-right'" />
        </div>

        <transition name="expand">
          <div v-if="testPlanExpanded" class="nav-children">
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_plan_transmitter' }"
              @click="select('test_plan_transmitter')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transmitter</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_plan_receiver' }"
              @click="select('test_plan_receiver')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Receiver</span>
            </div>
            <div
              class="nav-child-item"
              :class="{ 'child-active': activeSection === 'test_plan_transponder' }"
              @click="select('test_plan_transponder')"
            >
              <i class="pi pi-external-link child-icon" />
              <span>Transponder</span>
            </div>
          </div>
        </transition>
      </div>

      <!-- Other top-level sections -->
      <div
        v-for="item in topLevelItems"
        :key="item.key"
        class="nav-item"
        :class="{ active: activeSection === item.key }"
        @click="select(item.key)"
      >
        <span class="nav-icon"><i class="pi pi-database" /></span>
        <span class="nav-label">{{ item.label }}</span>
      </div>
    </nav>
  </div>
</template>

<script lang="ts" setup>
const emit = defineEmits<{
  (e: 'select', section: string): void;
  (e: 'add'): void;
  (e: 'remove'): void;
}>();

const props = defineProps<{
  activeSection: string;
}>();

const systemsExpanded = ref(true);
const specificationsExpanded = ref(true);
const testProfilesExpanded = ref(false);
const testSystemsExpanded = ref(false);
const onBoardLossesExpanded = ref(false);
const calibrationExpanded = ref(false);
const testPlanExpanded = ref(false);
const tpTxExpanded = ref(false);
const tpTxSpuriousExpanded = ref(false);

const systemSubItems = [
  { key: 'transmitter', label: 'Transmitter' },
  { key: 'receiver', label: 'Receiver' },
  { key: 'transponder', label: 'Transponder' },
];

const specificationSubItems = [
  { key: 'specifications_power', label: 'Power' },
  { key: 'specifications_frequency', label: 'Frequency' },
  { key: 'specifications_modulation_index', label: 'Modulation Index' },
  { key: 'specifications_spurious', label: 'Spurious' },
  { key: 'specifications_command_threshold', label: 'Command Threshold' },
  { key: 'specifications_ranging_threshold', label: 'Ranging Threshold' },
];

const isSpecificationsActive = computed(() =>
  specificationSubItems.some((sub) => sub.key === props.activeSection),
);

const isTestProfilesActive = computed(() =>
  ['tp_tx_spurious_bands', 'tp_tx_spurious_profile', 'test_profiles_receiver', 'test_profiles_transponder'].includes(props.activeSection),
);

const isTestSystemsActive = computed(() =>
  ['test_systems_instruments', 'test_systems_tsm_paths', 'test_systems_power_meter'].includes(props.activeSection),
);

const isOnBoardLossesActive = computed(() =>
  ['on_board_losses_transmitter', 'on_board_losses_receiver'].includes(props.activeSection),
);

const isCalibrationActive = computed(() =>
  ['calibration_transmitter', 'calibration_receiver', 'calibration_transponder'].includes(props.activeSection),
);

const isTestPlanActive = computed(() =>
  ['test_plan_transmitter', 'test_plan_receiver', 'test_plan_transponder'].includes(props.activeSection),
);

const topLevelItems = [
  { key: 'configurations', label: 'Configurations' },
  { key: 'tm_tc', label: 'TM TC' },
  { key: 'env_data', label: 'ENV Data' },
];

function select(section: string) {
  emit('select', section);
}

function expandAll() {
  systemsExpanded.value = true;
  specificationsExpanded.value = true;
  testProfilesExpanded.value = true;
  testSystemsExpanded.value = true;
  onBoardLossesExpanded.value = true;
  calibrationExpanded.value = true;
  testPlanExpanded.value = true;
  tpTxExpanded.value = true;
  tpTxSpuriousExpanded.value = true;
}

function collapseAll() {
  systemsExpanded.value = false;
  specificationsExpanded.value = false;
  testProfilesExpanded.value = false;
  testSystemsExpanded.value = false;
  onBoardLossesExpanded.value = false;
  calibrationExpanded.value = false;
  testPlanExpanded.value = false;
  tpTxExpanded.value = false;
  tpTxSpuriousExpanded.value = false;
}
</script>

<style scoped>
.db-sidebar {
  background-color: #0d1b2e;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #1e3050;
  padding-top: 0.75rem;
  user-select: none;
}

/* Top +/− controls */
.sidebar-controls {
  display: flex;
  gap: 0.5rem;
  padding: 0.6rem 0.8rem;
  border-bottom: 1px solid #1e3050;
}

.ctrl-btn {
  background: transparent;
  border: 1px solid #334155;
  color: #94a3b8;
  width: 1.8rem;
  height: 1.8rem;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.2s, color 0.2s;
}

.ctrl-btn:hover {
  border-color: #22d3ee;
  color: #22d3ee;
}

/* Navigation */
.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  padding: 0.4rem 0;
}

.nav-item,
.nav-group-header {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.55rem 1rem;
  cursor: pointer;
  color: #94a3b8;
  font-size: 0.875rem;
  transition: background 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}

.nav-item:hover,
.nav-group-header:hover {
  background: #132035;
  color: #e2e8f0;
}

.nav-item.active,
.nav-group-header.active {
  color: #22d3ee;
  border-left-color: #22d3ee;
  background: #0f2744;
}

.nav-icon {
  flex-shrink: 0;
  font-size: 0.9rem;
  color: #22d3ee;
}

.nav-label {
  flex: 1;
}

.nav-chevron {
  font-size: 0.7rem;
  margin-left: auto;
  color: #64748b;
}

/* Sub-items */
.nav-children {
  overflow: hidden;
}

.nav-child-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.45rem 1rem 0.45rem 2.2rem;
  cursor: pointer;
  color: #94a3b8;
  font-size: 0.85rem;
  transition: background 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}

.nav-child-item:hover {
  background: #132035;
  color: #e2e8f0;
}

.nav-child-item.child-active {
  color: #22d3ee;
  border-left-color: #22d3ee;
  background: #0f2744;
}

.child-icon {
  font-size: 0.75rem;
  color: #22d3ee;
}

/* Level 2 expandable children (no leaf click, has chevron) */
.nav-child-expandable {
  cursor: pointer;
}

/* Level 3 — sub-children wrapper */
.nav-subchildren {
  overflow: hidden;
}

.nav-subchild-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 1rem 0.4rem 3.4rem;
  cursor: pointer;
  color: #94a3b8;
  font-size: 0.83rem;
  transition: background 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}

.nav-subchild-item:hover {
  background: #132035;
  color: #e2e8f0;
}

.nav-subchild-item.subchild-active {
  color: #22d3ee;
  border-left-color: #22d3ee;
  background: #0f2744;
}

/* Level 4 — grandchildren wrapper */
.nav-grandchildren {
  overflow: hidden;
}

.nav-grandchild-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.38rem 1rem 0.38rem 4.6rem;
  cursor: pointer;
  color: #64748b;
  font-size: 0.82rem;
  transition: background 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}

.nav-grandchild-item:hover {
  background: #132035;
  color: #e2e8f0;
}

.nav-grandchild-item.grandchild-active {
  color: #22d3ee;
  border-left-color: #22d3ee;
  background: #0f2744;
}

.grandchild-icon {
  font-size: 0.6rem;
  color: #22d3ee;
}

/* Expand animation */
.expand-enter-active,
.expand-leave-active {
  transition: max-height 0.3s ease, opacity 0.2s;
  max-height: 600px;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

/* ── Light theme overrides ──────────────────────────────────────────────── */
html:not(.dark) .db-sidebar {
  background-color: var(--p-surface-0);
  border-right-color: var(--p-content-border-color);
}
html:not(.dark) .sidebar-controls {
  border-bottom-color: var(--p-content-border-color);
}
html:not(.dark) .ctrl-btn {
  border-color: var(--p-surface-300);
  color: var(--p-text-muted-color);
}
html:not(.dark) .ctrl-btn:hover {
  border-color: var(--p-primary-color);
  color: var(--p-primary-color);
}
html:not(.dark) .nav-item,
html:not(.dark) .nav-group-header,
html:not(.dark) .nav-child-item,
html:not(.dark) .nav-subchild-item,
html:not(.dark) .nav-grandchild-item {
  color: var(--p-text-color);
}
html:not(.dark) .nav-item:hover,
html:not(.dark) .nav-group-header:hover,
html:not(.dark) .nav-child-item:hover,
html:not(.dark) .nav-subchild-item:hover,
html:not(.dark) .nav-grandchild-item:hover {
  background: var(--p-surface-100);
  color: var(--p-text-color);
}
html:not(.dark) .nav-item.active,
html:not(.dark) .nav-group-header.active,
html:not(.dark) .nav-child-item.child-active,
html:not(.dark) .nav-subchild-item.subchild-active,
html:not(.dark) .nav-grandchild-item.grandchild-active {
  background: var(--p-primary-50);
  color: var(--p-primary-color);
  border-left-color: var(--p-primary-color);
}
html:not(.dark) .nav-chevron {
  color: var(--p-text-muted-color);
}
</style>
