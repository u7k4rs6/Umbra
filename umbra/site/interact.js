// The interaction layer.
//
// A torch that follows the pointer with a little lag, a trail of embers behind
// it, cards that tilt toward it, and controls that lean into it. All of it is
// transform and opacity, none of it is required to read the page, and all of
// it is off under prefers-reduced-motion and on touch, where there is no
// pointer to follow.

(function () {
  "use strict";

  var reduced = !!(window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches);
  var fine = !!(window.matchMedia && window.matchMedia("(pointer: fine)").matches);
  if (reduced || !fine) { return; }

  var all = function (sel) { return Array.prototype.slice.call(document.querySelectorAll(sel)); };

  // ------------------------------------------------------------ the torch

  // Two rings: one that snaps to the pointer and one that trails it. The lag
  // is what makes it read as a thing being carried rather than a cursor being
  // replaced.
  var dot = document.createElement("div");
  dot.className = "torch-dot";
  dot.setAttribute("aria-hidden", "true");
  var ring = document.createElement("div");
  ring.className = "torch-ring";
  ring.setAttribute("aria-hidden", "true");
  document.body.appendChild(dot);
  document.body.appendChild(ring);

  var px = window.innerWidth / 2, py = window.innerHeight / 2;
  var rx = px, ry = py;
  var down = false, over = false;

  // ------------------------------------------------------------ the embers

  // The trail is drawn into its own canvas so it can glow over everything
  // without touching the layout.
  var canvas = document.createElement("canvas");
  canvas.className = "torch-trail";
  canvas.setAttribute("aria-hidden", "true");
  document.body.appendChild(canvas);
  var ctx = canvas.getContext("2d");
  var dpr = 1, cw = 0, ch = 0;
  var embers = [];

  function size() {
    dpr = Math.min(2, window.devicePixelRatio || 1);
    cw = window.innerWidth;
    ch = window.innerHeight;
    canvas.width = Math.round(cw * dpr);
    canvas.height = Math.round(ch * dpr);
    canvas.style.width = cw + "px";
    canvas.style.height = ch + "px";
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }
  size();
  window.addEventListener("resize", size);

  var lastX = px, lastY = py;

  window.addEventListener("pointermove", function (e) {
    px = e.clientX;
    py = e.clientY;

    // One ember per few pixels travelled, so the trail is a function of speed
    // rather than of frame rate.
    var dx = px - lastX, dy = py - lastY;
    var d = Math.sqrt(dx * dx + dy * dy);
    if (d > 6 && embers.length < 90) {
      embers.push({
        x: px, y: py,
        vx: dx * 0.04 + (Math.random() - 0.5) * 0.4,
        vy: dy * 0.04 + (Math.random() - 0.5) * 0.4 - 0.16,
        life: 1,
        r: 1.4 + Math.random() * 2.6
      });
      lastX = px; lastY = py;
    }
  }, { passive: true });

  window.addEventListener("pointerdown", function () { down = true; });
  window.addEventListener("pointerup", function () { down = false; });
  document.addEventListener("pointerleave", function () {
    dot.style.opacity = "0";
    ring.style.opacity = "0";
  });
  document.addEventListener("pointerenter", function () {
    dot.style.opacity = "";
    ring.style.opacity = "";
  });

  // The torch grows over anything you can act on, which is the whole point of
  // carrying one.
  var HOT = "a, button, input, [role=slider], .card, .tile, summary";
  document.addEventListener("pointerover", function (e) {
    if (e.target.closest && e.target.closest(HOT)) { over = true; }
  });
  document.addEventListener("pointerout", function (e) {
    if (e.target.closest && e.target.closest(HOT)) { over = false; }
  });

  function frame() {
    // The ring chases the pointer; the dot is already there.
    rx += (px - rx) * 0.16;
    ry += (py - ry) * 0.16;

    var scale = over ? 2.1 : 1;
    if (down) { scale *= 0.78; }
    dot.style.transform = "translate3d(" + px + "px," + py + "px,0) translate(-50%,-50%)";
    ring.style.transform = "translate3d(" + rx + "px," + ry + "px,0) translate(-50%,-50%) scale(" + scale.toFixed(3) + ")";
    ring.classList.toggle("hot", over);

    ctx.clearRect(0, 0, cw, ch);
    for (var i = embers.length - 1; i >= 0; i--) {
      var p = embers[i];
      p.x += p.vx;
      p.y += p.vy;
      p.vy -= 0.012;          // embers rise
      p.vx *= 0.97;
      p.vy *= 0.985;
      p.life -= 0.018;
      if (p.life <= 0) { embers.splice(i, 1); continue; }

      var a = p.life * p.life;
      var r = p.r * (0.5 + p.life);
      var g = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, r * 4);
      g.addColorStop(0, "rgba(255,226,180," + (a * 0.9).toFixed(3) + ")");
      g.addColorStop(0.4, "rgba(255,163,60," + (a * 0.45).toFixed(3) + ")");
      g.addColorStop(1, "rgba(229,102,27,0)");
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(p.x, p.y, r * 4, 0, 6.2832);
      ctx.fill();
    }
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);

  // ------------------------------------------------------------ the tilt

  // Cards lean toward the pointer. Four degrees is enough to read as depth and
  // little enough that text stays sharp.
  function wireTilt() {
    all(".card, .tile, .panel").forEach(function (el) {
      el.addEventListener("pointermove", function (e) {
        var r = el.getBoundingClientRect();
        var mx = (e.clientX - r.left) / r.width;
        var my = (e.clientY - r.top) / r.height;
        el.style.setProperty("--mx", (mx * 100) + "%");
        el.style.setProperty("--my", (my * 100) + "%");
        el.style.setProperty("--ry", ((mx - 0.5) * 8).toFixed(2) + "deg");
        el.style.setProperty("--rx", ((0.5 - my) * 6).toFixed(2) + "deg");
      });
      el.addEventListener("pointerleave", function () {
        el.style.setProperty("--ry", "0deg");
        el.style.setProperty("--rx", "0deg");
      });
    });
  }

  // ---------------------------------------------------------- the magnets

  // Small controls lean into the pointer as it approaches, which is what makes
  // a button feel like it wants to be pressed.
  function wireMagnets() {
    all(".install button, .ghost, .pill a, .replay .replay-controls button, .filters button")
      .forEach(function (el) {
        el.addEventListener("pointermove", function (e) {
          var r = el.getBoundingClientRect();
          var dx = (e.clientX - (r.left + r.width / 2)) / r.width;
          var dy = (e.clientY - (r.top + r.height / 2)) / r.height;
          el.style.transform = "translate(" + (dx * 7).toFixed(2) + "px," + (dy * 5).toFixed(2) + "px)";
        });
        el.addEventListener("pointerleave", function () { el.style.transform = ""; });
      });
  }

  function boot() {
    document.documentElement.classList.add("has-torch");
    wireTilt();
    wireMagnets();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
