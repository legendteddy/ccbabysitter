/* The sky over the kangaroo: the sun by day and the moon by night, where
   the real ones are for the visitor's time zone and clock, and the theme
   switch at the same time. A tiny line beside them gives the time and one
   fact. The sky's arithmetic is done in the page and sends nothing; the
   site keeps only the theme choice (localStorage) and whether the intro
   played in this tab (sessionStorage). */
"use strict";
(function () {
  var RAD = Math.PI / 180, SYN = 29.530588, REF_NEW = Date.UTC(2000, 0, 6, 18, 14);
  var DOT = " \u00b7 ";

  function wrap(m) { return ((m % 1440) + 1440) % 1440; }
  function hm(m) { m = Math.round(wrap(m)) % 1440; return pad(Math.floor(m / 60)) + ":" + pad(m % 60); }
  function pad(n) { return (n < 10 ? "0" : "") + n; }
  function dur(m) { m = Math.round(m); var h = Math.floor(m / 60); return h ? h + "h " + pad(m % 60) + "m" : (m % 60) + "m"; }

  // placeFor turns a time zone name into the place it is named after, or
  // null for a zone the table does not know, such as UTC.
  function placeFor(zone) {
    var zones = window.ccbZones || {}, aliases = window.ccbZoneAliases || {};
    var z = zones[zone] || zones[aliases[zone]];
    return z ? { lat: z[0], lon: z[1] } : null;
  }

  // sunDay is the sun on a day of the year at a place, by the local clock:
  // standard approximations, good to a few minutes. Without a place it is
  // the textbook day.
  function sunDay(doy, place, offsetMin) {
    if (!place) return { rise: 360, set: 1080, maxAlt: 70, kind: "", south: false };
    var decl = 23.44 * Math.sin(2 * Math.PI * (doy - 81) / 365);
    var b = 2 * Math.PI * (doy - 81) / 364, eot = 9.87 * Math.sin(2 * b) - 7.53 * Math.cos(b) - 1.5 * Math.sin(b);
    var noon = 720 - 4 * place.lon - eot + offsetMin;
    var cosH = (Math.sin(-0.833 * RAD) - Math.sin(place.lat * RAD) * Math.sin(decl * RAD)) / (Math.cos(place.lat * RAD) * Math.cos(decl * RAD));
    var out = { rise: 0, set: 0, maxAlt: 90 - Math.abs(place.lat - decl), kind: "", south: place.lat < 0 };
    if (cosH < -1) { out.kind = "midnight sun"; out.rise = noon - 720; out.set = noon + 720; return out; }
    if (cosH > 1) { out.kind = "polar night"; out.rise = noon; out.set = noon; return out; }
    var h0 = Math.acos(cosH) / RAD;
    out.rise = noon - 4 * h0; out.set = noon + 4 * h0;
    return out;
  }

  function moonAge(ms) { return (((ms - REF_NEW) / 86400000) % SYN + SYN) % SYN; }
  function phaseName(a) {
    var f = a / SYN;
    if (f < 0.03 || f > 0.97) return "new moon";
    if (f < 0.22) return "waxing crescent";
    if (f < 0.28) return "first quarter";
    if (f < 0.47) return "waxing gibbous";
    if (f < 0.53) return "full moon";
    if (f < 0.72) return "waning gibbous";
    if (f < 0.78) return "last quarter";
    return "waning crescent";
  }
  function litPercent(a) { return Math.round((1 - Math.cos(2 * Math.PI * a / SYN)) / 2 * 100); }

  // skyWindow is when the body of a theme is up: the sun from sunrise to
  // sunset, the moon from sunset to the next sunrise and 25 minutes more,
  // since it crosses the sky a little slower.
  function skyWindow(theme, sun) {
    if (theme === "light") {
      if (sun.kind === "midnight sun") return { start: 0, len: 1440, always: true };
      if (sun.kind === "polar night") return { never: true };
      return { start: sun.rise, len: sun.set - sun.rise };
    }
    if (sun.kind === "midnight sun") return { never: true };
    if (sun.kind === "polar night") return { start: 0, len: 1440, always: true };
    return { start: sun.set, len: 1440 - (sun.set - sun.rise) + 25 };
  }

  // position is how far across its arc the body is, from 0 rising to 1
  // setting, or null while it waits below the horizon, with the minutes
  // until it rises.
  function position(m, win) {
    if (win.never) return { t: null, untilRise: null };
    var since = wrap(m - win.start);
    if (win.always) return { t: since / win.len, untilRise: null };
    if (since <= win.len) return { t: since / win.len, untilRise: null };
    return { t: null, untilRise: Math.round(1440 - since) };
  }

  // arcPoint is the body's top-left corner on its half ellipse over the
  // kangaroo. A waiting body sits where it will rise, never lower than
  // peeking, so it can always be clicked. East is left in the northern
  // hemisphere and right in the southern.
  function arcPoint(t, theme, sun, g) {
    var minR = Math.min(g.H - 40, g.rooH + 70), maxR = g.H - 40;
    var k = theme === "light" ? Math.max(0, Math.min(1, sun.maxAlt / 70)) : 0.8;
    var rx = 0.42 * g.W, ry = minR + (maxR - minR) * k, tt = t === null ? 0 : t;
    if (sun.south) tt = 1 - tt;
    var x = g.W / 2 - rx * Math.cos(Math.PI * tt), y = g.H - ry * Math.sin(Math.PI * tt);
    y = Math.min(y, g.H - g.size * 0.1);
    return { x: Math.round(x - g.size / 2), y: Math.round(y - g.size / 2) };
  }

  // specialDay names the four days of the year by calendar date, month 1
  // to 12 and day of the month: the equinoxes and solstices as the
  // northern hemisphere has them, swapped in the southern.
  function specialDay(month, dom, south) {
    var days = [[3, 20, south ? "autumn equinox" : "spring equinox"], [6, 21, south ? "shortest day" : "longest day"],
      [9, 22, south ? "spring equinox" : "autumn equinox"], [12, 21, south ? "longest day" : "shortest day"]];
    for (var i = 0; i < days.length; i++) if (month === days[i][0] && dom === days[i][1]) return days[i][2];
    return "";
  }

  // captionText is the line beside the body: the time, then one fact. When
  // the theme and the sky disagree the fact is simply when it rises.
  function captionText(m, theme, sun, pos, age, month, dom) {
    var fact;
    if (theme === "light") {
      if (sun.kind) fact = sun.kind;
      else if (pos.t === null) fact = "sunrise in " + dur(pos.untilRise);
      else fact = specialDay(month, dom, sun.south) || "sunset " + hm(sun.set);
    } else if (pos.t === null) {
      fact = sun.kind === "midnight sun" ? "midnight sun" : "moonrise in " + dur(pos.untilRise);
    } else {
      var n = phaseName(age);
      fact = n === "full moon" || n === "new moon" ? n : n + ", " + litPercent(age) + "%";
    }
    return hm(m) + DOT + fact;
  }

  // placeCaption puts the line in the first of a few places near the body
  // that stays inside the sky and touches no box: under it, over it, then
  // as two short lines on its outer side, over it and beside it, then the
  // two lines higher and higher over it.
  function placeCaption(text, body, sky, boxes, measure) {
    var two = text.split(DOT).join("\n"), m1 = measure(text), m2 = measure(two);
    var cx = body.x + body.size / 2, left = cx < sky.W / 2;
    function cl(x, w) { return Math.max(4, Math.min(sky.W - w - 4, x)); }
    function outer(w) { return left ? cl(body.x, w) : cl(body.x + body.size - w, w); }
    var tries = [
      [text, cl(cx - m1.w / 2, m1.w), body.y + body.size + 8, m1],
      [text, cl(cx - m1.w / 2, m1.w), body.y - m1.h - 8, m1],
      [two, outer(m2.w), body.y - m2.h - 8, m2],
      [two, left ? cl(body.x + body.size + 10, m2.w) : cl(body.x - m2.w - 10, m2.w), body.y + body.size / 2 - m2.h / 2, m2]
    ];
    for (var up = 6; up < sky.H; up += 6) tries.push([two, outer(m2.w), body.y - m2.h - 8 - up, m2]);
    function clear(x, y, m) {
      if (x < 4 || x + m.w > sky.W - 4 || y < 4 || y + m.h > sky.H - 3) return false;
      for (var i = 0; i < boxes.length; i++) {
        var b = boxes[i];
        if (x < b[2] && x + m.w > b[0] && y < b[3] && y + m.h > b[1]) return false;
      }
      return true;
    }
    for (var k = 0; k < tries.length; k++) {
      var c = tries[k];
      if (clear(c[1], c[2], c[3])) return { text: c[0], x: Math.round(c[1]), y: Math.round(c[2]), right: c[0] === two && !left };
    }
    return { text: two, x: 4, y: 4, right: false };
  }

  // localDay is the day of the year of a date by its own calendar, so a
  // night with a clock change still has the right day.
  function localDay(d) { return Math.round((Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()) - Date.UTC(d.getFullYear(), 0, 0)) / 86400000); }
  // noonOffset is the clock's offset from UTC at noon of a date's own day,
  // in minutes east of UTC, so the sun of a day with a clock change is
  // reckoned with the clock the day's daylight is read on.
  function noonOffset(d) { return -new Date(d.getFullYear(), d.getMonth(), d.getDate(), 12).getTimezoneOffset(); }

  window.ccbSky = { localDay: localDay, noonOffset: noonOffset, placeFor: placeFor, sunDay: sunDay, moonAge: moonAge, phaseName: phaseName, litPercent: litPercent,
    skyWindow: skyWindow, position: position, arcPoint: arcPoint, captionText: captionText, placeCaption: placeCaption };
  if (typeof document === "undefined" || !document.querySelector) return;
  var host = document.querySelector("[data-sky]");
  if (!host) return;
  var NS = "http://www.w3.org/2000/svg", root = document.documentElement;
  var SUN = ["....r....", ".r.....r.", "...ooo...", "..ooooo..", "r.ooooo.r", "..ooooo..", "...ooo...", ".r.....r.", "....r...."];
  var DISC = ["...xxx...", ".xxxxxxx.", ".xxxxxxx.", "xxxxxxxxx", "xxxxxxxxx", "xxxxxxxxx", ".xxxxxxx.", ".xxxxxxx.", "...xxx..."];
  var zone = "";
  try { zone = Intl.DateTimeFormat().resolvedOptions().timeZone || ""; } catch (e) { /* no time zone: the textbook day */ }
  var place = placeFor(zone);
  if (!place || typeof place.lat !== "number") place = null;
  var orb = document.createElement("button"), cap = document.createElement("div");
  orb.type = "button"; orb.className = "orb";
  cap.className = "cap"; cap.setAttribute("aria-hidden", "true");
  host.appendChild(orb); host.appendChild(cap);

  function reduced() { try { return window.matchMedia("(prefers-reduced-motion: reduce)").matches; } catch (e) { return false; } }
  function systemDark() { try { return window.matchMedia("(prefers-color-scheme: dark)").matches; } catch (e) { return false; } }
  function theme() { return root.getAttribute("data-theme") || (systemDark() ? "dark" : "light"); }

  function pixels(rows, fill) {
    var svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", "0 0 9 9"); svg.setAttribute("aria-hidden", "true");
    for (var y = 0; y < 9; y++) for (var x = 0; x < 9; x++) {
      var colour = fill(rows[y][x], x, y);
      if (!colour) continue;
      var r = document.createElementNS(NS, "rect");
      r.setAttribute("x", x); r.setAttribute("y", y); r.setAttribute("width", 1.02); r.setAttribute("height", 1.02);
      r.setAttribute("fill", colour); svg.appendChild(r);
    }
    return svg;
  }
  // The moon's lit part, pixel by pixel: waxing lights the right side and
  // waning the left, mirrored south of the equator. The unlit part is a
  // faint disc, so even a new moon can be found and clicked.
  function drawMoon(age, south) {
    var th = 2 * Math.PI * age / SYN;
    return pixels(DISC, function (c, x, y) {
      if (c !== "x") return "";
      var nx = (x - 4) / 4.5, ny = (y - 4) / 4.5, s = Math.sqrt(Math.max(0, 1 - ny * ny));
      if (south) nx = -nx;
      var lit = th < Math.PI ? nx > Math.cos(th) * s : nx < -Math.cos(th) * s;
      return lit ? "#ebe4d3" : "#2b2824";
    });
  }
  function draw(now) {
    var dark = theme() === "dark";
    orb.setAttribute("aria-label", dark ? "Switch to light theme" : "Switch to dark theme");
    var art = dark ? drawMoon(moonAge(now.getTime()), !!(place && place.lat < 0))
      : pixels(SUN, function (c) { return c === "o" ? "#e09a2c" : c === "r" ? "#e9b663" : ""; });
    while (orb.firstChild) orb.removeChild(orb.firstChild);
    orb.appendChild(art);
  }
  // kangarooBoxes is every cell any of her poses draws, in sky pixels, so
  // the line stays clear of her whatever she is doing.
  function kangarooBoxes() {
    var roo = host.querySelector(".roo"), art = window.ccbArt;
    if (!roo || !art || !art.footprint) return [];
    var hr = host.getBoundingClientRect(), rr = roo.getBoundingClientRect(), px = rr.width / 16;
    return art.footprint().map(function (c) {
      var l = rr.left - hr.left + c[0] * px, t = rr.top - hr.top + c[1] * px;
      return [l - 3, t - 3, l + px + 3, t + px + 3];
    });
  }
  function measure(text) { cap.textContent = text; return { w: cap.offsetWidth, h: cap.offsetHeight }; }

  function where(now, th) {
    var doy = localDay(now), m = now.getHours() * 60 + now.getMinutes();
    var sun = sunDay(doy, place, noonOffset(now));
    var pos = position(m, skyWindow(th, sun)), size = orb.offsetWidth;
    var roo = host.querySelector(".roo");
    var g = { W: host.clientWidth, H: host.clientHeight, rooH: roo ? roo.offsetHeight : 0, size: size };
    var p = arcPoint(pos.t, th, sun, g);
    return { p: p, g: g, text: captionText(m, th, sun, pos, moonAge(now.getTime()), now.getMonth() + 1, now.getDate()) };
  }
  function show(w) {
    orb.style.transform = "translate(" + w.p.x + "px," + w.p.y + "px)";
    var c = placeCaption(w.text, { x: w.p.x, y: w.p.y, size: w.g.size }, { W: w.g.W, H: w.g.H }, kangarooBoxes(), measure);
    cap.textContent = c.text;
    cap.style.textAlign = c.right ? "right" : "left";
    cap.style.transform = "translate(" + c.x + "px," + c.y + "px)";
  }
  var busy = false;
  function update() { if (busy) return; var now = new Date(); draw(now); show(where(now, theme())); }

  orb.addEventListener("click", function () {
    var switchTheme = window.ccbSite && window.ccbSite.switchTheme;
    if (!switchTheme || busy) return;
    if (reduced()) { switchTheme(); update(); return; }
    busy = true;
    var w = where(new Date(), theme());
    cap.classList.add("fade");
    orb.classList.add("glide");
    orb.style.transform = "translate(" + w.p.x + "px," + (w.g.H + 10) + "px)";
    function finish() {
      busy = false;
      try { orb.classList.remove("glide"); update(); } finally { cap.classList.remove("fade"); }
    }
    setTimeout(function () {
      try {
        switchTheme();
        var now = new Date(), next = where(now, theme());
        draw(now);
        orb.classList.remove("glide");
        orb.style.transform = "translate(" + next.p.x + "px," + (next.g.H + 10) + "px)";
        orb.getBoundingClientRect();
        orb.classList.add("glide");
        orb.style.transform = "translate(" + next.p.x + "px," + next.p.y + "px)";
      } catch (e) {
        finish();
        return;
      }
      setTimeout(finish, 650);
    }, 620);
  });

  update();
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(update);
  // The line shows the minute, so it ticks at each minute boundary of the
  // clock and re-arms itself from there.
  function tick() { update(); setTimeout(tick, 60000 - Date.now() % 60000 + 20); }
  setTimeout(tick, 60000 - Date.now() % 60000 + 20);
  document.addEventListener("visibilitychange", function () { if (!document.hidden) update(); });
  window.addEventListener("resize", update);
  try { window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", update); } catch (e) { /* older browser: the next minute catches up */ }
})();
