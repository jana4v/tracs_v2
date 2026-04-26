<script setup lang="ts">
import { initMenu, wamp_topic } from "@/composables/papert/SideNav";

definePageMeta({
  title: "PAPERT Application",
});

initMenu(0);

const sub_system_names = ref([]);
const selected_sub_system_names = ref([]);
const test_parameter_names = ref([]);
const selectedParameters = ref([]);
const selected_config_group_names = ref([]);
const config_group_names = ref([]);
const selected_cfgs_per_plot = ref({ v: 8 });
const selected_config_numbers = ref("");
const test_plans = ref([]);
const selected_test_plan = ref();
const test_phase_names = ref([]);
const selected_test_phase_names = ref();
const cfgs_per_plot_options = ref([]);
const trend_plot_layout_options = ref([]);
const selected_trend_plot_layout = ref({ v: [3, 3] });
const form_error = ref([]);

const pptx = ref();
const pptxUrl = "http://" + window.location.hostname + "/file_n/results.pptx";
const papertExecutionStatusStore = usePapertExecutionStatusStore();

useAPIFetch(`/papert/get_home_page_ui_data`, { method: "GET" }).then(
  async (res) => {
    if (res.error == null && res.data) {
      console.log(res.data);
      cfgs_per_plot_options.value = res.data.cfgs_per_plot.map((opt: any) => ({
        v: opt,
      }));
      trend_plot_layout_options.value = res.data.plot_layout.map(
        (opt: any) => ({ v: opt })
      );
      sub_system_names.value = res.data.sub_system_names.map((opt: any) => ({
        v: opt,
      }));
      selected_sub_system_names.value = [{ v: "TTC" }];
      test_plans.value = res.data.test_plan_names.map((opt) => ({ v: opt }));
      test_phase_names.value = res.data.test_phase_names.map((opt) => ({
        v: opt,
      }));
      config_group_names.value = res.data.cfg_group_names.map((opt) => ({
        v: opt,
      }));
      test_parameter_names.value = res.data.test_parameter_names;
    } else {
      let msg = {
        summary: "Failed to Get UI Data from Backend",
        status: `Failed to GetUI Data from Backend Error:${res.error.data.detail}`,
        progress: "0",
      };
      await publishToWampTopic(msg, wamp_topic);
    }
  }
);

// rpc("com.papert.get_home_page_ui_data",[]).then(
//   async (res) => {
//     console.log(res);
//     if (res?.error == null) {
//       cfgs_per_plot_options.value = res?.data.cfgs_per_plot.map((opt: any) => ({ v: opt }));
//       trend_plot_layout_options.value = res?.data.plot_layout.map((opt: any) => ({ v: opt }));
//       sub_system_names.value = res?.data.sub_system_names.map((opt: any) => ({ v: opt }));
//       selected_sub_system_names.value = sub_system_names.value;
//       test_plans.value = res?.data.test_plan_names.map((opt) => ({ v: opt }));
//       test_phase_names.value = res?.data.test_phase_names.map((opt) => ({ v: opt }));
//       config_group_names.value = res?.data.cfg_group_names.map((opt) => ({ v: opt }));
//       test_parameter_names.value = res?.data.test_parameter_names;

//     } else {
//       let msg = {
//         summary: "Failed to Get UI Data from Backend",
//         status: `Error:${res.error}`,
//         progress: "0",
//       };
//       wamp_publish(wamp_topic,[],msg);
//     }
//   }
// );

const sub_system_selection_changed = () => {
  let selected_sub_system_data = selected_sub_system_names.value.map(
    (item) => item.v
  );
  useAPIFetch(`/papert/get_parameter_names`, {
    method: "POST",
    body: selected_sub_system_data,
  }).then(async (res) => {
    if (res.error == null && res.data) {
      test_parameter_names.value = res.data;
    } else {
      let msg = {
        summary: "Failed to Get Test Parameter Names from Database",
        status: `Failed to Get Test Parameter Names from Database.Error:${res.error.data.detail}`,
        progress: "0",
      };
      await publishToWampTopic(msg, wamp_topic);
    }
  });

  useAPIFetch(`/papert/get_test_phase_names`, {
    method: "POST",
    body: selected_sub_system_data,
  }).then(async (res) => {
    if (res.error == null && res.data) {
      test_phase_names.value = res.data.map((opt) => ({ v: opt }));
    } else {
      let msg = {
        summary: "Failed to Get Test Parameter Names from Database",
        status: `Failed to Get Test Parameter Names from Database.Error:${res.error.data.detail}`,
        progress: "0",
      };
      await publishToWampTopic(msg, wamp_topic);
    }
  });
};

const generate_ppt = async () => {
  form_error.value = [];
  let body = {
    satellite_name: "",
    config_numbers_str: selected_config_numbers.value,
    config_groups: selected_config_group_names.value,
    test_paramater_titles: selectedParameters.value,
    test_phases: selected_test_phase_names.value,
    number_of_configs_per_chart: selected_cfgs_per_plot.value.v,
    plot_layout: selected_trend_plot_layout.value.v,
    normalize_data: true,
    skip_cfg_if_no_data: true,
    plot_cfgs: [],
    report_config_name: "",
  };
  let _selected_test_plan = selected_test_plan.value;
  if (_selected_test_plan === undefined) {
    _selected_test_plan = "";
    if (selectedParameters.value.length == 0) {
      form_error.value.push("Select At Least One Parameter...");
    }
    if (
      selected_config_group_names.value.length == 0 &&
      selected_config_numbers.value.length == 0
    ) {
      form_error.value.push("Please Enter Config. Numbers");
    }
    if (selected_config_group_names.value.length > 0) {
      body.config_groups = selected_config_group_names.value.map((i) => i.v);
    }
  } else {
    console.log(_selected_test_plan);
    body.config_numbers_str = "";
    body.config_groups = [];
    body.test_paramater_titles = [];
    body.report_config_name = _selected_test_plan?.name;
  }
  if (selected_test_phase_names.value === undefined) {
    form_error.value.push("Select At Least One Test Phase...");
  } else {
    body.test_phases = selected_test_phase_names.value.map((i) => i.v);
  }
  console.log(form_error.value);

  if (form_error.value.length > 0) {
    return;
  }
  body.test_paramater_titles = selectedParameters.value;
  console.log(body);
  // if (selected_test_plan.value == undefined ) {
  //     body.satellite_name = selected_sub_system_names.value.map(i=>i.name);
  // }

  try {
    useAPIFetch(`/papert/generate_ppt`, { method: "POST", body: body }).then(
      async (res) => {
        if (res?.error == null) {
          pptx.value.click();
        } else {
          let msg = {
            summary: "Failed to Generate PPT",
            status: `Error:${res?.error}`,
            progress: "0",
          };
          wamp_publish(wamp_topic, [], msg);
        }
      }
    );
  } catch (error) {
    console.error("Error generating PPT:", error);
  }
};

// const test_execution_status_store = useTestExecutionStatusStore();

watch(selectedParameters, (newVal) => {
  console.log(newVal);
});
</script>

<template>
  <div class="content min-h-screen w-full">
    <AppName appname="PAPERT"></AppName>
    <div class="grid grid-cols-3 gap-4 pt-1">
      <div class="flex flex-col">
        <label for="TestPlan">
          <h3>Test Plan</h3>
        </label>
        <Select
          v-model="selected_test_plan"
          :autoFilterFocus="true"
          showClear
          :options="test_plans"
          filter
          optionLabel="v"
          placeholder="Select Test Plan"
          class="w-full"
        />
      </div>
      <div v-if="selected_test_plan == undefined" class="flex flex-col">
        <label for="subsystem_names">
          <h3>Sub Systems</h3>
        </label>
        <MultiSelect
          v-model="selected_sub_system_names"
          :autoFilterFocus="true"
          :options="sub_system_names"
          filter
          optionLabel="v"
          placeholder="Select Sub Systems"
          @update:model-value="sub_system_selection_changed"
          display="chip"
          :maxSelectedLabels="15"
          class="w-full"
        />
      </div>

      <div v-if="selected_test_plan == undefined" class="flex flex-col">
        <label for="configGroups">
          <h3>Config Groups</h3>
        </label>
        <MultiSelect
          v-model="selected_config_group_names"
          :autoFilterFocus="true"
          :options="config_group_names"
          filter
          optionLabel="v"
          placeholder="Select Config Groups"
          @update:model-value="sub_system_selection_changed"
          display="chip"
          :maxSelectedLabels="15"
          class="w-full"
        />
      </div>
      <div
        v-if="
          selected_test_plan == undefined &&
          selected_config_group_names?.length == 0
        "
        class="flex flex-col"
      >
        <label for="cfgNumberString">
          <h3>Config Numbers</h3>
        </label>
        <InputText
          id="cfgNumberString"
          v-model="selected_config_numbers"
          aria-describedby="cfgNumberString-help"
        />
      </div>

      <div class="flex flex-col">
        <label for="test_phases">
          <h3>Test Phases</h3>
        </label>
        <MultiSelect
          v-model="selected_test_phase_names"
          :autoFilterFocus="true"
          :options="test_phase_names"
          filter
          optionLabel="v"
          placeholder="Select Test Phases"
          display="chip"
          :maxSelectedLabels="15"
          class="w-full"
        />
      </div>

      <div v-if="selected_test_plan == undefined" class="flex flex-col">
        <label for="Number Configs Per Plot">
          <h3>Configs. Per Plot</h3>
        </label>
        <Select
          v-model="selected_cfgs_per_plot"
          :options="cfgs_per_plot_options"
          optionLabel="v"
          class="w-full"
        />
      </div>

      <div v-if="selected_test_plan == undefined" class="flex flex-col">
        <label for="Trend Plots Layout">
          <h3>Trend Plots Layout</h3>
        </label>
        <Select
          v-model="selected_trend_plot_layout"
          :options="trend_plot_layout_options"
          optionLabel="v"
          class="w-full"
        />
      </div>
    </div>

    <div v-if="selected_test_plan == undefined" class="grid pt-4">
      <div class="col flex flex-column gap-2">
        <CheckBoxSelection
          title="Select Parameters"
          :parameters="test_parameter_names"
          v-model:selectedParams="selectedParameters"
        />
      </div>
    </div>

    <div class="flex mt-4">
      <div class="mr-4">
        <Button
          label="Generate PPT"
          @click="generate_ppt"
          severity="primary"
          raised
        />
      </div>
      <div class="mr-4">
        <Button
          label="Generate Summary Excel"
          @click="generate_ppt"
          severity="primary"
          raised
        />
      </div>

      <div class="mr-4">
        <Button
          label="Generate Doc"
          @click="generate_ppt"
          severity="primary"
          raised
        />
      </div>
    </div>
    <a v-bind:href="pptxUrl" ref="pptx"></a>

    <div class="flex">
      <div class="w-full">
        <ExecutionStatus
          :store="papertExecutionStatusStore"
          :height="'150px'"
        ></ExecutionStatus>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
