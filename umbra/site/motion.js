// The motion layer.
//
// Everything here is decoration in service of reading: content arrives out of
// shadow, the pointer carries a torch, the rail says where you are, and the
// six ranking factors light one at a time as you pass them. None of it changes
// a number, and none of it is required to read the page.
//
// Three rules it keeps. It touches only opacity, transform and filter, so
// nothing here forces a layout. It does nothing at all under
// prefers-reduced-motion, and the stylesheet resolves every reveal to its
// finished state in that case. And it is gated behind a .js class added here,
// so a reader with script off gets a page that is simply visible rather than a
// page waiting to be told it may appear.

(function () {
  "use strict";

  var reduced = !!(window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches);

  var root = document.documentElement;
  var all = function (sel, ctx) {
    return Array.prototype.slice.call((ctx || document).querySelectorAll(sel));
  };

  // ------------------------------------------------------------- the reveals

  // The hero is staged rather than revealed: its five pieces arrive in reading
  // order, a beat apart, which is the difference between a page that appears
  // and a page that assembles.
  function stageHero() {
    var staged = all("[data-stage]").sort(function (a, b) {
      return (+a.getAttribute("data-stage")) - (+b.getAttribute("data-stage"));
    });
    staged.forEach(function (el, i) {
      el.style.setProperty("--d", (0.06 + i * 0.09).toFixed(2) + "s");
    });
    // Two frames, so the first paint has already happened with the finished
    // layout in place and only the transition runs.
    requestAnimationFrame(function () {
      requestAnimationFrame(function () {
        staged.forEach(function (el) { el.classList.add("in"); });
      });
    });
  }

  // The headline arrives a word at a time, each one rising out of its own
  // clipped line. The text nodes are untouched: every word keeps its own
  // characters in order and the accessible name of the heading does not
  // change, which is the difference between a text animation and a text
  // animation that breaks a screen reader.
  function splitHeadline() {
    var h = document.querySelector("h1[data-stage]");
    if (!h || h.getAttribute("data-split") === "1") { return; }

    var pieces = [];
    (function walk(node) {
      Array.prototype.slice.call(node.childNodes).forEach(function (n) {
        if (n.nodeType === 3) {
          n.nodeValue.split(/(\s+)/).forEach(function (part) {
            if (part === "") { return; }
            pieces.push({ text: part, em: node !== h });
          });
        } else if (n.nodeType === 1) {
          walk(n);
        }
      });
    })(h);

    if (!pieces.length) { return; }
    h.textContent = "";
    var i = 0;
    pieces.forEach(function (piece) {
      if (/^\s+$/.test(piece.text)) {
        h.appendChild(document.createTextNode(" "));
        return;
      }
      var line = document.createElement("span");
      line.className = "wline";
      var word = document.createElement(piece.em ? "em" : "span");
      word.className = "word-in";
      word.textContent = piece.text;
      word.style.setProperty("--wd", (0.10 + i * 0.055).toFixed(3) + "s");
      i++;
      line.appendChild(word);
      h.appendChild(line);
    });
    h.setAttribute("data-split", "1");
    requestAnimationFrame(function () {
      requestAnimationFrame(function () { h.classList.add("words-in"); });
    });
  }

  function wireReveals() {
    var bands = all("[data-reveal]");
    if (!("IntersectionObserver" in window)) {
      bands.forEach(function (el) { el.classList.add("in"); });
      return;
    }
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) { return; }
        e.target.classList.add("in");
        io.unobserve(e.target);
      });
    }, { rootMargin: "0px 0px -12% 0px", threshold: 0.06 });
    bands.forEach(function (el) { io.observe(el); });
  }

  // --------------------------------------------------------------- the torch

  function wireTorch() {
    all("[data-torch]").forEach(function (el) {
      el.addEventListener("pointermove", function (e) {
        var r = el.getBoundingClientRect();
        el.style.setProperty("--mx", (e.clientX - r.left) + "px");
        el.style.setProperty("--my", (e.clientY - r.top) + "px");
      });
    });
  }

  // ------------------------------------------------------- scroll driven bits

  var topbar = document.getElementById("topbar");
  var progress = document.querySelector(".progress");
  var stage = document.getElementById("stage");
  var spyLinks = all(".pill a[data-spy]");
  var factors = all("#factors li");
  var sections = spyLinks.map(function (a) {
    return document.getElementById(a.getAttribute("data-spy"));
  });

  var ticking = false;

  function onScroll() {
    if (ticking) { return; }
    ticking = true;
    requestAnimationFrame(function () {
      ticking = false;
      var y = window.scrollY || window.pageYOffset || 0;
      var vh = window.innerHeight || 1;

      if (topbar) { topbar.classList.toggle("stuck", y > 24); }

      if (progress) {
        var max = Math.max(1, document.documentElement.scrollHeight - vh);
        progress.style.setProperty("--read", ((y / max) * 100).toFixed(2) + "%");
      }

      // The hero recedes over its own height, and stops at 1 rather than
      // running past it.
      if (stage) {
        var p = Math.max(0, Math.min(1, y / (vh * 0.85)));
        root.style.setProperty("--hero-p", p.toFixed(3));
      }

      // The rail follows the section that owns the reading line, a third of
      // the way down the viewport, rather than whatever happens to be visible.
      var line = y + vh * 0.34;
      var active = -1;
      sections.forEach(function (sec, i) {
        if (sec && sec.offsetTop <= line) { active = i; }
      });
      spyLinks.forEach(function (a, i) { a.classList.toggle("on", i === active); });

      // One factor at a time, the one nearest the same reading line.
      if (factors.length) {
        var best = -1, bestD = Infinity;
        factors.forEach(function (li, i) {
          var box = li.getBoundingClientRect();
          var mid = box.top + box.height / 2;
          var d = Math.abs(mid - vh * 0.45);
          if (d < bestD) { bestD = d; best = i; }
        });
        factors.forEach(function (li, i) {
          li.classList.toggle("lit", i === best && bestD < vh * 0.42);
        });
      }
    });
  }

  // ------------------------------------------------------------- the numbers

  // The counts in the contradiction line count up when the hero arrives. The
  // numbers are the report's own; this only changes how they are delivered,
  // and it ends on the exact value site.js wrote.
  function countUp() {
    var line = document.getElementById("contradiction");
    var targets = [];
    if (line) { targets = targets.concat(all("b", line)); }
    targets = targets.concat(all(".stat dd, .card .count"));
    targets.forEach(function (b, i) {
      var target = parseInt(b.textContent, 10);
      if (!isFinite(target) || target <= 0) { return; }
      var started = null;
      var dur = 620 + (i % 5) * 90;
      b.textContent = "0";
      function frame(t) {
        if (!started) { started = t; }
        var k = Math.min(1, (t - started) / dur);
        var eased = 1 - Math.pow(1 - k, 3);
        b.textContent = String(Math.round(target * eased));
        if (k < 1) { requestAnimationFrame(frame); }
        else { b.textContent = String(target); }
      }
      requestAnimationFrame(frame);
    });
  }

  // ------------------------------------------------------------------- boot

  function boot() {
    // The js class is set in the head, before first paint, so the reveal rules
    // are already in force when the page is first drawn and nothing flashes
    // visible and then hides itself.
    root.classList.add("js");
    if (reduced) {
      // The stylesheet has already resolved every reveal. Wire the rail so it
      // still says where you are, and nothing else.
      wireSpyOnly();
      return;
    }
    splitHeadline();
    stageHero();
    wireReveals();
    wireTorch();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    onScroll();
    // site.js fills the contradiction line on its own boot, so wait a frame
    // for it rather than racing it.
    setTimeout(countUp, 260);
  }

  function wireSpyOnly() {
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
