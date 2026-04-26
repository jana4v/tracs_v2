import Handsontable from "handsontable";
import { createApp, h, nextTick } from "vue";
import PrimeVue from "primevue/config";
import Select from "primevue/select";
import HtableSelectDropdown from "./HtableSelectDropdown.vue";

export default class SelectEditor extends Handsontable.editors.BaseEditor {
  init() {
    this.container = document.createElement("div");
    this.container.style.position = "absolute";
    this.container.style.zIndex = "9999";
    document.body.appendChild(this.container);

    this.container.addEventListener("mousedown", (event) => event.stopPropagation());
  }

  getValue() {
    return this.value;
  }

  setValue(value) {
    console.log("✅ setValue() called with:", value);
    this.value = value;
  }

 
  open() {
    console.log("Opening editor...");
  
    const rect = this.TD.getBoundingClientRect();
    this.container.style.top = `${rect.top + window.scrollY}px`;
    this.container.style.left = `${rect.left + window.scrollX}px`;
  
    if (this.vueApp) {
      this.vueApp.unmount();
      this.vueApp = null;
    }
    const handsontableEditor = this; 
    this.vueApp = createApp({
      data: () => ({
        selected: this.value,
        options: [
          { name: "New York", code: "NY" },
          { name: "Rome", code: "RM" },
          { name: "London", code: "LDN" },
          { name: "Istanbul", code: "IST" },
          { name: "Paris", code: "PRS" },
        ],
      }),
      methods: {
        onSelectChange(value) {
          console.log("🔄 Change event:", value);
          this.selected = value;
          handsontableEditor.setValue(value);
          handsontableEditor.finishEditing();
          
        },
      },
      render() {
        return h(HtableSelectDropdown, {
          modelValue: this.selected,
          "onUpdate:modelValue": (event,v) => {  // ✅ Ensure correct binding
                    this.onSelectChange(event);
          },
          options: this.options,
          optionLabel: "name",
          class: "w-full md:w-56",
        });
      },
    });
  
    this.vueApp.use(PrimeVue);
    this.vueApp.component('HtableSelectDropdown', HtableSelectDropdown);
    nextTick(() => {
      this.vueInstance = this.vueApp.mount(this.container);
      console.log("✅ Vue App Mounted Successfully!");
    });
    
    
    if (this.vueInstance) {
      console.log("✅ Vue App Mounted Successfully!");
    } else {
      console.error("❌ Vue App Did Not Mount");
    }
  }
  

  close() {
    if (this.vueApp) {
      this.vueApp.unmount();
      this.vueApp = null;
    }
    document.removeEventListener("mousedown", this.handleOutsideClick);
  }

  focus() {
    nextTick(() => {
      this.container.querySelector(".p-select")?.focus();
    });
  }

  handleOutsideClick = (event) => {
    if (!this.container.contains(event.target)) {
      this.close();
    }
  };
}

// import Handsontable from "handsontable";

Handsontable.editors.registerEditor("htableSelectEditor", SelectEditor);
