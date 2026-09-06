// Umbra report. Vanilla JavaScript, no libraries, no requests.
//
// Layout positions come from the JSON. This file never computes a position; it
// draws what layout.go decided and styles it by state.

(function () {
  "use strict";

  var SVG = "http://www.w3.org/2000/svg";

  function el(name, attrs) {
    var node = document.createElementNS(SVG, name);
    if (attrs) {
      for (var k in attrs) {
        if (Object.prototype.hasOwnProperty.call(attrs, k)) {
          node.setAttribute(k, String(attrs[k]));
        }
      }
    }
    return node;
  }

  function readData() {
    var tag = document.getElementById("umbra-data");
    if (!tag) { return null; }
    try {
      return JSON.parse(tag.textContent);
    } catch (e) {
      return null;
    }
  }

  // The light field: background, ring guides, the glow around each source, and
  // the shadow mask whose blurred blobs subtract light where the unexamined
  // nodes are. Drawn in the order the spec sets, before anything else.
  function drawField(svg, data) {
    var layout = data.layout;
    var vb = layout.view_box;
    svg.setAttribute("viewBox", vb.join(" "));
    svg.setAttribute("role", "group");

    var defs = el("defs");
    svg.appendChild(defs);

    // 1. Background.
    svg.appendChild(el("rect", {
      x: vb[0], y: vb[1], width: vb[2], height: vb[3], fill: "var(--night)"
    }));

    // 2. Ring guides.
    var rings = el("g", { "class": "rings" });
    (layout.rings || []).forEach(function (r) {
      var c = el("circle", {
        cx: layout.centre.x, cy: layout.centre.y, r: r.radius,
        fill: "none", stroke: "var(--rule)", "stroke-width": 1, opacity: 0.5
      });
      if (r.dashed) { c.setAttribute("stroke-dasharray", "4 5"); }
      rings.appendChild(c);

      if (r.label) {
        var t = el("text", {
          x: layout.centre.x, y: layout.centre.y - r.radius - 6,
          "text-anchor": "middle", fill: "var(--ink-2)", "font-size": 10.5,
          "font-family": "var(--ui)"
        });
        t.textContent = r.label;
        rings.appendChild(t);
      }
    });
    svg.appendChild(rings);

    // Unknown state draws no light field and no mask: the map goes flat.
    if (allUnknown(data)) { return; }

    // 3. The radial glow around each source, masked by the shadow blobs.
    var grad = el("radialGradient", { id: "glow" });
    grad.appendChild(el("stop", { offset: "0%", "stop-color": "var(--lit-glow)", "stop-opacity": 0.32 }));
    grad.appendChild(el("stop", { offset: "100%", "stop-color": "var(--lit-glow)", "stop-opacity": 0 }));
    defs.appendChild(grad);

    // 4. The shadow mask. White lets light through; the blurred black blobs
    // subtract it, so several umbra nodes close together merge into one pool.
    var mask = el("mask", { id: "shadow", maskUnits: "userSpaceOnUse",
      x: vb[0], y: vb[1], width: vb[2], height: vb[3] });
    mask.appendChild(el("rect", {
      x: vb[0], y: vb[1], width: vb[2], height: vb[3], fill: "#fff"
    }));

    var blobs = el("g", { filter: "url(#pool)", "class": "pools" });
    (data.nodes || []).forEach(function (n) {
      var place = layout.nodes[n.id];
      if (!place) { return; }
      var spec = poolFor(n.state);
      if (!spec) { return; }
      blobs.appendChild(el("circle", {
        cx: place.x, cy: place.y, r: spec.r, fill: spec.fill,
        "data-pool": n.id
      }));
    });
    mask.appendChild(blobs);
    defs.appendChild(mask);

    var blur = el("filter", { id: "pool", x: "-50%", y: "-50%", width: "200%", height: "200%" });
    blur.appendChild(el("feGaussianBlur", { stdDeviation: 14 }));
    defs.appendChild(blur);

    var light = el("g", { mask: "url(#shadow)", "class": "light" });

    // Several sources each draw the same gradient, and overlapping gradients
    // add up: four of them at full strength wash the middle of the map out
    // and take the labels with it. Scaling by the count keeps the brightest
    // point near the single-source value however many lights there are.
    var ids = Object.keys(layout.sources || {});
    var share = ids.length > 1 ? 1 / Math.sqrt(ids.length) : 1;
    ids.forEach(function (id) {
      var p = layout.sources[id];
      light.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: 500, fill: "url(#glow)", opacity: share
      }));
    });
    svg.appendChild(light);

    // The same blobs again, painted rather than subtracted. On the night
    // scheme --pool-ink is transparent and this layer does nothing; on a light
    // ground it is what makes a pool darker instead of brighter.
    var shade = el("g", { filter: "url(#pool)", "class": "shade" });
    (data.nodes || []).forEach(function (n) {
      var place = layout.nodes[n.id];
      if (!place) { return; }
      var spec = poolFor(n.state);
      if (!spec) { return; }
      shade.appendChild(el("circle", {
        cx: place.x, cy: place.y, r: spec.r, fill: "var(--pool-ink)",
        opacity: n.state === "penumbra" ? 0.55 : 1,
        "data-shade": n.id
      }));
    });
    svg.appendChild(shade);
  }

  // poolFor returns the mask blob for a state, or null when the state casts no
  // shadow. Umbra pools are the largest and fully black; penumbra pools are
  // smaller and partial, so a half-seen node dims the light without hiding it.
  function poolFor(state) {
    if (state === "umbra") { return { r: 42, fill: "#000" }; }
    if (state === "penumbra") { return { r: 28, fill: "rgba(0,0,0,0.55)" }; }
    return null;
  }

  function allUnknown(data) {
    var nodes = data.nodes || [];
    if (!nodes.length) { return false; }
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].state !== "unknown") { return false; }
    }
    return true;
  }

  // 5. Edges, under the nodes. Each node draws one edge to the previous step
  // on its shortest path to a source; a co-change node draws a dashed edge.
  function drawEdges(svg, data) {
    var layout = data.layout;
    var g = el("g", { "class": "edges" });

    (data.nodes || []).forEach(function (n) {
      var to = layout.nodes[n.id];
      if (!to) { return; }
      var prev = previousOnPath(n);
      var from = (prev && layout.nodes[prev]) || layout.sources[prev] ||
        (n.sources && layout.sources[n.sources[0]]) || layout.centre;
      if (!from) { return; }

      var line = el("line", {
        x1: from.x, y1: from.y, x2: to.x, y2: to.y,
        "class": "edge", "data-edge": n.id
      });
      styleEdge(line, n);
      g.appendChild(line);
    });
    svg.appendChild(g);
  }

  function previousOnPath(n) {
    var p = n.path || [];
    if (p.length < 2) { return (n.sources || [])[0]; }
    return p[p.length - 2];
  }

  function styleEdge(line, n) {
    var state = n.state;
    if (state === "lit") {
      line.setAttribute("stroke", "var(--lit-glow)");
      line.setAttribute("stroke-opacity", 0.55);
      line.setAttribute("stroke-width", 1.25);
    } else if (state === "penumbra") {
      line.setAttribute("stroke", "var(--lit-glow)");
      line.setAttribute("stroke-opacity", 0.3);
      line.setAttribute("stroke-width", 1.25);
    } else {
      line.setAttribute("stroke", "var(--umbra-edge)");
      line.setAttribute("stroke-opacity", 0.45);
      line.setAttribute("stroke-width", 1.25);
      line.setAttribute("stroke-dasharray", "2 3");
    }
    if (hasMod(n, "co-change only")) {
      line.setAttribute("stroke-dasharray", "6 5");
    }
  }

  function hasMod(n, name) {
    return (n.modifiers || []).indexOf(name) >= 0;
  }

  // 6. Nodes. The shape carries the state so the map survives greyscale: a
  // filled disc with a halo is lit, a half disc is penumbra, a hatched disc is
  // umbra, a dashed outline is unknown.
  function drawNodes(svg, data, defs) {
    var layout = data.layout;

    // The hatch that makes umbra readable without colour.
    var pat = el("pattern", {
      id: "hatch", width: 4, height: 4, patternUnits: "userSpaceOnUse",
      patternTransform: "rotate(45)"
    });
    pat.appendChild(el("rect", { width: 4, height: 4, fill: "var(--umbra)" }));
    pat.appendChild(el("line", { x1: 0, y1: 0, x2: 0, y2: 4, stroke: "var(--umbra-edge)", "stroke-width": 1 }));
    defs.appendChild(pat);

    var halo = el("filter", { id: "halo", x: "-80%", y: "-80%", width: "260%", height: "260%" });
    halo.appendChild(el("feGaussianBlur", { stdDeviation: 5 }));
    defs.appendChild(halo);

    var g = el("g", { "class": "nodes" });

    // Sources first, so a dependent never hides a light.
    (data.sources || []).forEach(function (s) {
      var p = layout.sources[s.id];
      if (!p) { return; }
      var sg = el("g", { "class": "source" });
      sg.appendChild(el("circle", { cx: p.x, cy: p.y, r: 16, fill: "var(--lit-glow)", opacity: 0.5, filter: "url(#halo)" }));
      sg.appendChild(el("circle", { cx: p.x, cy: p.y, r: 10, fill: "var(--lit)" }));
      g.appendChild(sg);
    });

    (data.nodes || []).forEach(function (n) {
      var p = layout.nodes[n.id];
      if (!p) { return; }
      g.appendChild(nodeGroup(n, p));
    });
    svg.appendChild(g);
  }

  function nodeGroup(n, p) {
    var g = el("g", {
      "class": "node", "data-node": n.id,
      role: "button", tabindex: -1, "aria-label": nodeLabel(n)
    });

    var r = radiusFor(n);

    // A generous transparent target so the node is easy to hit.
    g.appendChild(el("circle", { cx: p.x, cy: p.y, r: r + 8, "class": "hit" }));

    if (n.state === "lit") {
      g.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: r + 6, fill: "var(--lit-glow)", opacity: 0.6, filter: "url(#halo)"
      }));
      g.appendChild(el("circle", { cx: p.x, cy: p.y, r: r, fill: "var(--lit)" }));

    } else if (n.state === "penumbra") {
      appendPenumbra(g, n, p, r);

    } else if (n.state === "umbra") {
      g.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: r, fill: "url(#hatch)",
        stroke: "var(--umbra-edge)", "stroke-width": 1
      }));

    } else {
      g.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: r, fill: "none",
        stroke: "var(--unknown)", "stroke-width": 1, "stroke-dasharray": "3 3"
      }));
    }

    // A test carries a second ring, coloured by its outcome.
    if (n.is_test) {
      var ringColour = "var(--umbra-edge)";
      if (n.result === "pass") { ringColour = "var(--pass)"; }
      if (n.result === "fail") { ringColour = "var(--fail)"; }
      g.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: 10.5, fill: "none", stroke: ringColour, "stroke-width": 1
      }));
    }

    // A cracked probe gets two fixed strokes, so every crack looks the same.
    if (n.result === "fail") {
      g.appendChild(el("polyline", {
        points: crackPoints(p, r), fill: "none",
        stroke: "var(--fail)", "stroke-width": 1.5
      }));
    }

    // Modifiers read from the call site.
    if (hasMod(n, "far field")) {
      var t = tickAway(p, n);
      g.appendChild(el("line", {
        x1: t.x1, y1: t.y1, x2: t.x2, y2: t.y2,
        stroke: "var(--umbra-edge)", "stroke-width": 1
      }));
    }
    if (n.beacon) {
      g.appendChild(el("polygon", {
        points: (p.x - 4) + "," + (p.y - r - 6) + " " + (p.x + 4) + "," + (p.y - r - 6) +
          " " + p.x + "," + (p.y - r - 12),
        fill: "var(--lit)"
      }));
    }

    g.appendChild(el("circle", {
      cx: p.x, cy: p.y, r: r + 5, fill: "none", stroke: "var(--focus)",
      "stroke-width": 2, "stroke-dasharray": "3 3", "class": "ring-focus"
    }));
    return g;
  }

  // The penumbra half disc: the lit half faces the source, so the map shows
  // which way the light came from. The tier is drawn on the disc so it
  // survives greyscale.
  function appendPenumbra(g, n, p, r) {
    var angle = Math.atan2(p.toward.y - p.y, p.toward.x - p.x) * 180 / Math.PI;

    var whole = el("circle", { cx: p.x, cy: p.y, r: r, fill: "var(--penumbra)" });
    if (n.tier === "echo") {
      // Nothing was read at all, so the whole disc is dim with an outline.
      whole.setAttribute("opacity", 0.5);
      whole.setAttribute("stroke", "var(--penumbra)");
      whole.setAttribute("stroke-width", 1);
      whole.setAttribute("opacity", 0.5);
      g.appendChild(whole);
      return;
    }
    g.appendChild(whole);

    var sweep = n.tier === "glimpse" ? 90 : 180;
    var lit = el("path", {
      d: halfDisc(p, r, angle, sweep), fill: "var(--lit)",
      transform: ""
    });
    g.appendChild(lit);

    if (n.tier === "quoted") {
      g.appendChild(el("path", {
        d: halfDisc(p, r - 2, angle, sweep), fill: "none",
        stroke: "var(--penumbra)", "stroke-width": 1
      }));
    }
    if (n.tier === "afterimage") {
      g.appendChild(el("circle", {
        cx: p.x, cy: p.y, r: r + 3, fill: "none",
        stroke: "var(--penumbra)", "stroke-width": 1, "stroke-dasharray": "1 2"
      }));
    }
  }

  // halfDisc draws a wedge of the given sweep centred on the angle facing the
  // source.
  function halfDisc(p, r, angleDeg, sweepDeg) {
    var a0 = (angleDeg - sweepDeg / 2) * Math.PI / 180;
    var a1 = (angleDeg + sweepDeg / 2) * Math.PI / 180;
    var x0 = p.x + r * Math.cos(a0), y0 = p.y + r * Math.sin(a0);
    var x1 = p.x + r * Math.cos(a1), y1 = p.y + r * Math.sin(a1);
    var large = sweepDeg > 180 ? 1 : 0;
    return "M " + p.x + " " + p.y + " L " + x0 + " " + y0 +
      " A " + r + " " + r + " 0 " + large + " 1 " + x1 + " " + y1 + " Z";
  }

  function crackPoints(p, r) {
    return [
      (p.x - r * 0.7) + "," + (p.y - r * 0.2),
      (p.x - r * 0.1) + "," + (p.y + r * 0.3),
      (p.x + r * 0.3) + "," + (p.y - r * 0.4),
      (p.x + r * 0.8) + "," + (p.y + r * 0.1)
    ].join(" ");
  }

  function tickAway(p, n) {
    var dx = p.x - p.toward.x, dy = p.y - p.toward.y;
    var len = Math.hypot(dx, dy) || 1;
    var ux = dx / len, uy = dy / len;
    var r = radiusFor(n);
    return {
      x1: p.x + ux * (r + 2), y1: p.y + uy * (r + 2),
      x2: p.x + ux * (r + 8), y2: p.y + uy * (r + 8)
    };
  }

  function radiusFor(n) {
    if (n.kind === "class" || n.kind === "type") { return 9; }
    return 7;
  }

  function nodeLabel(n) {
    var parts = [n.name];
    if (n.is_test) { parts.push("test"); }
    parts.push(n.state);
    parts.push("depth " + n.depth);
    if (n.result === "fail") { parts.push("failed"); }
    return parts.join(", ");
  }

  // 7. Labels, placed outward from the ring by the anchor layout.go chose.
  function drawLabels(svg, data) {
    var layout = data.layout;
    var g = el("g", { "class": "labels" });

    var sources = data.sources || [];
    var multi = sources.length > 1;
    sources.forEach(function (s) {
      var p = layout.sources[s.id];
      if (!p) { return; }

      // One source sits alone at the centre and gets the full 22px label.
      // Several sit on the inner ring, so their labels are pushed outward
      // along their own angle and set smaller, or they pile on each other.
      var dx = p.x - layout.centre.x, dy = p.y - layout.centre.y;
      var len = Math.hypot(dx, dy);
      var lx = p.x, ly = p.y + 34, anchor = "middle";
      if (multi && len > 1) {
        var ux = dx / len, uy = dy / len;
        lx = p.x + ux * 46;
        ly = p.y + uy * 46 + 4;
        anchor = ux > 0.2 ? "start" : (ux < -0.2 ? "end" : "middle");
      }

      var name = el("text", {
        x: lx, y: ly, "text-anchor": anchor,
        "class": "src-label" + (multi ? " src-small" : "")
      });
      name.textContent = s.name;
      g.appendChild(name);

      var kind = el("text", {
        x: lx, y: ly + (multi ? 13 : 16), "text-anchor": anchor, "class": "src-kind"
      });
      kind.textContent = s.change + " changed";
      g.appendChild(kind);
    });

    (data.nodes || []).forEach(function (n) {
      var p = layout.nodes[n.id];
      if (!p) { return; }
      var t = el("text", {
        "class": "label " + (n.state === "lit" ? "lit" : "dim"),
        "data-label": n.id
      });
      var r = radiusFor(n) + 6;
      if (p.anchor === "right") {
        t.setAttribute("x", p.x + r); t.setAttribute("y", p.y + 4); t.setAttribute("text-anchor", "start");
      } else if (p.anchor === "left") {
        t.setAttribute("x", p.x - r); t.setAttribute("y", p.y + 4); t.setAttribute("text-anchor", "end");
      } else if (p.anchor === "above") {
        t.setAttribute("x", p.x); t.setAttribute("y", p.y - r - 6); t.setAttribute("text-anchor", "middle");
      } else {
        t.setAttribute("x", p.x); t.setAttribute("y", p.y + r + 12); t.setAttribute("text-anchor", "middle");
      }
      t.textContent = n.name;
      t.setAttribute("data-score", n.score || 0);
      g.appendChild(t);
    });
    svg.appendChild(g);
    hideCollidingLabels(g);
  }

  // Labels never overlap. When two would collide the lower-scored one is
  // hidden until the reader interacts, so the map stays readable without
  // pretending the node is not there: its disc is still drawn.
  function hideCollidingLabels(g) {
    var labels = Array.prototype.slice.call(g.querySelectorAll(".label"));
    labels.sort(function (a, b) {
      return Number(b.getAttribute("data-score")) - Number(a.getAttribute("data-score"));
    });

    var kept = [];
    labels.forEach(function (t) {
      var box;
      try {
        box = t.getBBox();
      } catch (e) {
        return;
      }
      var clash = kept.some(function (b) { return overlaps(box, b); });
      if (clash) {
        t.hidden = true;
        t.setAttribute("data-collided", "1");
      } else {
        kept.push(box);
      }
    });
  }

  function overlaps(a, b) {
    var pad = 2;
    return !(a.x + a.width + pad < b.x || b.x + b.width + pad < a.x ||
             a.y + a.height + pad < b.y || b.y + b.height + pad < a.y);
  }

  function boot() {
    var data = readData();
    var svg = document.getElementById("map");
    if (!data || !svg) { return; }
    svg.textContent = "";

    drawField(svg, data);
    var defs = svg.querySelector("defs");
    drawEdges(svg, data);
    drawNodes(svg, data, defs);
    drawLabels(svg, data);

    svg.setAttribute("aria-label", mapLabel(data));
    wireSelection(svg, data);
    wireFilters();
    wireDetail(data);
    wireReplay(svg, data);
    wireSpotlight(svg);
    wireKeyboard(svg, data);
  }

  // The detail panel. One is open at a time; Escape closes it.
  var detailState = { data: null, open: null };

  function wireDetail(data) {
    detailState.data = data;
    var panel = document.getElementById("detail");
    if (!panel) { return; }
    panel.querySelector(".detail-close").addEventListener("click", closeDetail);
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") { closeDetail(); }
    });
  }

  function closeDetail() {
    var panel = document.getElementById("detail");
    if (panel) { panel.hidden = true; }
    detailState.open = null;
  }

  function openDetail(id) {
    var data = detailState.data;
    var panel = document.getElementById("detail");
    if (!data || !panel) { return; }

    var node = null;
    (data.nodes || []).forEach(function (n) { if (n.id === id) { node = n; } });
    if (!node) { return; }

    var body = panel.querySelector(".detail-body");
    body.textContent = "";

    // 1. Name, kind and span.
    var h = document.createElement("h3");
    h.textContent = node.name;
    body.appendChild(h);

    var meta = document.createElement("p");
    meta.className = "sub";
    meta.textContent = (node.kind || "symbol") + "  " + node.file +
      "  lines " + node.span[0] + " to " + node.span[1];
    body.appendChild(meta);

    // 2. The state sentence, in plain words, then the call-site facts.
    var state = document.createElement("p");
    state.className = "detail-state";
    var word = document.createElement("b");
    word.textContent = capitalise(node.state) + (node.tier ? ", " + node.tier : "") + ". ";
    state.appendChild(word);
    state.appendChild(document.createTextNode(node.sentence || ""));
    body.appendChild(state);

    (node.modifiers || []).forEach(function (m) {
      if (m === "far field" || m === "fault line" || m === "scar") {
        body.appendChild(para(capitalise(m) + ": " + modifierFact(m, node), "sub"));
      }
    });
    if (node.beacon) {
      body.appendChild(para("Beacon: the call site at line " + node.call_site + " reads " + node.beacon, "sub"));
    }

    // 3. The path back to the source, as a chain.
    if ((node.path || []).length > 1) {
      body.appendChild(para("Path to the source", "detail-h"));
      var chain = document.createElement("p");
      chain.className = "chain";
      node.path.forEach(function (step, i) {
        if (i > 0) { chain.appendChild(document.createTextNode("  <-  ")); }
        var b = document.createElement("button");
        b.type = "button";
        b.textContent = nameOf(data, step);
        b.addEventListener("click", function () { select(step); });
        chain.appendChild(b);
      });
      body.appendChild(chain);
    }

    // 4. What the session did to this file, if anything.
    if ((node.exposures || []).length) {
      body.appendChild(para("Exposures", "detail-h"));
      var ul = document.createElement("ul");
      node.exposures.forEach(function (x) {
        var li = document.createElement("li");
        li.textContent = "seq " + x.seq + "  " + x.kind +
          (x.range ? "  lines " + x.range[0] + " to " + x.range[1] : "");
        ul.appendChild(li);
      });
      body.appendChild(ul);
    } else {
      body.appendChild(para("No exposure: the session never touched this file.", "sub"));
    }

    // 5. Probes.
    if ((node.tests || []).length) {
      body.appendChild(para("Probes", "detail-h"));
      var tl = document.createElement("ul");
      node.tests.forEach(function (t) {
        var li = document.createElement("li");
        li.textContent = t.id + "  " + t.outcome;
        tl.appendChild(li);
      });
      body.appendChild(tl);
    }
    if (node.failure_excerpt) {
      var pre = document.createElement("pre");
      pre.textContent = node.failure_excerpt;
      body.appendChild(pre);
    }

    // 6. The factors as a stacked bar, drawn in CSS with no chart library.
    body.appendChild(para("Why it ranks " + Number(node.score).toFixed(1), "detail-h"));
    if (node.beacon) {
      body.appendChild(para("pinned to the top by a beacon", "sub"));
    }
    body.appendChild(factorBar(node));

    // 7. Commands a reader can run to check any of this.
    body.appendChild(para("Reproduce", "detail-h"));
    var cmds = [
      "entire graph impact --symbol " + node.name + " --repo .",
      "entire graph def " + node.name + " --repo ."
    ];
    (node.tests || []).forEach(function (t) { cmds.push(t.id); });
    cmds.forEach(function (c) {
      var pre = document.createElement("pre");
      pre.className = "cmd";
      pre.textContent = c;
      body.appendChild(pre);
    });

    if (node.signature) {
      body.appendChild(para("Declaration", "detail-h"));
      var sig = document.createElement("pre");
      sig.textContent = node.signature;
      body.appendChild(sig);
    }

    panel.hidden = false;
    detailState.open = id;
    panel.querySelector(".detail-close").focus();
  }

  function modifierFact(m, node) {
    if (m === "far field") { return node.file + " is a different package from its source."; }
    if (m === "fault line") { return "the call sits inside error handling at line " + node.call_site + "."; }
    if (m === "scar") { return "this file has several recent fix commits."; }
    return "";
  }

  function factorBar(node) {
    var wrap = document.createElement("div");
    wrap.className = "bar";
    var order = ["relation", "dependents", "state", "farfield", "faultline", "scar", "test"];
    var total = 0;
    order.forEach(function (k) {
      if (node.factors && typeof node.factors[k] === "number") { total += node.factors[k]; }
    });
    if (total <= 0) { total = 1; }
    order.forEach(function (k) {
      var v = node.factors && node.factors[k];
      if (typeof v !== "number" || v === 0) { return; }
      var seg = document.createElement("span");
      seg.className = "seg seg-" + k;
      seg.style.width = (100 * v / total) + "%";
      seg.textContent = k + " " + v;
      seg.title = k + " " + v;
      wrap.appendChild(seg);
    });
    return wrap;
  }

  function nameOf(data, id) {
    var out = id;
    (data.nodes || []).forEach(function (n) { if (n.id === id) { out = n.name; } });
    (data.sources || []).forEach(function (s) { if (s.id === id) { out = s.name; } });
    return out;
  }

  function para(text, cls) {
    var p = document.createElement("p");
    if (cls) { p.className = cls; }
    p.textContent = text;
    return p;
  }

  function capitalise(s) {
    return String(s || "").charAt(0).toUpperCase() + String(s || "").slice(1);
  }

  var select = function () {};

  // ---------------------------------------------------------------------
  // State reconstruction.
  //
  // These rules mirror internal/shadow/classify.go exactly, including the
  // order of the penumbra tiers. A test feeds the recorded scenarios through
  // both and compares, so the two cannot drift apart silently.
  // ---------------------------------------------------------------------

  var TIER_ORDER = ["glance", "glimpse", "quoted", "afterimage", "echo"];

  function tierRank(t) {
    var i = TIER_ORDER.indexOf(t);
    return i < 0 ? TIER_ORDER.length : i;
  }

  // stateAt returns the state and tier of a node considering only the evidence
  // at or before seq. Pass Infinity for the final state at commit.
  function stateAt(node, exposures, t0, hasAnyEvidence, seq) {
    if (!hasAnyEvidence) { return { state: "unknown", tier: "" }; }

    exposures = exposures || [];
    var span = node.span || [0, 0];
    var best = "";
    var consider = function (t) {
      if (best === "" || tierRank(t) < tierRank(best)) { best = t; }
    };
    var saw = false;

    for (var i = 0; i < exposures.length; i++) {
      var x = exposures[i];
      if (x.seq > seq) { continue; }
      saw = true;

      if (x.kind === "edit") { return { state: "lit", tier: "" }; }

      if (x.kind === "read") {
        var covers = !x.range || (x.range[0] <= span[0] && x.range[1] >= span[1]);
        var afterCut = !t0 || x.seq >= t0;
        if (covers && afterCut) { return { state: "lit", tier: "" }; }
        if (covers) { consider("afterimage"); } else { consider("glance"); }
      } else if (x.kind === "grep" || x.kind === "glob") {
        consider("glimpse");
      } else if (x.kind === "quoted") {
        consider("quoted");
      } else if (x.kind === "mention") {
        consider("echo");
      }
    }

    if (!saw || best === "") { return { state: "umbra", tier: "" }; }
    return { state: "penumbra", tier: best };
  }

  // Expose the reconstruction for the cross-check test that compares it with
  // the Go classifier. Nothing on the page uses this.
  if (typeof globalThis !== "undefined") {
    globalThis.__umbra = { stateAt: stateAt, tierRank: tierRank };
  }

  // ---------------------------------------------------------------------
  // Attention replay.
  // ---------------------------------------------------------------------

  function wireReplay(svg, data) {
    var strip = document.getElementById("replay");
    if (!strip) { return; }

    var timeline = data.timeline || [];
    if (!timeline.length) {
      strip.hidden = true;
      return;
    }

    var maxSeq = timeline[timeline.length - 1].seq;
    var hasAny = anyEvidence(data);
    var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    var current = maxSeq;      // the page loads showing the eclipse
    var playing = false;
    var speed = 1;
    var timer = null;

    var track = strip.querySelector(".track");
    var headline = strip.querySelector(".now");
    var counter = strip.querySelector(".counter");
    var playBtn = strip.querySelector('[data-act="play"]');

    // Ticks: one per event, shaped by kind so the sequence reads as a sequence.
    timeline.forEach(function (ev, i) {
      var t = document.createElement("button");
      t.type = "button";
      t.className = "tick tick-" + ev.kind + (ev.seq === data.t0 ? " cut" : "");
      t.style.left = (100 * i / Math.max(1, timeline.length - 1)) + "%";
      t.title = describe(ev, data);
      t.setAttribute("aria-label", describe(ev, data));
      t.addEventListener("click", function () { goTo(ev.seq); });
      track.appendChild(t);
    });

    var head = document.createElement("div");
    head.className = "playhead";
    track.appendChild(head);

    function render() {
      applyState(svg, data, current, hasAny, reduced);

      var ev = eventAt(timeline, current);
      headline.textContent = ev ? describe(ev, data) : "";
      counter.textContent = "seq " + current + " of " + maxSeq;

      var idx = indexOf(timeline, current);
      head.style.left = (100 * idx / Math.max(1, timeline.length - 1)) + "%";

      strip.classList.toggle("at-end", current >= maxSeq);
    }

    function goTo(seq) {
      current = Math.max(0, Math.min(maxSeq, seq));
      render();
    }

    function step(dir) {
      pause();
      var idx = indexOf(timeline, current) + dir;
      idx = Math.max(0, Math.min(timeline.length - 1, idx));
      goTo(timeline[idx].seq);
    }

    function play() {
      if (playing) { return; }
      // Pressing play at the end starts the sweep again from the beginning.
      if (current >= maxSeq) { current = 0; }
      playing = true;
      playBtn.textContent = "Pause";
      playBtn.setAttribute("aria-label", "Pause the sweep");
      tick();
    }

    function pause() {
      playing = false;
      if (timer) { clearTimeout(timer); timer = null; }
      playBtn.textContent = "Play";
      playBtn.setAttribute("aria-label", "Play the sweep");
    }

    function tick() {
      if (!playing) { return; }
      var idx = indexOf(timeline, current) + 1;
      if (idx >= timeline.length) { pause(); goTo(maxSeq); return; }
      goTo(timeline[idx].seq);
      // At 1x each event holds for 600 milliseconds; at 4x, 150.
      timer = setTimeout(tick, 600 / speed);
    }

    strip.addEventListener("click", function (e) {
      var act = e.target.getAttribute && e.target.getAttribute("data-act");
      if (!act) { return; }
      if (act === "play") { playing ? pause() : play(); }
      if (act === "back") { step(-1); }
      if (act === "fwd") { step(1); }
      if (act === "speed") {
        speed = speed === 1 ? 4 : 1;
        e.target.textContent = speed + "x";
      }
    });

    document.addEventListener("keydown", function (e) {
      var tag = (document.activeElement && document.activeElement.tagName) || "";
      if (tag === "INPUT" || tag === "TEXTAREA") { return; }
      if (e.key === " ") { e.preventDefault(); playing ? pause() : play(); }
      if (e.key === "[") { step(-1); }
      if (e.key === "]") { step(1); }
      if (e.key === "Home") { pause(); goTo(0); }
      if (e.key === "End") { pause(); goTo(maxSeq); }
    });

    render();
  }

  function anyEvidence(data) {
    var nodes = data.nodes || [];
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].state !== "unknown") { return true; }
    }
    return nodes.length === 0;
  }

  function indexOf(timeline, seq) {
    var idx = 0;
    for (var i = 0; i < timeline.length; i++) {
      if (timeline[i].seq <= seq) { idx = i; }
    }
    return idx;
  }

  function eventAt(timeline, seq) {
    return timeline[indexOf(timeline, seq)];
  }

  // One line of plain text naming the current event.
  function describe(ev, data) {
    if (ev.seq === data.t0 && ev.kind === "edit") {
      return "the cut  edit " + ev.path + "  the change begins here";
    }
    switch (ev.kind) {
      case "read":
        return "read " + ev.path + (ev.range ? "  lines " + ev.range[0] + " to " + ev.range[1] : "  whole file");
      case "edit":
        return "edit " + ev.path;
      case "grep":
      case "glob":
        return ev.kind + "  hit " + (ev.paths || []).join(", ");
      case "quoted":
        return "content of " + (ev.paths || []).join(", ") + " appeared in a tool result";
      case "mention":
        return "mentioned " + ((ev.symbols || []).concat(ev.paths || [])).join(", ");
      case "command":
        return ev.cmd || "ran a command";
    }
    return ev.kind;
  }

  // applyState restyles the map for the evidence up to seq. Rendering at a seq
  // is a pure function of the report and that integer.
  function applyState(svg, data, seq, hasAny, reduced) {
    (data.nodes || []).forEach(function (n) {
      var st = stateAt(n, n.exposures || [], data.t0, hasAny, seq);
      var g = svg.querySelector('[data-node="' + cssEscape(n.id) + '"]');
      if (!g) { return; }

      var place = data.layout.nodes[n.id];
      var shown = Object.assign({}, n, { state: st.state, tier: st.tier });
      var fresh = nodeGroup(shown, place);
      fresh.setAttribute("class", g.getAttribute("class"));
      if (reduced) { fresh.style.transition = "none"; }
      g.parentNode.replaceChild(fresh, g);

      var edge = svg.querySelector('[data-edge="' + cssEscape(n.id) + '"]');
      if (edge) { styleEdge(edge, shown); }

      var spec = poolFor(st.state);
      var pool = svg.querySelector('[data-pool="' + cssEscape(n.id) + '"]');
      if (pool) {
        pool.setAttribute("r", spec ? spec.r : 0);
        pool.setAttribute("fill", spec ? spec.fill : "#000");
        pool.setAttribute("opacity", spec ? 1 : 0);
      }
      var shade = svg.querySelector('[data-shade="' + cssEscape(n.id) + '"]');
      if (shade) {
        shade.setAttribute("r", spec ? spec.r : 0);
        shade.setAttribute("opacity", spec ? (st.state === "penumbra" ? 0.55 : 1) : 0);
      }
    });
    rewireNodes(svg);
  }

  function rewireNodes(svg) {
    Array.prototype.forEach.call(svg.querySelectorAll(".node"), function (g) {
      if (g.getAttribute("data-wired")) { return; }
      g.setAttribute("data-wired", "1");
      g.addEventListener("click", function () {
        select(g.getAttribute("data-node"));
        openDetail(g.getAttribute("data-node"));
      });
    });
  }

  // Selecting a node highlights its docket row, and hovering a row highlights
  // the node and its path, so the two halves of the page stay in step.
  function wireSelection(svg, data) {
    var rows = document.querySelectorAll("table.docket tbody tr");

    select = function (id) {
      Array.prototype.forEach.call(rows, function (row) {
        var on = row.getAttribute("data-id") === id;
        row.classList.toggle("on", on);
        if (on) { row.scrollIntoView({ block: "nearest" }); }
      });
      Array.prototype.forEach.call(svg.querySelectorAll(".edge"), function (e) {
        e.setAttribute("stroke-opacity", e.getAttribute("data-edge") === id ? 1 : "");
      });
    };

    Array.prototype.forEach.call(svg.querySelectorAll(".node"), function (g) {
      g.addEventListener("click", function () {
        select(g.getAttribute("data-node"));
        openDetail(g.getAttribute("data-node"));
      });
      g.addEventListener("focus", function () { select(g.getAttribute("data-node")); });
    });

    Array.prototype.forEach.call(rows, function (row) {
      row.addEventListener("mouseenter", function () { select(row.getAttribute("data-id")); });
      row.addEventListener("click", function () {
        select(row.getAttribute("data-id"));
        openDetail(row.getAttribute("data-id"));
      });
    });
  }

  // The filters hide docket rows and drop the matching map labels, so the map
  // never claims a node does not exist.
  function wireFilters() {
    var bar = document.getElementById("filters");
    if (!bar) { return; }
    var rows = document.querySelectorAll("table.docket tbody tr");

    function apply(mode) {
      Array.prototype.forEach.call(rows, function (row) {
        var state = row.getAttribute("data-state");
        var show = true;
        if (mode === "shadowed") { show = state !== "lit"; }
        if (mode === "probes") { show = row.getAttribute("data-probe") === "1" || state === "umbra"; }
        if (mode === "cracks") { show = row.getAttribute("data-crack") === "1"; }
        if (mode === "leaks") { show = row.getAttribute("data-leak") === "1"; }
        row.hidden = !show;

        var label = document.querySelector('[data-label="' + cssEscape(row.getAttribute("data-id")) + '"]');
        if (label) { label.hidden = !show; }
      });
    }

    Array.prototype.forEach.call(bar.querySelectorAll("button"), function (b) {
      b.addEventListener("click", function () {
        Array.prototype.forEach.call(bar.querySelectorAll("button"), function (o) {
          o.classList.toggle("on", o === b);
        });
        apply(b.getAttribute("data-filter"));
      });
    });
    apply("shadowed");
  }

  // The torch: a soft disc following the pointer that reveals labels hidden by
  // the collision rule. It is decoration in service of exploration; it changes
  // no data. Off for touch and for readers who asked for less motion.
  function wireSpotlight(svg) {
    var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    var fine = window.matchMedia && window.matchMedia("(pointer: fine)").matches;
    if (reduced || !fine) { return; }

    var defs = svg.querySelector("defs");
    var grad = el("radialGradient", { id: "torch" });
    grad.appendChild(el("stop", { offset: "0%", "stop-color": "var(--lit-glow)", "stop-opacity": 0.18 }));
    grad.appendChild(el("stop", { offset: "100%", "stop-color": "var(--lit-glow)", "stop-opacity": 0 }));
    defs.appendChild(grad);

    var torch = el("circle", { r: 110, fill: "url(#torch)", "class": "torch", cx: -999, cy: -999 });
    svg.appendChild(torch);

    svg.addEventListener("pointermove", function (e) {
      var pt = toViewBox(svg, e.clientX, e.clientY);
      torch.setAttribute("cx", pt.x);
      torch.setAttribute("cy", pt.y);
      revealNear(svg, pt, 110);
    });
    svg.addEventListener("pointerleave", function () {
      torch.setAttribute("cx", -999);
      torch.setAttribute("cy", -999);
      revealNear(svg, { x: -9999, y: -9999 }, 0);
    });
  }

  function toViewBox(svg, clientX, clientY) {
    var r = svg.getBoundingClientRect();
    var vb = svg.viewBox.baseVal;
    return {
      x: vb.x + (clientX - r.left) * vb.width / r.width,
      y: vb.y + (clientY - r.top) * vb.height / r.height
    };
  }

  // A label hidden by the collision rule shows while the torch is over it, so
  // the reader can read what is in shadow without the map being cluttered.
  function revealNear(svg, pt, radius) {
    Array.prototype.forEach.call(svg.querySelectorAll('[data-collided="1"]'), function (t) {
      var x = Number(t.getAttribute("x")), y = Number(t.getAttribute("y"));
      t.hidden = Math.hypot(x - pt.x, y - pt.y) > radius;
    });
  }

  // Keyboard: the map is one tab stop; inside it the arrows move through the
  // nodes and Enter opens the detail panel. Nothing is reachable by pointer
  // only.
  function wireKeyboard(svg, data) {
    var order = (data.nodes || []).map(function (n) { return n.id; });
    var at = -1;

    svg.setAttribute("tabindex", "0");

    function focusAt(i) {
      if (!order.length) { return; }
      at = (i + order.length) % order.length;
      var g = svg.querySelector('[data-node="' + cssEscape(order[at]) + '"]');
      if (!g) { return; }
      Array.prototype.forEach.call(svg.querySelectorAll(".node"), function (o) {
        o.setAttribute("tabindex", "-1");
      });
      g.setAttribute("tabindex", "0");
      g.focus();
      select(order[at]);
    }

    svg.addEventListener("focus", function () {
      if (at < 0) { focusAt(0); }
    });

    svg.addEventListener("keydown", function (e) {
      if (e.key === "ArrowDown" || e.key === "ArrowRight") { e.preventDefault(); focusAt(at + 1); }
      if (e.key === "ArrowUp" || e.key === "ArrowLeft") { e.preventDefault(); focusAt(at - 1); }
      if (e.key === "Enter" && at >= 0) { e.preventDefault(); openDetail(order[at]); }
      if (e.key === "l" || e.key === "L") { toggleLabels(svg); }
    });
  }

  function toggleLabels(svg) {
    var g = svg.querySelector(".labels");
    if (!g) { return; }
    var on = g.getAttribute("data-all") === "1";
    g.setAttribute("data-all", on ? "0" : "1");
    Array.prototype.forEach.call(g.querySelectorAll('[data-collided="1"]'), function (t) {
      t.hidden = on;
    });
  }

  function cssEscape(s) {
    return String(s).replace(/["\\]/g, "\\$&");
  }

  function mapLabel(data) {
    var s = data.summary || {};
    return "Eclipse map: " + (s.lit || 0) + " lit, " + (s.penumbra || 0) +
      " penumbra, " + (s.umbra || 0) + " umbra, " + (s.unknown || 0) + " unknown.";
  }

  // In a browser the page boots itself. Loaded without a document, for the
  // cross-check test that compares this reconstruction with the Go
  // classifier, the file only exposes globalThis.__umbra and draws nothing.
  if (typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", boot);
    } else {
      boot();
    }
  }
})();
