/* The page's pixel art: a kangaroo who babysits, in four poses, with the
   baby beside her while all is calm, and one icon for each kind of app a
   session can live in. Every picture is a 16 by 16 grid below, one
   character per pixel and a dot for nothing, turned into inline SVG
   rectangles when the page asks for it. The kangaroo and the icons are
   this program's own drawings; colours come from the stylesheet. */

"use strict";

(function () {

  var NS = "http://www.w3.org/2000/svg";

  /* Mascot pixels: o outline, b body, l belly, d pouch rim, p eye, h the
     baby's hair, s her skin, c her onesie, z the question and the sleep
     marks. H, S and C are the baby standing beside the kangaroo: drawn on a
     layer of their own as h, s and c, which the hop hides while the baby is
     in the air. */
  var POSES = {
    calm: [
      "........o.o.....",
      ".......obobo....",
      ".......obobo....",
      "......obbbboo...",
      "......obbpbbbo..",
      ".......obbbbo...",
      ".......obboo....",
      "......obbbbo....",
      ".....obbllbo....",
      ".....obllbbo.HH.",
      "....obbllbo.HSSH",
      "ooooobbllbo..SS.",
      "bbbbbbbbbbo.CCCC",
      "oooobbbbbbo..CC.",
      "....oooooo..S..S",
      "................"
    ],
    alert: [
      "........o.o.....",
      ".......obobo....",
      ".......obobo....",
      "......obbbboo...",
      "......obbpbbbo..",
      ".......obbbbo...",
      ".......obboo....",
      "......obbbbo....",
      ".....obbhhbo....",
      ".....obsssbo....",
      "....obbddbo.....",
      "ooooobbllbo.....",
      "bbbbbbbbbbo.....",
      "oooobbbbbbo.....",
      "....oooooo......",
      "................"
    ],
    worried: [
      "........o.o.....",
      ".......obobo.zz.",
      ".......obobo..z.",
      "......obbbboo...",
      "......obpbbbboz.",
      ".......obbbbo...",
      ".......obboo....",
      "......obbbbo....",
      ".....obbllbo....",
      ".....obllbbo....",
      "....obbllbo.....",
      "ooooobbllbo.....",
      "bbbbbbbbbbo.....",
      "oooobbbbbbo.....",
      "....oooooo......",
      "................"
    ],
    asleep: [
      "........o.o.....",
      ".......obobo.zz.",
      ".......obobo..z.",
      "......obbbboozz.",
      "......obbbbbbo..",
      ".......obbbbo...",
      ".......obboo....",
      "......obbbbo....",
      ".....obbllbo....",
      ".....obllbbo....",
      "....obbllbo.....",
      "ooooobbllbo.....",
      "bbbbbbbbbbo.....",
      "oooobbbbbbo.....",
      "....oooooo......",
      "................"
    ]
  };

  /* The baby two rows up, shown in turns with the standing baby while she
     hops beside a calm kangaroo. */
  var HOP = [
      "................",
      "................",
      "................",
      "................",
      "................",
      "................",
      "................",
      ".............hh.",
      "............hssh",
      ".............ss.",
      "............cccc",
      ".............cc.",
      "............s..s",
      "................",
      "................",
      "................"
    ];

  /* Icon pixels: o outline, f screen, a mark. Outline and mark take the
     colour of the text around them, so an icon matches its label. */
  var ICONS = {
    desktop: [
      "................",
      "................",
      ".oooooooooooooo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".oooooooooooooo.",
      "......oooo......",
      "......oooo......",
      "....oooooooo....",
      "................",
      "................",
      "................"
    ],
    terminal: [
      "................",
      "................",
      ".oooooooooooooo.",
      ".oooooooooooooo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".ofaffffffffffo.",
      ".offafffffffffo.",
      ".ofaffffffffffo.",
      ".offffaaafffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".oooooooooooooo.",
      "................",
      "................",
      "................"
    ],
    vscode: [
      "................",
      "................",
      ".oooooooooooooo.",
      ".oooooooooooooo.",
      ".offffffffffffo.",
      ".offfffffaffffo.",
      ".offfaffafafffo.",
      ".offafffaffaffo.",
      ".offfafaffafffo.",
      ".offfffaffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".oooooooooooooo.",
      "................",
      "................",
      "................"
    ],
    background: [
      "................",
      "................",
      "................",
      "................",
      "......oooo......",
      ".....offffo.....",
      "...oooffffffo...",
      "..offfffffffoo..",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".offffffffffffo.",
      ".oooooooooooooo.",
      "................",
      "................",
      "................",
      "................"
    ],
    other: [
      "................",
      "................",
      "..oooooooooooo..",
      "..offffffffffo..",
      "..offfaaaafffo..",
      "..offaffffaffo..",
      "..offfffffaffo..",
      "..offfffaafffo..",
      "..offfffaffffo..",
      "..offffffffffo..",
      "..offfffaffffo..",
      "..offffffffffo..",
      "..offffffffffo..",
      "..oooooooooooo..",
      "................",
      "................"
    ]
  };

  /* paint adds one rectangle per run of equal pixels in a row, with the
     class prefix plus the pixel's letter. keep says which letters to draw
     and as what, so one grid can feed more than one layer. */
  function paint(parent, rows, prefix, keep) {
    for (var y = 0; y < rows.length; y++) {
      var row = rows[y];
      var x = 0;
      while (x < row.length) {
        var ch = keep(row.charAt(x));
        if (!ch) { x++; continue; }
        var start = x;
        while (x < row.length && keep(row.charAt(x)) === ch) { x++; }
        var rect = document.createElementNS(NS, "rect");
        rect.setAttribute("x", String(start));
        rect.setAttribute("y", String(y));
        rect.setAttribute("width", String(x - start));
        rect.setAttribute("height", "1");
        rect.setAttribute("class", prefix + ch);
        parent.appendChild(rect);
      }
    }
  }

  function layer(svg, cls) {
    var g = document.createElementNS(NS, "g");
    g.setAttribute("class", cls);
    svg.appendChild(g);
    return g;
  }

  function frame(cls, label) {
    var svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", "0 0 16 16");
    svg.setAttribute("class", cls);
    svg.setAttribute("shape-rendering", "crispEdges");
    if (label) {
      svg.setAttribute("role", "img");
      svg.setAttribute("aria-label", label);
    } else {
      svg.setAttribute("aria-hidden", "true");
    }
    return svg;
  }

  function drawPose(svg, pose) {
    var rows = POSES[pose] || POSES.calm;
    svg.replaceChildren();
    paint(layer(svg, "m-base"), rows, "m-", function (ch) {
      return ch === "." || ch !== ch.toLowerCase() ? "" : ch;
    });
    paint(layer(svg, "m-rest"), rows, "m-", function (ch) {
      return ch !== ch.toLowerCase() ? ch.toLowerCase() : "";
    });
    paint(layer(svg, "m-hop"), HOP, "m-", function (ch) { return ch === "." ? "" : ch; });
    svg.setAttribute("data-pose", POSES[pose] ? pose : "calm");
  }

  /* mascot returns a new kangaroo in the given pose. */
  function mascot(pose, label) {
    var svg = frame("mascot", label);
    drawPose(svg, pose);
    return svg;
  }

  /* setPose redraws a kangaroo only when its pose really changes, so a
     page that renders often does not rebuild it every time. */
  function setPose(svg, pose) {
    if (svg.getAttribute("data-pose") !== pose) { drawPose(svg, pose); }
  }

  /* hop plays the baby's short hop once, beside a calm kangaroo. The stylesheet
     does the moving, and turns it off for anyone who asked for less
     motion. */
  var HOP_MS = 1800;
  function hop(svg) {
    svg.classList.remove("hopping");
    svg.getBoundingClientRect();
    svg.classList.add("hopping");
    window.setTimeout(function () { svg.classList.remove("hopping"); }, HOP_MS);
  }

  /* hostIcon returns a new icon for the kind of app a session lives in. */
  function hostIcon(host) {
    var kind = ICONS[host] ? host : "other";
    var svg = frame("hicon i-" + kind, "");
    paint(svg, ICONS[kind], "i-", function (ch) { return ch === "." ? "" : ch; });
    return svg;
  }

  window.ccbArt = { mascot: mascot, setPose: setPose, hop: hop, hostIcon: hostIcon };

}());
