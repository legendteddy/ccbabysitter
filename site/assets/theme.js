/* Applies the theme chosen on an earlier visit before the page paints.
   Without a stored choice the stylesheet follows the system setting. */
(function () {
  try {
    var v = window.localStorage.getItem("ccb.site.theme");
    if (v === "light" || v === "dark") document.documentElement.setAttribute("data-theme", v);
  } catch (e) { /* storage blocked: follow the system */ }
})();
