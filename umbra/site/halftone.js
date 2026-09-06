// The halftone field.
//
// A grid of dots whose size and warmth come from a light field: bright and
// wide near the source, small and cold far from it, with a slow travelling
// wave so the field is never quite still and a swell under the pointer.
//
// The idea is a halftone background of the kind component libraries ship as a
// Three.js point cloud. This is Canvas 2D and about a hundred lines, because
// this site takes no dependency and makes no request. Nothing here is loaded
// from anywhere; it is drawn.
//
// It is also not only decoration. The field is Umbra's own subject: the
// dependents of a change, lit near the source and dark where nothing reached.
// The dots follow the same falloff the corona does.

(function () {
  "use strict";

  var PITCH = 26;          // distance between dots, in css pixels
  var MAX_R = 5.2;         // radius of the brightest dot
  var canvas, ctx, w, h, dpr, cols, rows, raf, t0;
  var reduced = !!(window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches);

  // Where the light is, in fractions of the viewport. The hero puts the
  // eclipse right of centre, so the field is brightest there and the copy on
  // the left sits on quiet ground.
  var light = { x: 0.68, y: 0.42 };
  var pointer = { x: -1, y: -1, on: false };

  function size() {
    dpr = Math.min(2, window.devicePixelRatio || 1);
    w = window.innerWidth;
    h = window.innerHeight;
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
    canvas.style.width = w + "px";
    canvas.style.height = h + "px";
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    cols = Math.ceil(w / PITCH) + 1;
    rows = Math.ceil(h / PITCH) + 1;
  }

  // The three stops of the ramp, so the field is the same hue as everything
  // else on the page.
  function warm(v) {
    // v runs 0 to 1. Below the first stop the dot is a cold ember, above the
    // second it is nearly white.
    var r, g, b;
    if (v < 0.5) {
      var k = v / 0.5;
      r = 122 + (255 - 122) * k;
      g = 47 + (163 - 47) * k;
      b = 8 + (60 - 8) * k;
    } else {
      var j = (v - 0.5) / 0.5;
      r = 255;
      g = 163 + (226 - 163) * j;
      b = 60 + (180 - 60) * j;
    }
    return "rgb(" + (r | 0) + "," + (g | 0) + "," + (b | 0) + ")";
  }

  function draw(time) {
    var t = (time - t0) / 1000;
    ctx.clearRect(0, 0, w, h);

    var lx = light.x * w;
    var ly = light.y * h;
    var reach = Math.max(w, h) * 0.62;

    for (var iy = 0; iy < rows; iy++) {
      var y = iy * PITCH;
      for (var ix = 0; ix < cols; ix++) {
        var x = ix * PITCH + (iy % 2 ? PITCH / 2 : 0);

        var dx = x - lx;
        var dy = y - ly;
        var d = Math.sqrt(dx * dx + dy * dy) / reach;

        // Falloff, the same shape the corona uses: fast at first, a long tail.
        var v = 1 - Math.min(1, d);
        v = v * v * v;

        // A slow wave travelling out from the light, so the field breathes
        // without anything moving.
        if (!reduced) {
          v *= 0.82 + 0.18 * Math.sin(d * 9 - t * 0.55);
        }

        // The pointer swells the dots it passes over.
        if (pointer.on) {
          var px = x - pointer.x;
          var py = y - pointer.y;
          var pd = Math.sqrt(px * px + py * py);
          if (pd < 190) {
            var k = 1 - pd / 190;
            v += k * k * 0.55;
          }
        }

        if (v <= 0.012) { continue; }
        v = Math.min(1, v);

        var r = MAX_R * v;
        if (r < 0.28) { continue; }

        ctx.globalAlpha = Math.min(0.92, 0.10 + v * 0.85);
        ctx.fillStyle = warm(v);
        ctx.beginPath();
        ctx.arc(x, y, r, 0, 6.2832);
        ctx.fill();
      }
    }
    ctx.globalAlpha = 1;

    if (!reduced) { raf = requestAnimationFrame(draw); }
  }

  function start() {
    canvas = document.createElement("canvas");
    canvas.className = "halftone";
    canvas.setAttribute("aria-hidden", "true");
    document.body.insertBefore(canvas, document.body.firstChild);
    ctx = canvas.getContext("2d");
    if (!ctx) { return; }

    size();
    t0 = performance.now();

    window.addEventListener("resize", function () {
      size();
      if (reduced) { draw(performance.now()); }
    });

    // The pointer moves the swell, not the light. The light stays where the
    // eclipse is.
    window.addEventListener("pointermove", function (e) {
      pointer.x = e.clientX;
      pointer.y = e.clientY;
      pointer.on = true;
    }, { passive: true });
    window.addEventListener("pointerleave", function () { pointer.on = false; });

    // A background that keeps painting in a hidden tab is a background that
    // costs somebody battery for nothing.
    document.addEventListener("visibilitychange", function () {
      if (document.hidden) {
        if (raf) { cancelAnimationFrame(raf); raf = 0; }
      } else if (!reduced && !raf) {
        t0 = performance.now();
        raf = requestAnimationFrame(draw);
      }
    });

    if (reduced) { draw(performance.now()); } else { raf = requestAnimationFrame(draw); }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
