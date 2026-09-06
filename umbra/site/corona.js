// The corona renderer.
//
// The eclipse behind the map is drawn, not photographed. There is no raster
// asset on this page and no request for one: the corona is inline SVG built
// from the report's own numbers, so the same umbra.json always produces the
// same corona.
//
// Cost. Filters over a large area are expensive, so everything here is drawn
// once into one fixed-size layer with filterUnits="userSpaceOnUse" and an
// explicit region. No filter primitive is ever animated. The only thing that
// changes after the first paint is opacity on a group.

(function () {
  "use strict";

  var VIEW = { w: 1200, h: 800 };
  var NS = "http://www.w3.org/2000/svg";

  function el(name, attrs) {
    var n = document.createElementNS(NS, name);
    Object.keys(attrs || {}).forEach(function (k) { n.setAttribute(k, attrs[k]); });
    return n;
  }

  // A deterministic generator. Nothing here calls Math.random, because a
  // corona that changed between two loads of the same report would be
  // decoration pretending to be data.
  function hash(text) {
    var h = 2166136261;
    for (var i = 0; i < String(text).length; i++) {
      h ^= String(text).charCodeAt(i);
      h = (h * 16777619) >>> 0;
    }
    return h >>> 0;
  }

  function lcg(seed) {
    var s = (seed >>> 0) || 1;
    return function () {
      s = (1103515245 * s + 12345) >>> 0;
      return s / 4294967296;
    };
  }

  // Filament count reads the light. More lit dependents means a denser corona,
  // because more of the field is in the light. The numbers come from the
  // report's summary, never from a constant chosen to look good.
  function density(data) {
    var s = (data && data.summary) || {};
    var lit = s.lit || 0;
    var total = (s.lit || 0) + (s.penumbra || 0) + (s.umbra || 0) + (s.unknown || 0);
    var sources = Object.keys((data.layout && data.layout.sources) || {}).length || 1;
    var frac = total > 0 ? lit / total : 0;
    return {
      filaments: Math.round(20 + 46 * frac + 3 * sources),
      reach: 0.72 + 0.28 * frac,
      lit: lit,
      total: total,
      frac: frac
    };
  }

  // build draws the corona into svg for a report. moonR is the radius of the
  // occluding disc, which is the changed symbol at the centre of the map.
  function build(svg, data, moonR) {
    if (!svg || !data || !data.layout) { return false; }
    svg.textContent = "";

    var c = data.layout.centre || { x: VIEW.w / 2, y: VIEW.h / 2 };
    var d = density(data);
    var seed = hash((data.checkpoint && data.checkpoint.commit) || "umbra");
    var rand = lcg(seed);
    var R0 = moonR || 132;

    var uid = "cor";
    var defs = el("defs", {});
    svg.appendChild(defs);

    // The body of the corona: three layered radial gradients, hollow at the
    // centre because the moon covers it, falling to nothing well inside the
    // frame so there is no rim.
    // The brightest light sits just outside the limb, so the gradient peaks at
    // the moon's own radius rather than at the centre where nothing is
    // visible. innerAt is that radius as a fraction of the gradient circle.
    var bodyR = Math.round(520 * d.reach);
    var innerAt = R0 / bodyR;
    defs.appendChild(radial(uid + "-inner", [
      [0.00, "var(--amber-1)", 0],
      [innerAt * 0.94, "var(--amber-1)", 0],
      [innerAt * 1.02, "var(--amber-1)", 0.55],
      [innerAt * 1.22, "var(--amber-2)", 0.26],
      [innerAt * 1.60, "var(--amber-2)", 0.11],
      [innerAt * 2.20, "var(--amber-3)", 0.04],
      [0.86, "var(--amber-3)", 0.010],
      [1.00, "var(--amber-3)", 0]
    ]));
    // The outer tail is written in absolute fractions of the gradient circle,
    // not as multiples of the limb radius. Tied to the limb it ran off the end
    // of the gradient and dropped to nothing in the last one percent, which
    // drew a faint circular rim around the whole corona. There is no rim.
    defs.appendChild(radial(uid + "-outer", [
      [0.00, "var(--amber-2)", 0],
      [innerAt * 0.96, "var(--amber-2)", 0],
      [0.45, "var(--amber-2)", 0.048],
      [0.60, "var(--amber-3)", 0.024],
      [0.74, "var(--amber-3)", 0.009],
      [0.87, "var(--amber-4)", 0.003],
      [1.00, "var(--amber-4)", 0]
    ]));

    // One filament gradient, reused by every streamer: bright where it leaves
    // the limb, gone by its outer end.
    var fil = el("linearGradient", { id: uid + "-fil", x1: "0", y1: "0.5", x2: "1", y2: "0.5" });
    stop(fil, 0, "var(--amber-1)", 0.30);
    stop(fil, 0.14, "var(--amber-1)", 0.19);
    stop(fil, 0.48, "var(--amber-2)", 0.075);
    stop(fil, 1, "var(--amber-3)", 0);
    defs.appendChild(fil);

    // The warp. A fixed seed, so this is the same displacement every time.
    var warp = el("filter", {
      id: uid + "-warp", filterUnits: "userSpaceOnUse",
      x: 0, y: 0, width: VIEW.w, height: VIEW.h, "color-interpolation-filters": "sRGB"
    });
    warp.appendChild(el("feTurbulence", {
      type: "fractalNoise", baseFrequency: "0.009 0.019", numOctaves: 4,
      seed: 7, stitchTiles: "stitch", result: "noise"
    }));
    warp.appendChild(el("feDisplacementMap", {
      in: "SourceGraphic", in2: "noise", scale: 22,
      xChannelSelector: "R", yChannelSelector: "G"
    }));
    defs.appendChild(warp);

    var soft = el("filter", {
      id: uid + "-soft", filterUnits: "userSpaceOnUse",
      x: 0, y: 0, width: VIEW.w, height: VIEW.h, "color-interpolation-filters": "sRGB"
    });
    soft.appendChild(el("feGaussianBlur", { stdDeviation: 9 }));
    defs.appendChild(soft);

    // The occlusion. White shows the corona, the black disc takes it away, so
    // no light is painted where the moon is.
    var mask = el("mask", { id: uid + "-moon", maskUnits: "userSpaceOnUse", x: 0, y: 0, width: VIEW.w, height: VIEW.h });
    mask.appendChild(el("rect", { x: 0, y: 0, width: VIEW.w, height: VIEW.h, fill: "#fff" }));
    mask.appendChild(el("circle", { cx: c.x, cy: c.y, r: R0, fill: "#000" }));
    defs.appendChild(mask);

    var field = el("g", { "class": "corona-field", mask: "url(#" + uid + "-moon)" });
    svg.appendChild(field);

    var warped = el("g", { filter: "url(#" + uid + "-warp)" });
    field.appendChild(warped);

    warped.appendChild(el("circle", {
      cx: c.x, cy: c.y, r: bodyR, fill: "url(#" + uid + "-outer)",
      filter: "url(#" + uid + "-soft)"
    }));
    warped.appendChild(el("circle", { cx: c.x, cy: c.y, r: bodyR, fill: "url(#" + uid + "-inner)" }));

    // Streamers. Each one is an ellipse laid along the x axis and rotated to
    // its angle, so the gradient runs outward from the limb.
    var streamers = el("g", { "class": "corona-filaments" });
    warped.appendChild(streamers);
    for (var i = 0; i < d.filaments; i++) {
      var a = (i / d.filaments) * 360 + rand() * (320 / d.filaments);
      // A few streamers run a long way out and most do not, which is what
      // makes a corona read as a corona rather than as a sunburst.
      var roll = rand();
      var len = (40 + roll * roll * 420) * d.reach;
      var thick = 0.9 + rand() * 2.6;
      var inner = R0 + 1;
      var g = el("g", { transform: "rotate(" + a.toFixed(2) + " " + c.x + " " + c.y + ")" });
      g.appendChild(el("ellipse", {
        cx: (c.x + inner + len / 2).toFixed(1), cy: c.y,
        rx: (len / 2).toFixed(1), ry: thick.toFixed(1),
        fill: "url(#" + uid + "-fil)"
      }));
      streamers.appendChild(g);
    }

    // A thin bright ring hugging the limb. This is the light that survives
    // right at the edge of the disc, and without it the corona floats.
    var limbRing = el("circle", {
      cx: c.x, cy: c.y, r: R0 + 2, fill: "none",
      stroke: "var(--amber-1)", "stroke-width": 3, opacity: 0.42,
      filter: "url(#" + uid + "-soft)"
    });
    svg.appendChild(limbRing);

    // One bright point on the limb, with a flare that is longer than it is
    // wide. The angle comes from the commit, so it does not wander. The
    // streaks are drawn before the moon so they come out from behind it.
    var limbAngle = (seed % 360) * (Math.PI / 180);
    var lx = c.x + Math.cos(limbAngle) * R0;
    var ly = c.y + Math.sin(limbAngle) * R0;
    var lim = flare(uid, lx, ly, limbAngle, defs);
    svg.appendChild(lim.streaks);

    // The moon itself. It is the changed symbol, and it is the only place on
    // the page with no light in it at all.
    svg.appendChild(el("circle", { "class": "corona-moon", cx: c.x, cy: c.y, r: R0, fill: "var(--night)" }));
    svg.appendChild(lim.bead);

    svg.setAttribute("viewBox", "0 0 " + VIEW.w + " " + VIEW.h);
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");
    return d;
  }

  // flare returns the limb point: a bead, a long streak along the limb's
  // tangent, and a shorter one across it. The streaks are plain rectangles
  // filled with a rotated linear gradient and blurred once.
  function flare(uid, x, y, angle, defs) {
    var streaks = el("g", { "class": "corona-limb" });

    var lin = el("linearGradient", { id: uid + "-flare", x1: "0", y1: "0.5", x2: "1", y2: "0.5" });
    stop(lin, 0, "var(--amber-2)", 0);
    stop(lin, 0.34, "var(--amber-1)", 0.7);
    stop(lin, 0.5, "var(--amber-1)", 0.95);
    stop(lin, 0.66, "var(--amber-1)", 0.7);
    stop(lin, 1, "var(--amber-2)", 0);
    defs.appendChild(lin);

    var blur = el("filter", {
      id: uid + "-flareblur", filterUnits: "userSpaceOnUse",
      x: x - 320, y: y - 320, width: 640, height: 640, "color-interpolation-filters": "sRGB"
    });
    blur.appendChild(el("feGaussianBlur", { stdDeviation: 4 }));
    defs.appendChild(blur);

    var deg = angle * 180 / Math.PI;
    var inner = el("g", { filter: "url(#" + uid + "-flareblur)" });

    // Along the tangent, then across it. Two rectangles, one gradient.
    inner.appendChild(el("rect", {
      x: x - 300, y: y - 5, width: 600, height: 10,
      fill: "url(#" + uid + "-flare)",
      transform: "rotate(" + (deg + 90).toFixed(2) + " " + x.toFixed(1) + " " + y.toFixed(1) + ")"
    }));
    inner.appendChild(el("rect", {
      x: x - 78, y: y - 2.5, width: 156, height: 5,
      fill: "url(#" + uid + "-flare)",
      transform: "rotate(" + deg.toFixed(2) + " " + x.toFixed(1) + " " + y.toFixed(1) + ")"
    }));
    inner.appendChild(el("circle", { cx: x, cy: y, r: 11, fill: "var(--amber-1)", opacity: 0.85 }));
    streaks.appendChild(inner);

    var bead = el("g", { "class": "corona-bead" });
    bead.appendChild(el("circle", { cx: x, cy: y, r: 7, fill: "var(--amber-1)", opacity: 0.55 }));
    bead.appendChild(el("circle", { cx: x, cy: y, r: 3, fill: "var(--lit, #FFF3DC)" }));
    return { streaks: streaks, bead: bead };
  }

  // orb draws the legend's three orbs with the same primitive as the corona,
  // at three intensities. LIT is a full corona, PENUMBRA half of one, UMBRA
  // none at all, which is the whole vocabulary in one picture.
  function orb(svg, level) {
    var uid = "orb" + level;
    svg.textContent = "";
    svg.setAttribute("viewBox", "0 0 40 40");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");

    var defs = el("defs", {});
    svg.appendChild(defs);

    // Three intensities of one primitive, and the radius moves with the
    // intensity as well as the opacity. Two states that differ only in alpha
    // read as the same picture at this size.
    var peak = level === 2 ? 0.85 : (level === 1 ? 0.45 : 0);
    var reach = level === 2 ? 19 : (level === 1 ? 13 : 0);

    defs.appendChild(radial(uid + "-g", [
      [0.00, "var(--amber-1)", 0],
      [0.36, "var(--amber-1)", peak],
      [0.56, "var(--amber-2)", peak * 0.5],
      [0.80, "var(--amber-3)", peak * 0.16],
      [1.00, "var(--amber-4)", 0]
    ]));

    if (reach) {
      svg.appendChild(el("circle", { cx: 20, cy: 20, r: reach, fill: "url(#" + uid + "-g)" }));
    }
    svg.appendChild(el("circle", {
      cx: 20, cy: 20, r: 7.5, fill: "var(--night)",
      stroke: level === 0 ? "var(--umbra-edge)" : "none", "stroke-width": 1.25
    }));
    return svg;
  }

  function radial(id, stops) {
    var g = el("radialGradient", { id: id, cx: "0.5", cy: "0.5", r: "0.5" });
    stops.forEach(function (s) { stop(g, s[0], s[1], s[2]); });
    return g;
  }

  function stop(parent, offset, colour, opacity) {
    parent.appendChild(el("stop", {
      offset: (offset * 100) + "%", "stop-color": colour, "stop-opacity": opacity
    }));
  }

  globalThis.__corona = { build: build, orb: orb, density: density };
})();
