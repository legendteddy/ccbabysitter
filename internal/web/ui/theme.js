/* Owns the theme. The setting is dark, light or auto, and the page is only
   ever painted dark or light: auto is turned into one of the two here, from
   the light or dark mode of the computer showing the page, and follows it
   when it changes. The last setting this browser saw is applied before the
   page paints, so a reload does not flash the wrong palette. The real
   setting arrives with the first view and replaces it through apply. */
(function () {
  "use strict";
  var light = null;
  try { light = window.matchMedia("(prefers-color-scheme: light)"); } catch (err) { /* no media queries: dark it is */ }
  var setting = "dark";

  /* A setting saved as Follow system before Light existed is Auto now. */
  function known(value) {
    if (value === "light" || value === "auto") { return value; }
    if (value === "system") { return "auto"; }
    return "dark";
  }

  function paint() {
    var look = setting === "auto" ? (light && light.matches ? "light" : "dark") : setting;
    if (document.documentElement.getAttribute("data-theme") !== look) {
      document.documentElement.setAttribute("data-theme", look);
    }
  }

  function apply(value) {
    setting = known(value);
    try { window.localStorage.setItem("ccb.theme", setting); } catch (err) { /* storage is optional */ }
    paint();
  }

  if (light) {
    if (light.addEventListener) { light.addEventListener("change", paint); } else if (light.addListener) { light.addListener(paint); }
  }
  try { setting = known(window.localStorage.getItem("ccb.theme")); } catch (err) { /* storage can be switched off */ }
  paint();

  window.ccbTheme = { apply: apply };
})();
