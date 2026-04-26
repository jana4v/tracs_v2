import Handsontable from "handsontable";
import { createApp, h, nextTick } from "vue";
import PrimeVue from "primevue/config";
import Select from "primevue/select";
// import "primevue/resources/themes/lara-light-blue/theme.css";
// import "primevue/resources/primevue.min.css";
// import "primeicons/primeicons.css";



class CustomSelectEditor extends Handsontable.editors.BaseEditor {
  init() {
    this.container = document.createElement("div");
    this.container.style.position = "absolute";
    this.container.style.zIndex = "9999";
    document.body.appendChild(this.container);
  }

  getValue() {
    console.log("🔄 getValue() returning:", this.selectedValue);
    return this.selectedValue;
  }

  setValue(value) {
    console.log("✅ setValue() called with:", value);
    this.selectedValue = value;
  }

  open() {
    console.log("📌 Editor opened at row:", this.row, "col:", this.col);

    this.vueInstance = createApp({
      data: () => ({
        selectedCity: this.selectedValue,
        cities: [
          { name: "New York", code: "NY" },
          { name: "Rome", code: "RM" },
          { name: "London", code: "LDN" },
          { name: "Istanbul", code: "IST" },
          { name: "Paris", code: "PRS" },
        ],
      }),
      methods: {
        onSelect(newValue) {
          console.log("🆕 User selected:", newValue.name);
          this.selectedCity = newValue.name;
          this.$emit("update:modelValue", newValue);
        },
      },
      template: `
        <Select v-model="selectedCity" 
                :options="cities" 
                optionLabel="name" 
                placeholder="Select a City" 
                @update:modelValue="onSelect"
                class="w-full md:w-56" />
      `,
    });

    this.vueApp = this.vueInstance.mount(this.container);
  }

  hide() {
    console.log("📌 hide() called"); // Debug
    if (this.vueInstance) {
      this.vueInstance.unmount();
      this.vueInstance = null;
    }
    document.body.removeChild(this.container);
  }

  close() {
    console.log("🚪 close() called"); // Debug
    this.hide();
  }
}

export default CustomSelectEditor;