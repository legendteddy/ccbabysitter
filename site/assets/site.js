/* The page's small behaviours: the light and dark switch, the environment
   switch of an install command, and copying a command. Each one works
   without the others. */
"use strict";
(function () {
  var KEY = "ccb.site.theme";
  var ENVS = [
    { name: "macOS &amp; Linux", prompt: "$", cmd: "curl -fsSL https://ccbabysitter.dev/install.sh | sh" },
    { name: "Windows", prompt: "&gt;", cmd: "irm https://ccbabysitter.dev/install.ps1 | iex" },
    { name: "Go", prompt: "$", cmd: "go install ccbabysitter.dev/ccbabysitter/cmd/ccbabysitter@latest" }
  ];

  function readTheme(storage) {
    try { var v = storage.getItem(KEY); return v === "light" || v === "dark" ? v : ""; } catch (e) { return ""; }
  }
  function writeTheme(storage, v) {
    try { storage.setItem(KEY, v); } catch (e) { /* the choice lasts for this visit only */ }
  }
  function nextTheme(current, systemDark) {
    var dark = current ? current === "dark" : systemDark;
    return dark ? "light" : "dark";
  }
  // chooseTheme is the theme after one click. A choice equal to the
  // system's is not kept, so the page follows the system again from then
  // on, including when the system changes.
  function chooseTheme(current, systemDark) {
    var theme = nextTheme(current, systemDark), system = systemDark ? "dark" : "light";
    return theme === system ? { theme: theme, attr: "", store: "" } : { theme: theme, attr: theme, store: theme };
  }
  function commandFor(i) { return ENVS[((i % ENVS.length) + ENVS.length) % ENVS.length]; }

  // copyText copies with the clipboard API when it is there and allowed,
  // and otherwise selects the text so the visitor can copy it by hand.
  function copyText(text, doc, nav, selectNode) {
    function fallback() {
      try {
        if (selectNode && doc.createRange && doc.getSelection) {
          var range = doc.createRange(); range.selectNodeContents(selectNode);
          var sel = doc.getSelection(); sel.removeAllRanges(); sel.addRange(range);
        }
      } catch (e) { /* nothing more to do */ }
      return false;
    }
    try {
      if (nav && nav.clipboard && nav.clipboard.writeText) {
        return nav.clipboard.writeText(text).then(function () { return true; }, fallback);
      }
    } catch (e) { /* fall through */ }
    return { then: function (f) { return f(fallback()); } };
  }

  window.ccbSite = { readTheme: readTheme, writeTheme: writeTheme, nextTheme: nextTheme, chooseTheme: chooseTheme, commandFor: commandFor, copyText: copyText };
  if (typeof document === "undefined" || !document.querySelector) return;

  var root = document.documentElement;

  // switchTheme applies one click of the theme switch to the page and to
  // storage, and returns the theme the page now shows.
  function switchTheme() {
    var sys = false;
    try { sys = window.matchMedia("(prefers-color-scheme: dark)").matches; } catch (e) { /* no preference: light */ }
    var r = chooseTheme(root.getAttribute("data-theme") || "", sys);
    if (r.attr) root.setAttribute("data-theme", r.attr); else root.removeAttribute("data-theme");
    try {
      if (r.store) writeTheme(window.localStorage, r.store); else window.localStorage.removeItem(KEY);
    } catch (e) { /* the choice lasts for this visit only */ }
    return r.theme;
  }
  window.ccbSite.switchTheme = switchTheme;

  var themeButton = document.querySelector("[data-theme-switch]");
  if (themeButton) themeButton.addEventListener("click", function () { switchTheme(); });

  var start = /Win/.test((navigator && navigator.platform) || "") ? 1 : 0;
  Array.prototype.forEach.call(document.querySelectorAll("[data-env-switch]"), function (sw) {
    var line = document.getElementById(sw.getAttribute("data-env-switch"));
    if (!line) return;
    var k = start;
    function show() {
      var e = commandFor(k);
      sw.querySelector("[data-env-name]").innerHTML = e.name;
      line.querySelector("[data-prompt]").innerHTML = e.prompt;
      line.querySelector("[data-cmd]").textContent = e.cmd;
    }
    sw.addEventListener("click", function () { k += 1; show(); });
    if (k !== 0) show();
  });

  Array.prototype.forEach.call(document.querySelectorAll("[data-copy]"), function (btn) {
    btn.addEventListener("click", function () {
      var cmd = btn.querySelector("[data-cmd]");
      var label = btn.querySelector("[data-copy-label]");
      copyText(cmd.textContent, document, navigator, cmd).then(function (ok) {
        if (!label) return;
        label.textContent = ok ? "Copied" : "Selected";
        setTimeout(function () { label.textContent = "Copy"; }, 1300);
      });
    });
  });
})();
