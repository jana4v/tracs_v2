<script setup lang="ts">
const side_nav_config = useState<SideNavConfig>("side_nav_config");
side_nav_config.value = {
  show_side_nav: false,
  app_name: "",
  logo_url: "",
  logo_width: "",
  items: [],
};
</script>

<template>
  <div class="bg mt-20">
    <div class="landing_page">
      <div class="mt-10 grid grid-cols-4 gap-4">
        <div class="">
          <router-link v-slot="{ navigate }" to="/tc" custom>
            <div class="card" v-ripple @click="navigate">
              <div class="flex flex-col" style="width: 30rem; overflow: hidden">
                <span class="text text-4xl text-primary">TC </span>
                <span class="text text-2xl text-secondary">Payload</span>
              </div>
            </div>
          </router-link>
        </div>

        <div class="">
          <router-link v-slot="{ navigate }" to="/papert" custom>
            <div class="card" v-ripple @click="navigate">
              <div class="flex flex-col" style="width: 30rem; overflow: hidden">
                <span class="text text-4xl text-primary">PAPERT</span>
                <span class="text text-2xl text-secondary">Payload</span>
              </div>
            </div>
          </router-link>
        </div>


      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
// .card_text_group {
//   color: rgb(132, 158, 241);
// }

// .card_text_app {
//   color: rgb(181, 199, 18);
//   font-size: x-large;
//   font: bold;
// }

// .bg {
//   height: 100vh;
//   background-image: url("/sc1.gif");
//   background-size: cover; // Ensure the background covers the entire area
//   background-position: center; // Center the background image
//   margin: 0;
//   padding: 0;
// }

// .landing_page {
//   padding: 20px; // Add padding to the landing page container which effectively gives margin to cards
// }

// .grid {
//   display: flex;
//   flex-wrap: wrap;
//   gap: 20px; // This creates space between grid items without using margins on the cards directly
// }

// .cardxx {
//   //background-image: url('/bg1.jpg');
//   // background-size: cover;  // Ensure the background covers the entire area
//   //background-position: center;
//   // background-color: #2f6591;
//   padding: 20px; // Inner padding for card content
//   box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); // Optional: adds shadow for depth
//   border-radius: 8px; // Optional: rounded corners
//   // Remove margin if previously set
//   opacity: 60%;
//   cursor: pointer;
//   &:hover {
//     opacity: 90%;
//   }
// }

.card {
  border: 2px solid transparent;
  padding: 16px;
  transition: border-color 0.3s ease, border-width 0.3s ease;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  opacity: 60%;
  border-width: 3px;
  cursor: pointer;
  &:hover {
    border-color: #3498db;
    border-width: 3px;
    opacity: 100%;
  }
}

// .cardxx {
//   position: relative;
//   padding: 16px;
//   border-radius: 8px;
//   overflow: hidden;
//   background-color: #fff;
//   box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);

//   cursor: pointer;
//   &::before {
//     content: "";
//     position: absolute;
//     top: -2px;
//     left: -2px;
//     right: -2px;
//     bottom: -2px;
//     border-radius: inherit;
//     border: 2px solid transparent;
//     background: linear-gradient(120deg, #3498db, #e74c3c, #f39c12, #3498db);
//     background-size: 300% 300%;
//     animation: rotateBorder 3s linear infinite;
//     z-index: -1;
//   }
//   &:hover::before {
//     animation-play-state: running;
//   }
// }
// @keyframes rotateBorder {
//   0% {
//     background-position: 0% 50%;
//   }
//   100% {
//     background-position: 100% 50%;
//   }
// }
</style>
