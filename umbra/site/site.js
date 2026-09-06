// Landing page behaviour.
//
// The map, the replay and the state rules all come from umbra.js, which is the
// report's own script and is not modified here. This file adds only what the
// landing page has and the report does not: the corona behind the map, the
// legend that filters it, the four beats, the day and night divider, and the
// drop-in viewer.
//
// Nothing is uploaded and no request is made.

(function () {
  "use strict";

  var MAX_BYTES = 20 * 1024 * 1024;
  var MOON_R = 132;

  var sample = null;
  var hasAny = true;
  var maxSeq = 0;
  var reduced = false;

  function $(sel) { return document.querySelector(sel); }
  function all(sel) { return Array.prototype.slice.call(document.querySelectorAll(sel)); }

  function msg(text) {
    var el = document.getElementById("dropmsg");
    if (el) { el.textContent = text || ""; }
  }

  // ------------------------------------------------------------------ copy

  function showSaid(data) {
    var el = document.getElementById("said");
    if (!el) { return; }
    el.textContent = "";
    if (!data || !data.session_said) { return; }
    el.appendChild(document.createTextNode("\u201C" + data.session_said + "\u201D"));
    if (data.session_said_from) {
      var note = document.createElement("span");
      note.className = "mono";
      note.style.display = "block";
      note.style.marginTop = "8px";
      note.textContent = data.session_said_from;
      el.appendChild(note);
    }
  }

  // The contradiction is counted from the report, never written by hand. If
  // the numbers change, the sentence changes with them.
  function showContradiction(data) {
    var el = document.getElementById("contradiction");
    if (!el || !data) { return; }
    var s = data.summary || {};
    var total = (s.lit || 0) + (s.penumbra || 0) + (s.umbra || 0) + (s.unknown || 0);
    var fails = ((data.execution || {}).new_failures || []).length;

    el.textContent = "";
    add(el, "The change reaches ");
    strong(el, String(total));
    add(el, total === 1 ? " symbol. The session opened " : " symbols. The session opened ");
    strong(el, String(s.lit || 0));
    add(el, ". ");
    if (s.umbra) {
      strong(el, String(s.umbra));
      add(el, s.umbra === 1
        ? " was never touched or mentioned. "
        : " were never touched or mentioned. ");
    }
    if (fails) {
      strong(el, String(fails));
      add(el, fails === 1 ? " test broke." : " tests broke.");
    }
  }

  // The four numbers in the hero and the three counts on the state cards are
  // the report's own. Nothing on this page is a figure somebody chose.
  function showStats(data) {
    if (!data) { return; }
    var sum = data.summary || {};
    var v = {
      lit: sum.lit || 0,
      penumbra: sum.penumbra || 0,
      umbra: sum.umbra || 0,
      unknown: sum.unknown || 0,
      fails: ((data.execution || {}).new_failures || []).length
    };
    v.total = v.lit + v.penumbra + v.umbra + v.unknown;
    all("[data-stat]").forEach(function (el) {
      var key = el.getAttribute("data-stat");
      if (key in v) { el.textContent = String(v[key]); }
    });
  }

  function add(el, text) { el.appendChild(document.createTextNode(text)); }
  function strong(el, text) {
    var b = document.createElement("b");
    b.textContent = text;
    el.appendChild(b);
  }

  function wireCopy() {
    var btn = document.getElementById("copy");
    var cmd = document.getElementById("install-cmd");
    var out = document.getElementById("copied");
    if (!btn || !cmd) { return; }
    btn.addEventListener("click", function () {
      var text = cmd.textContent;
      var done = function () { if (out) { out.textContent = "Copied."; } };
      var failed = function () {
        if (out) { out.textContent = "Copying is blocked here. Select the line and copy it."; }
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, failed);
      } else {
        failed();
      }
    });
  }

  // ---------------------------------------------------------------- corona

  function drawCorona(data) {
    var svg = document.getElementById("corona");
    if (!svg || !globalThis.__corona) { return; }
    globalThis.__corona.build(svg, data, MOON_R);
  }

  function drawOrbs() {
    if (!globalThis.__corona) { return; }
    all(".orb").forEach(function (svg) {
      globalThis.__corona.orb(svg, parseInt(svg.getAttribute("data-orb"), 10));
    });
  }

  // ---------------------------------------------------------------- replay

  // The landing page drives the map through the replay strip's own controls,
  // which is the same path a reader takes. It does not reach inside umbra.js.

  function currentSeq() {
    var counter = $("#replay .counter");
    var m = counter && /seq\s+(\d+)/.exec(counter.textContent || "");
    return m ? parseInt(m[1], 10) : maxSeq;
  }

  function pressKey(key) {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: key, bubbles: true }));
  }

  function goToSeq(seq) {
    var timeline = (sample && sample.timeline) || [];
    var ticks = all("#replay .track .tick");
    var idx = -1;
    for (var i = 0; i < timeline.length; i++) {
      if (timeline[i].seq <= seq) { idx = i; }
    }
    if (idx >= 0 && ticks[idx]) { ticks[idx].click(); }
  }

  var BEATS = {
    claim: function () { pressKey("Home"); },
    field: function () {
      var t0 = sample ? sample.t0 : 0;
      var last = 0;
      ((sample && sample.timeline) || []).forEach(function (e) {
        if (e.seq < t0) { last = e.seq; }
      });
      goToSeq(last);
    },
    cut: function () { goToSeq(sample ? sample.t0 : 0); },
    sweep: function () { pressKey("End"); }
  };

  function wireBeats() {
    var bar = document.getElementById("beats");
    if (!bar) { return; }
    var buttons = Array.prototype.slice.call(bar.querySelectorAll("button"));
    buttons.forEach(function (b) {
      b.addEventListener("click", function () {
        buttons.forEach(function (o) { o.setAttribute("aria-pressed", String(o === b)); });
        var fn = BEATS[b.getAttribute("data-beat")];
        if (fn) { fn(); }
      });
    });
  }

  // ---------------------------------------------------------------- legend

  var filters = {};

  function wireLegend() {
    var bar = document.getElementById("legend");
    if (!bar) { return; }
    Array.prototype.slice.call(bar.querySelectorAll("button")).forEach(function (b) {
      b.addEventListener("click", function () {
        var state = b.getAttribute("data-state");
        filters[state] = !filters[state];
        b.setAttribute("aria-pressed", String(!!filters[state]));
        applyFilter();
      });
    });
  }

  // A filtered node keeps its disc and loses its emphasis. Removing it would
  // make the map claim the node is not there, which is the one thing the map
  // must never do.
  var filtering = false;

  function applyFilter() {
    var svg = document.getElementById("map");
    if (filtering || !svg || !sample || !globalThis.__umbra) { return; }
    filtering = true;
    var on = Object.keys(filters).filter(function (k) { return filters[k]; });
    var seq = currentSeq();

    (sample.nodes || []).forEach(function (n) {
      var st = globalThis.__umbra.stateAt(n, n.exposures || [], sample.t0, hasAny, seq);
      var muted = on.length > 0 && on.indexOf(st.state) < 0;
      var g = svg.querySelector('[data-node="' + cssEscape(n.id) + '"]');
      if (g) { g.classList.toggle("muted", muted); }
      var label = svg.querySelector('[data-label="' + cssEscape(n.id) + '"]');
      if (label) { label.classList.toggle("muted", muted); }
    });
    filtering = false;
  }

  function cssEscape(s) { return String(s).replace(/["\\]/g, "\\$&"); }

  // ------------------------------------------------------------- day layer

  // The day scheme is a second copy of the same two SVGs under day tokens,
  // clipped by the divider. It is a real inversion rather than a filter: the
  // gradients and pools are drawn again from the day values, so shadow pools
  // stay darker than the paper around them.
  //
  // It is built only when it is visible, so an ordinary visit rasterises the
  // corona's turbulence filter once, not twice.

  var dayDirty = true;
  var daySyncTimer = null;

  function reid(root, suffix) {
    var nodes = root.querySelectorAll("*");
    Array.prototype.forEach.call(nodes, function (n) {
      if (n.id) { n.id = n.id + suffix; }
      Array.prototype.forEach.call(n.attributes, function (a) {
        if (a.value.indexOf("url(#") >= 0) {
          n.setAttribute(a.name, a.value.replace(/url\(#([^)]+)\)/g, function (m, id) {
            return "url(#" + id + suffix + ")";
          }));
        }
      });
    });
  }

  function syncDay() {
    var layer = document.getElementById("daylayer");
    var corona = document.getElementById("corona");
    var map = document.getElementById("map");
    if (!layer || !corona || !map) { return; }

    layer.textContent = "";
    [corona, map].forEach(function (src) {
      var copy = src.cloneNode(true);
      copy.removeAttribute("id");
      copy.setAttribute("aria-hidden", "true");
      copy.setAttribute("focusable", "false");
      // Duplicate ids in one document resolve to whichever came first, which
      // would point the day copy at the night layer's gradients and hand it
      // the night palette. Every id in the copy gets a suffix.
      reid(copy, "-day");
      Array.prototype.forEach.call(copy.querySelectorAll("[tabindex]"), function (n) {
        n.setAttribute("tabindex", "-1");
      });
      layer.appendChild(copy);
    });
    dayDirty = false;
  }

  function ensureDay() {
    if (!dayDirty) { return; }
    syncDay();
  }

  function watchMap() {
    var map = document.getElementById("map");
    if (!map || typeof MutationObserver === "undefined") { return; }
    new MutationObserver(function () {
      // applyFilter writes classes back into the map, so its own mutations
      // must not start another pass.
      if (filtering) { return; }
      dayDirty = true;
      applyFilter();
      if (wipe < 100) {
        if (daySyncTimer) { clearTimeout(daySyncTimer); }
        daySyncTimer = setTimeout(ensureDay, 120);
      }
    }).observe(map, { childList: true, subtree: true, attributes: true });
  }

  // ---------------------------------------------------------------- divider

  var wipe = 100;

  function setWipe(v) {
    wipe = Math.max(0, Math.min(100, v));
    var stage = document.getElementById("stage");
    var handle = document.getElementById("divider");
    if (stage) { stage.style.setProperty("--wipe", wipe + "%"); }
    if (handle) {
      handle.setAttribute("aria-valuenow", String(Math.round(wipe)));
      handle.setAttribute("aria-valuetext",
        wipe >= 99.5 ? "night" : (wipe <= 0.5 ? "day" : Math.round(wipe) + " percent night"));
    }
    if (wipe < 100) { ensureDay(); }
  }

  // On release the divider settles to whichever scheme it was left nearest,
  // so the page is always in one scheme or the other once the reader lets go.
  function settle() {
    var target = wipe < 50 ? 0 : 100;
    if (reduced) { setWipe(target); return; }
    var from = wipe;
    var start = 0;
    function frame(t) {
      if (!start) { start = t; }
      var k = Math.min(1, (t - start) / 320);
      var eased = 1 - Math.pow(1 - k, 3);
      setWipe(from + (target - from) * eased);
      if (k < 1) { requestAnimationFrame(frame); }
    }
    requestAnimationFrame(frame);
  }

  function wireDivider() {
    var handle = document.getElementById("divider");
    var stage = document.getElementById("stage");
    if (!handle || !stage) { return; }

    function fromEvent(e) {
      var box = stage.getBoundingClientRect();
      if (!box.width) { return; }
      setWipe(((e.clientX - box.left) / box.width) * 100);
    }

    handle.addEventListener("pointerdown", function (e) {
      e.preventDefault();
      handle.setPointerCapture(e.pointerId);
      handle.classList.add("dragging");
      ensureDay();
    });
    handle.addEventListener("pointermove", function (e) {
      if (handle.hasPointerCapture && handle.hasPointerCapture(e.pointerId)) { fromEvent(e); }
    });
    handle.addEventListener("pointerup", function (e) {
      if (handle.hasPointerCapture && handle.hasPointerCapture(e.pointerId)) {
        handle.releasePointerCapture(e.pointerId);
      }
      handle.classList.remove("dragging");
      settle();
    });

    // The arrow keys move it. Home, End and Space are stopped here rather than
    // passed on, because umbra.js binds those on the document for the replay
    // and a focused divider should not also scrub the sweep.
    handle.addEventListener("keydown", function (e) {
      var step = e.shiftKey ? 12 : 4;
      var handled = true;
      if (e.key === "ArrowLeft" || e.key === "ArrowDown") { ensureDay(); setWipe(wipe - step); }
      else if (e.key === "ArrowRight" || e.key === "ArrowUp") { ensureDay(); setWipe(wipe + step); }
      else if (e.key === "Home") { ensureDay(); setWipe(0); }
      else if (e.key === "End") { setWipe(100); }
      else if (e.key === "Enter" || e.key === " ") { settle(); }
      else { handled = false; }
      if (handled) {
        e.preventDefault();
        e.stopPropagation();
      }
    });
  }

  // ------------------------------------------------------------------ view

  function draw(data, label) {
    if (!globalThis.__umbra || !globalThis.__umbra.render) { return; }
    if (!globalThis.__umbra.render(data)) {
      msg("That file did not look like an Umbra report: it has no layout block.");
      return;
    }
    sample = data;
    hasAny = anyEvidence(data);
    maxSeq = data.timeline && data.timeline.length
      ? data.timeline[data.timeline.length - 1].seq : 0;

    drawCorona(data);
    showSaid(data);
    showContradiction(data);
    showStats(data);
    applyFilter();
    dayDirty = true;
    if (wipe < 100) { ensureDay(); }

    msg(label || "");
    var back = document.getElementById("back");
    if (back) { back.hidden = (data === original); }
  }

  function anyEvidence(data) {
    var nodes = data.nodes || [];
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].state !== "unknown") { return true; }
    }
    return nodes.length === 0;
  }

  var original = null;

  function readFile(file) {
    if (!file) { return; }
    if (file.size > MAX_BYTES) {
      msg("That file is " + Math.round(file.size / 1048576) + " MB. The viewer refuses anything over 20 MB.");
      return;
    }
    var reader = new FileReader();
    reader.onerror = function () { msg("That file could not be read."); };
    reader.onload = function () {
      var data;
      try {
        data = JSON.parse(reader.result);
      } catch (e) {
        msg("That file was not an Umbra report: it is not valid JSON.");
        return;
      }
      if (!data || typeof data !== "object" || !data.nodes || !data.layout) {
        msg("That file was not an Umbra report. An Umbra report has a nodes list and a layout block.");
        return;
      }
      draw(data, "Showing " + file.name + ". Nothing was uploaded.");
    };
    reader.readAsText(file);
  }

  // The second map is a report from a session in another project. It is drawn
  // once, without the replay controls, and never replaced by a dropped file.
  function drawImported() {
    var tag = document.getElementById("umbra-imported");
    var svg = document.getElementById("map-imported");
    if (!tag || !svg || !globalThis.__umbra || !globalThis.__umbra.renderInto) { return; }
    var data;
    try {
      data = JSON.parse(tag.textContent);
    } catch (e) {
      return;
    }
    if (!data || !data.layout) { return; }
    globalThis.__umbra.renderInto(svg, data);

    var said = document.getElementById("said-imported");
    if (said) {
      said.textContent = data.session_said
        ? "\u201C" + data.session_said + "\u201D"
        : "The session left no account of this change that Entire could store.";
    }
  }

  function boot() {
    reduced = !!(window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches);

    drawOrbs();
    drawImported();

    var tag = document.getElementById("umbra-data");
    try {
      original = JSON.parse(tag.textContent);
    } catch (e) {
      original = null;
    }
    if (original && original.layout) { draw(original, ""); }

    wireCopy();
    wireLegend();
    wireBeats();
    wireDivider();
    watchMap();

    var zone = document.getElementById("drop");
    var input = document.getElementById("file");
    var back = document.getElementById("back");

    if (zone) {
      ["dragenter", "dragover"].forEach(function (t) {
        zone.addEventListener(t, function (e) {
          e.preventDefault();
          zone.classList.add("over");
        });
      });
      ["dragleave", "drop"].forEach(function (t) {
        zone.addEventListener(t, function (e) {
          e.preventDefault();
          zone.classList.remove("over");
        });
      });
      zone.addEventListener("drop", function (e) {
        var files = e.dataTransfer && e.dataTransfer.files;
        if (files && files.length) { readFile(files[0]); }
      });
    }
    if (input) {
      input.addEventListener("change", function () { readFile(input.files[0]); });
    }
    if (back) {
      back.addEventListener("click", function () {
        if (original) { draw(original, "Back to the sample."); }
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
