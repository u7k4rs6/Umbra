// Landing page behaviour: show the sample, and render any umbra.json the
// reader drops in. Nothing is uploaded and no request is made.

(function () {
  "use strict";

  var MAX_BYTES = 20 * 1024 * 1024;
  var sample = null;

  function msg(text) {
    var el = document.getElementById("dropmsg");
    if (el) { el.textContent = text || ""; }
  }

  function showSaid(data) {
    var el = document.getElementById("said");
    if (!el) { return; }
    el.textContent = data && data.session_said ? "Session said: " + data.session_said : "";
  }

  function draw(data, label) {
    if (!globalThis.__umbra || !globalThis.__umbra.render) { return; }
    if (!globalThis.__umbra.render(data)) {
      msg("That file did not look like an Umbra report: it has no layout block.");
      return;
    }
    showSaid(data);
    msg(label || "");
    var back = document.getElementById("back");
    if (back) { back.hidden = (data === sample); }
  }

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
        ? "Session said: " + data.session_said
        : "The session left no account of this change that Entire could store.";
    }
  }

  function boot() {
    drawImported();
    var tag = document.getElementById("umbra-data");
    try {
      sample = JSON.parse(tag.textContent);
    } catch (e) {
      sample = null;
    }
    if (sample && sample.layout) { draw(sample, ""); }

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
        if (sample) { draw(sample, "Back to the sample."); }
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
