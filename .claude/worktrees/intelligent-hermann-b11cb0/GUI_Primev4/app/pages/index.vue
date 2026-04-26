
<script setup lang="ts">
const side_nav_config = useState<SideNavConfig>("side_nav_config");
side_nav_config.value = {
  show_side_nav: false,
  app_name: "",
  logo_url: "",
  logo_width: "",
  items: [],
};

const spasdacs_url = "http://" + window.location.hostname + "/spasdacs/";
const ttc_url = "http://" + window.location.hostname + "/ttc/";
const papert_url = "http://" + window.location.hostname + "/papert/";
const tracking_url = "http://" + window.location.hostname + "/tracking/";
</script>


<template>
  <div class="bg mt-20">
    <div class="landing_page">
      <div class="mt-10 grid grid-cols-4 gap-4">

        <!-- Internal router-link cards -->
        <router-link to="/tc" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">Telecommand</span>
            <div class="block mb-3 card_text_app">Telecommand</div>
          </div>
        </router-link>

        <!-- External link card -->
        <a :href="spasdacs_url" target="_blank" rel="noopener noreferrer" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">SPASDACS</span>
            <div class="block mb-3 card_text_app">SPASDACS</div>
          </div>
        </a>

        <a href="/tm/" target="_blank" rel="noopener noreferrer" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">Telemetry</span>
            <div class="block mb-3 card_text_app">TM</div>
          </div>
        </a>

        <a :href="ttc_url" target="_blank" rel="noopener noreferrer" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">TRACS</span>
            <div class="block mb-3 card_text_app">TTC</div>
          </div>
        </a>

        <a :href="papert_url" target="_blank" rel="noopener noreferrer" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">PAPERT</span>
            <div class="block mb-3 card_text_app">PAPERT</div>
          </div>
        </a>

        <!-- Another internal link for PAPERT NEW -->
        <router-link to="/papert" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">PAPERT NEW</span>
            <div class="block mb-3 card_text_app">PAPERT</div>
          </div>
        </router-link>

        <a :href="tracking_url" target="_blank" rel="noopener noreferrer" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">Tracking</span>
            <div class="block mb-3 card_text_app">Tracking</div>
          </div>
        </a>

<!--         <router-link to="gisat/phaseCal" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">GISAT</span>
            <div class="block mb-3 card_text_app">Phase Calibration Test</div>
          </div>
        </router-link>

        <router-link to="gisat/tc" class="card-link">
          <div class="card" v-ripple>
            <span class="block mb-3 card_text_group">GISAT</span>
            <div class="block mb-3 card_text_app">Telecommand</div>
          </div>
        </router-link> -->

      </div>
    </div>
  </div>
</template>


<style lang="scss" scoped>

.bg {
  min-height: 90vh;
  background: linear-gradient(120deg, #23243a 0%, #283048 100%);
  background-attachment: fixed;
  transition: background 0.8s;
  position: relative;
  z-index: 1;
  padding-bottom: 32px;
}

.landing_page {
  padding: 2vw 2vw 0 2vw;
  width: 100vw;
  max-width: 1600px;
  margin: 0 auto;
}

.grid {
  gap: 2rem;
  @media (max-width: 900px) {
    grid-template-columns: 1fr 1fr;
  }
  @media (max-width: 600px) {
    grid-template-columns: 1fr;
    gap: 1rem;
  }
}

.card-link {
  text-decoration: none !important;
  color: inherit !important;
  display: block;
  height: 100%;
}

.card {
  background: rgba(30, 34, 45, 0.8);
  border-radius: 18px;
  padding: 2rem 1.5rem 1.5rem 1.5rem;
  box-shadow: 0 10px 30px 0 rgba(40,50,90,0.15), 0 1.5px 4px rgba(60,80,180,0.12);
  border: 2.5px solid transparent;
  backdrop-filter: blur(8px);
  transition:
    border-color 0.33s cubic-bezier(.4,0,.2,1),
    box-shadow 0.33s cubic-bezier(.4,0,.2,1),
    transform 0.22s cubic-bezier(.4,0,.2,1),
    opacity 0.2s;
  opacity: 0.90;
  cursor: pointer;
  position: relative;

  &:hover, &:focus-within {
    border-image: linear-gradient(120deg,#3cc0fa,#32cd7e 60%,#ffcc29) 1;
    box-shadow: 0 16px 32px 0 rgba(51,200,250,0.12), 0 2px 8px rgba(60,180,180,0.13);
    opacity: 1.0;
    transform: scale(1.045);
  }

  &::before {
    content: "";
    position: absolute;
    inset: -3px;
    z-index: 0;
    border-radius: 20px;
    background: linear-gradient(120deg, #3cc0fa 20%, #32cd7e 60%, #ffcc29 100%);
    opacity: 0;
    transition: opacity 0.3s;
    pointer-events: none;
  }
  &:hover::before {
    opacity: 0.45;
    filter: blur(8px);
  }
}

.card_text_group {
  font-size: 1.07rem;
  font-weight: 500;
  letter-spacing: 0.03em;
  color: #a0baff;
  margin-bottom: 0.18em;
  opacity: 0.95;
}

.card_text_app {
  font-size: 1.35rem;
  font-weight: 700;
  color: #ffd347;
  letter-spacing: 0.01em;
  text-shadow: 0 2px 8px rgba(255, 230, 120, 0.09);
}

@media (max-width: 900px) {
  .card { padding: 1.5rem 1rem; }
}
@media (max-width: 600px) {
  .landing_page { padding: 0.5rem; }
  .card { padding: 1.15rem 0.8rem; }
}



</style>