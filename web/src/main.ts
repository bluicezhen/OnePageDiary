import { createApp } from "vue";
import App from "./App.vue";
import "./styles.css";

const app = createApp(App);
app.mount("#app");

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {
      // Service worker failure should not block app usage.
    });
  });
}
