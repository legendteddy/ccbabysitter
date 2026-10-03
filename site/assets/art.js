/* The pixel kangaroo with the baby she babysits. Every pose is a 16 by 16
   grid, one character per pixel and a dot for nothing, drawn as SVG
   rectangles. mount(host, animate) draws her calm, and when animate is
   true plays the loop: the baby breaks apart, she looks around, the baby
   comes back in her pouch, then steps out beside her again. The first time
   a browser tab shows the loop, it opens with the baby alone: she looks
   around once, then the kangaroo arrives. A host marked data-roo="baby"
   gets mountBaby instead: the baby of the calm pose on her own, looking
   around now and then. */
"use strict";
(function () {
  var POSES = {
    calm:    ["........o.o.....", ".......obobo....", ".......obobo....", "......obbbboo...", "......obbpbbbo..", ".......obbbbo...", ".......obboo....", "......obbbbo....", ".....obbllbo....", ".....obllbbo.hh.", "....obbllbo.hssh", "ooooobbllbo..ss.", "bbbbbbbbbbo.cccc", "oooobbbbbbo..cc.", "....oooooo..s..s", "................"],
    worried: ["........o.o.....", ".......obobo.zz.", ".......obobo..z.", "......obbbboo...", "......obpbbbboz.", ".......obbbbo...", ".......obboo....", "......obbbbo....", ".....obbllbo....", ".....obllbbo....", "....obbllbo.....", "ooooobbllbo.....", "bbbbbbbbbbo.....", "oooobbbbbbo.....", "....oooooo......", "................"],
    alert:   ["........o.o.....", ".......obobo....", ".......obobo....", "......obbbboo...", "......obbpbbbo..", ".......obbbbo...", ".......obboo....", "......obbbbo....", ".....obbhhbo....", ".....obsssbo....", "....obbddbo.....", "ooooobbllbo.....", "bbbbbbbbbbo.....", "oooobbbbbbo.....", "....oooooo......", "................"]
  };
  // The baby is the h, s and c cells of the calm pose, and her head is
  // those of rows 9 to 11. babyOnly is the calm pose with only her in it.
  function babyAt(y, x) { var ch = POSES.calm[y][x]; return ch === "h" || ch === "s" || ch === "c"; }
  function headAt(y, x) { return y >= 9 && y <= 11 && babyAt(y, x); }
  var BABY_ONLY = POSES.calm.map(function (row, y) {
    return row.split("").map(function (ch, x) { return babyAt(y, x) ? ch : "."; }).join("");
  });
  var COLOR = { o: "#3a2213", b: "#c8834a", l: "#f1d3a6", d: "#9a5a2c", p: "#20140c", z: "#2f7fb8", h: "#5a3a26", s: "#f2c9a2", c: "#9cc4e4" };
  var NS = "http://www.w3.org/2000/svg";
  var INTRO = "ccb.site.intro";
  // The intro, in ms from load: the baby turns her head from LOOK, the
  // kangaroo's pixels gather from ARRIVE over SPREAD, and once the last
  // has finished its .9s move in the stylesheet the loop takes over.
  var LOOK = 400, ARRIVE = 2500, SPREAD = 1200, SETTLED = ARRIVE + SPREAD + 900;

  function reduced() {
    try { return window.matchMedia("(prefers-reduced-motion: reduce)").matches; } catch (e) { return false; }
  }

  // firstView reports whether this browser tab has not seen the intro yet,
  // and notes that it now has. Storage that is missing or throws counts as
  // seen, so a tab that cannot remember the intro is not shown it on every
  // load.
  function firstView() {
    try {
      var store = window.sessionStorage;
      if (store.getItem(INTRO)) return false;
      store.setItem(INTRO, "1");
      return true;
    } catch (e) { return false; }
  }

  // lookAround turns a head group a pixel left, then a pixel right, then
  // back to the middle, starting delay ms from now. Each turn takes .25s in
  // the stylesheet and holds for .6s. A turn that falls while the page is in
  // a background tab is not made; the return to the middle always is.
  function lookAround(head, delay) {
    function turn(px) { head.style.transform = px ? "translate(" + px + "px,0)" : ""; }
    setTimeout(function () { if (!document.hidden) turn(-1); }, delay);
    setTimeout(function () { if (!document.hidden) turn(1); }, delay + 850);
    setTimeout(function () { turn(0); }, delay + 1700);
  }

  function mount(host, animate) {
    var svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", "0 0 16 16");
    svg.setAttribute("aria-hidden", "true");
    // The baby's head cells sit in a group of their own, as in mountBaby,
    // so the intro can turn it.
    var head = document.createElementNS(NS, "g");
    head.setAttribute("class", "head");
    var rects = [];
    for (var y = 0; y < 16; y++) {
      for (var x = 0; x < 16; x++) {
        var r = document.createElementNS(NS, "rect");
        r.setAttribute("x", x); r.setAttribute("y", y);
        r.setAttribute("width", 1.02); r.setAttribute("height", 1.02);
        r.style.opacity = 0;
        (headAt(y, x) ? head : svg).appendChild(r); rects.push(r);
      }
    }
    svg.appendChild(head);
    host.appendChild(svg);
    var current = null;
    function at(pose, i) { return (pose === "babyOnly" ? BABY_ONLY : POSES[pose])[Math.floor(i / 16)][i % 16]; }
    function rnd(a, b) { return a + Math.random() * (b - a); }
    // Leaving pixels drift off and fade, arriving ones gather from a short
    // way off, the rest change colour, each at its own moment.
    // An instant change (the first drawing, or one due while the page is in
    // a background tab) sets no timers at all, so a still kangaroo really is
    // still, and turns the transitions off while it draws, so the first
    // paint is already the finished picture.
    function to(pose, spread, instant) {
      function later(fn) { setTimeout(fn, rnd(0, spread)); }
      var drawn = [];
      rects.forEach(function (r, i) {
        var was = current ? at(current, i) : ".", now = at(pose, i);
        if (was === now) return;
        if (instant) {
          r.style.transition = "none";
          if (now !== ".") r.style.fill = COLOR[now];
          r.style.transform = "none";
          r.style.opacity = now === "." ? 0 : 1;
          drawn.push(r);
        } else if (now === ".") {
          later(function () { r.style.transform = "translate(" + rnd(2, 6) + "px," + rnd(-0.5, 0.5) + "px)"; r.style.opacity = 0; });
        } else if (was === ".") {
          r.style.transition = "none";
          r.style.fill = COLOR[now];
          r.style.transform = "translate(" + rnd(-3, 3) + "px," + rnd(-2, 0.5) + "px)";
          void r.getBoundingClientRect();
          r.style.transition = "";
          later(function () { r.style.transform = "none"; r.style.opacity = 1; });
        } else {
          later(function () { r.style.fill = COLOR[now]; });
        }
      });
      if (drawn.length) {
        void svg.getBoundingClientRect();
        drawn.forEach(function (r) { r.style.transition = ""; });
      }
      current = pose;
    }
    // While the page is in a background tab a cycle plays nothing and
    // only waits for the next one, so she is calm when the visitor returns.
    function loop() {
      if (!document.hidden) {
        setTimeout(function () { to("worried", 900); }, 4000);
        setTimeout(function () { to("alert", 1100); }, 6600);
        setTimeout(function () { to("calm", 1100); }, 12000);
      }
      setTimeout(loop, 15200);
    }
    var still = !animate || reduced();
    if (still || !firstView()) {
      to("calm", 0, true);
      if (!still) loop();
      return;
    }
    // The intro: the baby alone, where she stands in the calm pose, looks
    // around, then the kangaroo gathers around her. A step due while the
    // page is hidden finishes at once, so the kangaroo is there when the
    // visitor comes back, and the loop starts on time either way.
    to("babyOnly", 0, true);
    lookAround(head, LOOK);
    setTimeout(function () { if (document.hidden) to("calm", 0, true); else to("calm", SPREAD); }, ARRIVE);
    setTimeout(loop, SETTLED);
  }

  // mountBaby draws only the baby of the calm pose, her h, s and c cells.
  // The viewBox frames just her (columns 12 to 15, rows 9 to 14), so the
  // host's size sets the size of a pixel. Her head, the h and s cells of
  // rows 9 to 11, sits in a group of its own, and every 5 to 7 seconds she
  // looks around with it.
  function mountBaby(host) {
    var svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", "12 9 4 6");
    svg.setAttribute("aria-hidden", "true");
    var head = document.createElementNS(NS, "g");
    head.setAttribute("class", "head");
    svg.appendChild(head);
    POSES.calm.forEach(function (row, y) {
      for (var x = 0; x < row.length; x++) {
        if (!babyAt(y, x)) continue;
        var r = document.createElementNS(NS, "rect");
        r.setAttribute("x", x); r.setAttribute("y", y);
        r.setAttribute("width", 1.02); r.setAttribute("height", 1.02);
        r.style.fill = COLOR[row[x]];
        (headAt(y, x) ? head : svg).appendChild(r);
      }
    });
    host.appendChild(svg);
    if (reduced()) return;
    function wait() { return 5000 + Math.random() * 2000; }
    // As with the kangaroo, a look that falls while the page is in a
    // background tab is skipped, and only the next one is waited for.
    function look() {
      if (!document.hidden) lookAround(head, 0);
      setTimeout(look, wait());
    }
    setTimeout(look, wait());
  }

  // footprint lists every cell any pose ever draws, so something placed
  // beside the kangaroo can stay clear of her in all of them.
  function footprint() {
    var out = [], seen = {};
    Object.keys(POSES).forEach(function (name) {
      POSES[name].forEach(function (row, y) {
        for (var x = 0; x < row.length; x++) {
          if (row[x] !== "." && !seen[x + "," + y]) { seen[x + "," + y] = true; out.push([x, y]); }
        }
      });
    });
    return out;
  }

  window.ccbArt = { mount: mount, mountBaby: mountBaby, poses: POSES, colors: COLOR, footprint: footprint };
  if (typeof document === "undefined") return;
  var hosts = document.querySelectorAll("[data-roo]");
  for (var i = 0; i < hosts.length; i++) {
    var mode = hosts[i].getAttribute("data-roo");
    if (mode === "baby") mountBaby(hosts[i]);
    else mount(hosts[i], mode === "animate");
  }
})();
