#!/usr/bin/env python3
"""Generates the eight BUILDATHON.md images.

Deterministic: a fixed LCG seed drives every filament, so re-running produces
byte identical files. No network, no fonts, no scripts inside the output.
"""

import os
import math

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "img")
os.makedirs(OUT, exist_ok=True)

# ---------------------------------------------------------------- palettes

NIGHT = dict(
    ground="#000000",
    ink="#F2EBE3",
    dim="#8A7B6C",
    faint="#4A4038",
    hot="#FFD9A0",
    bright="#FFA33C",
    mid="#FF7A18",
    deep="#C2450A",
    pool="#0B0906",
    rule="#2A231C",
    disc="#000000",
)

DAY = dict(
    ground="#F7F1E8",
    ink="#17120E",
    dim="#6B5D50",
    faint="#BEB0A0",
    hot="#8A3606",
    bright="#C4510A",
    mid="#E0761C",
    deep="#F0A martian",  # replaced below
    pool="#D8CCBC",
    rule="#DED2C2",
    disc="#241C15",
)
DAY["deep"] = "#EFB877"

SERIF = "Georgia, 'Iowan Old Style', 'Palatino Linotype', 'Times New Roman', serif"
MONO = "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace"


class LCG:
    def __init__(self, seed):
        self.s = seed

    def next(self):
        self.s = (self.s * 1103515245 + 12345) % (1 << 31)
        return self.s / float(1 << 31)

    def between(self, a, b):
        return a + (b - a) * self.next()


def falloff_stops(color, peak):
    """Ten stops on (1-t)^3, alpha zero by 85 percent so there is no rim."""
    out = []
    for i in range(10):
        t = i / 9.0
        offset = t * 0.85
        alpha = peak * ((1.0 - t) ** 3)
        out.append(
            '<stop offset="%.4f" stop-color="%s" stop-opacity="%.4f"/>'
            % (offset, color, alpha)
        )
    out.append('<stop offset="1" stop-color="%s" stop-opacity="0"/>' % color)
    return "".join(out)


def corona(p, cx, cy, r, seed=20260906, count=44, night=True):
    """Filaments plus glow. Deterministic from seed."""
    rng = LCG(seed)
    parts = []
    for i in range(count):
        a = rng.between(0, math.tau)
        length = r * rng.between(0.22, 1.05)
        width = rng.between(1.0, 4.6)
        op = rng.between(0.10, 0.42)
        x1 = cx + math.cos(a) * (r * 0.99)
        y1 = cy + math.sin(a) * (r * 0.99)
        x2 = cx + math.cos(a) * (r + length)
        y2 = cy + math.sin(a) * (r + length)
        bow = rng.between(-0.22, 0.22)
        mx = (x1 + x2) / 2 - math.sin(a) * length * bow
        my = (y1 + y2) / 2 + math.cos(a) * length * bow
        parts.append(
            '<path d="M%.1f %.1f Q%.1f %.1f %.1f %.1f" stroke="url(#fil)" '
            'stroke-width="%.2f" fill="none" opacity="%.3f" stroke-linecap="round"/>'
            % (x1, y1, mx, my, x2, y2, width, op)
        )
    return "".join(parts)


def eclipse(p, cx, cy, r, seed=20260906, count=44, flare=True, gid=""):
    """Full eclipse unit: glow, filaments, occluding disc, limb flare."""
    g = gid
    s = []
    s.append(
        '<circle cx="%.1f" cy="%.1f" r="%.1f" fill="url(#glow%s)"/>'
        % (cx, cy, r * 3.1, g)
    )
    s.append('<g filter="url(#warp%s)">' % g)
    s.append(
        '<circle cx="%.1f" cy="%.1f" r="%.1f" fill="url(#ring%s)"/>'
        % (cx, cy, r * 1.75, g)
    )
    s.append(corona(p, cx, cy, r, seed=seed, count=count))
    s.append("</g>")
    # occluding disc
    s.append(
        '<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>' % (cx, cy, r, p["disc"])
    )
    s.append(
        '<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" '
        'stroke-width="1.1" opacity="0.55"/>' % (cx, cy, r, p["mid"])
    )
    if flare:
        fa = -0.72
        fx = cx + math.cos(fa) * (r * 1.34)
        fy = cy + math.sin(fa) * (r * 1.34)
        s.append(
            '<g filter="url(#soft%s)"><ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" opacity="0.55" '
            'fill="url(#flare%s)" transform="rotate(-41 %.1f %.1f)"/></g>'
            % (g, fx, fy, r * 0.78, r * 0.040, g, fx, fy)
        )
        s.append(
            '<circle cx="%.1f" cy="%.1f" r="%.1f" fill="url(#point%s)"/>'
            % (fx, fy, r * 0.30, g)
        )
    return "".join(s)


def defs(p, w, h, gid=""):
    g = gid
    return """<defs>
<radialGradient id="glow%s">%s</radialGradient>
<radialGradient id="ring%s"><stop offset="0.50" stop-color="%s" stop-opacity="0"/><stop offset="0.585" stop-color="%s" stop-opacity="0.75"/><stop offset="0.66" stop-color="%s" stop-opacity="0.34"/><stop offset="0.80" stop-color="%s" stop-opacity="0.10"/><stop offset="1" stop-color="%s" stop-opacity="0"/></radialGradient>
<linearGradient id="fil" x1="0" y1="0" x2="1" y2="0">
  <stop offset="0" stop-color="%s" stop-opacity="0.9"/>
  <stop offset="0.55" stop-color="%s" stop-opacity="0.5"/>
  <stop offset="1" stop-color="%s" stop-opacity="0"/>
</linearGradient>
<radialGradient id="point%s">
  <stop offset="0" stop-color="%s" stop-opacity="1"/>
  <stop offset="0.35" stop-color="%s" stop-opacity="0.85"/>
  <stop offset="1" stop-color="%s" stop-opacity="0"/>
</radialGradient>
<linearGradient id="flare%s" x1="0" y1="0" x2="1" y2="0">
  <stop offset="0" stop-color="%s" stop-opacity="0"/>
  <stop offset="0.5" stop-color="%s" stop-opacity="0.75"/>
  <stop offset="1" stop-color="%s" stop-opacity="0"/>
</linearGradient>
<filter id="warp%s" filterUnits="userSpaceOnUse" x="0" y="0" width="%d" height="%d">
  <feTurbulence type="fractalNoise" baseFrequency="0.011" numOctaves="3" seed="7" result="n"/>
  <feDisplacementMap in="SourceGraphic" in2="n" scale="9" xChannelSelector="R" yChannelSelector="G"/>
  <feGaussianBlur stdDeviation="1.1"/>
</filter>
<filter id="soft%s" filterUnits="userSpaceOnUse" x="0" y="0" width="%d" height="%d">
  <feGaussianBlur stdDeviation="9"/>
</filter>
<filter id="pool%s" filterUnits="userSpaceOnUse" x="0" y="0" width="%d" height="%d">
  <feGaussianBlur stdDeviation="9"/>
</filter>
</defs>""" % (
        g,
        falloff_stops(p["mid"], 0.55),
        g,
        p["bright"],
        p["hot"],
        p["bright"],
        p["mid"],
        p["mid"],
        p["hot"],
        p["mid"],
        p["deep"],
        g,
        p["hot"],
        p["bright"],
        p["mid"],
        g,
        p["hot"],
        p["hot"],
        p["hot"],
        g,
        w,
        h,
        g,
        w,
        h,
        g,
        w,
        h,
    )


def head(w, h, title, desc):
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" '
        'height="%d" role="img" aria-labelledby="t d">'
        '<title id="t">%s</title><desc id="d">%s</desc>' % (w, h, w, h, title, desc)
    )


def text(x, y, s, fill, size, family=MONO, weight="400", anchor="start", ls="0", op="1"):
    return (
        '<text x="%.1f" y="%.1f" fill="%s" font-family="%s" font-size="%s" '
        'font-weight="%s" text-anchor="%s" letter-spacing="%s" opacity="%s">%s</text>'
        % (x, y, fill, family, size, weight, anchor, ls, op, s)
    )


# ------------------------------------------------------------------- hero

def hero(p, night):
    W, H = 1200, 360
    s = [head(W, H, "Umbra",
              "A total eclipse. The dark disc is the changed symbol; the corona "
              "around it is the code that change reaches.")]
    s.append(defs(p, W, H))
    s.append('<rect width="%d" height="%d" fill="%s"/>' % (W, H, p["ground"]))
    s.append(eclipse(p, 872, 176, 94, seed=20260906, count=56))

    s.append(text(72, 168, "Umbra", p["ink"], "76", SERIF, "400", ls="-1"))
    s.append('<line x1="72" y1="196" x2="150" y2="196" stroke="%s" stroke-width="1.5"/>'
             % p["mid"])
    s.append(text(72, 236,
                  "the code your agent&#8217;s change reaches,",
                  p["dim"], "17", MONO, ls="0.4"))
    s.append(text(72, 262, "that your agent never looked at", p["dim"], "17", MONO,
                  ls="0.4"))
    s.append("</svg>")
    return "\n".join(s)


# ------------------------------------------------------- subtraction diagram

def subtraction(p, night):
    W, H = 1200, 420
    s = [head(W, H, "The subtraction",
              "Three panels. The blast radius of a change, minus the code the "
              "session examined, leaves the shadow.")]
    s.append(defs(p, W, H))
    s.append('<rect width="%d" height="%d" fill="%s"/>' % (W, H, p["ground"]))

    cy = 196
    panels = [
        (188, "Field", "everything the change reaches"),
        (600, "Examined", "what the session opened and read"),
        (1012, "Shadow", "what is left, and the tests in it"),
    ]

    rng = LCG(4242)
    pts = []
    for i in range(13):
        a = math.tau * i / 13.0 + 0.22
        rr = 96 + rng.between(-16, 16)
        pts.append((math.cos(a) * rr, math.sin(a) * rr * 0.80))

    # panel 1: the field, every dependent shown
    cx = 188
    s.append('<circle cx="%d" cy="%d" r="118" fill="none" stroke="%s" '
             'stroke-width="1" stroke-dasharray="2 6" opacity="0.6"/>'
             % (cx, cy, p["rule"]))
    s.append('<circle cx="%d" cy="%d" r="26" fill="%s"/>' % (cx, cy, p["disc"]))
    s.append('<circle cx="%d" cy="%d" r="26" fill="none" stroke="%s" stroke-width="1.4"/>'
             % (cx, cy, p["mid"]))
    for (dx, dy) in pts:
        s.append('<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" '
                 'stroke-width="0.8" opacity="0.30"/>'
                 % (cx, cy, cx + dx, cy + dy, p["mid"]))
        s.append('<circle cx="%.1f" cy="%.1f" r="6" fill="%s"/>'
                 % (cx + dx, cy + dy, p["bright"]))

    # panel 2: the examined set, a torch beam over part of the same field
    cx = 600
    s.append('<circle cx="%d" cy="%d" r="118" fill="none" stroke="%s" '
             'stroke-width="1" stroke-dasharray="2 6" opacity="0.6"/>'
             % (cx, cy, p["rule"]))
    s.append('<path d="M%d %d L%.1f %.1f A150 150 0 0 1 %.1f %.1f Z" fill="%s" '
             'opacity="0.16"/>'
             % (cx, cy,
                cx + math.cos(-1.30) * 150, cy + math.sin(-1.30) * 150,
                cx + math.cos(0.10) * 150, cy + math.sin(0.10) * 150,
                p["mid"]))
    s.append('<circle cx="%d" cy="%d" r="26" fill="%s"/>' % (cx, cy, p["disc"]))
    s.append('<circle cx="%d" cy="%d" r="26" fill="none" stroke="%s" stroke-width="1.4"/>'
             % (cx, cy, p["mid"]))
    for i, (dx, dy) in enumerate(pts):
        a = math.atan2(dy, dx)
        seen = -1.30 < a < 0.10
        col = p["hot"] if seen else p["faint"]
        op = "1" if seen else "0.55"
        s.append('<circle cx="%.1f" cy="%.1f" r="6" fill="%s" opacity="%s"/>'
                 % (cx + dx, cy + dy, col, op))

    # panel 3: the shadow, pools under everything the beam missed
    cx = 1012
    s.append('<circle cx="%d" cy="%d" r="118" fill="none" stroke="%s" '
             'stroke-width="1" stroke-dasharray="2 6" opacity="0.6"/>'
             % (cx, cy, p["rule"]))
    s.append('<g filter="url(#pool)">')
    for (dx, dy) in pts:
        a = math.atan2(dy, dx)
        if not (-1.30 < a < 0.10):
            s.append('<circle cx="%.1f" cy="%.1f" r="26" fill="%s" opacity="0.85"/>'
                     % (cx + dx, cy + dy, p["pool"]))
    s.append("</g>")
    s.append('<circle cx="%d" cy="%d" r="26" fill="%s"/>' % (cx, cy, p["disc"]))
    s.append('<circle cx="%d" cy="%d" r="26" fill="none" stroke="%s" stroke-width="1.4" '
             'opacity="0.5"/>' % (cx, cy, p["mid"]))
    for (dx, dy) in pts:
        a = math.atan2(dy, dx)
        if -1.30 < a < 0.10:
            s.append('<circle cx="%.1f" cy="%.1f" r="6" fill="%s" opacity="0.35"/>'
                     % (cx + dx, cy + dy, p["faint"]))
        else:
            s.append('<circle cx="%.1f" cy="%.1f" r="6" fill="none" stroke="%s" '
                     'stroke-width="1.6"/>' % (cx + dx, cy + dy, p["bright"]))

    # operators
    for x, ch in ((394, "&#8722;"), (806, "=")):
        s.append(text(x, 206, ch, p["dim"], "30", MONO, anchor="middle"))

    for (x, name, sub) in panels:
        s.append(text(x, 356, name, p["ink"], "20", SERIF, anchor="middle"))
        s.append(text(x, 382, sub, p["dim"], "13", MONO, anchor="middle", ls="0.3"))

    s.append("</svg>")
    return "\n".join(s)


# ----------------------------------------------------------------- result

def result(p, night):
    W, H = 1200, 400
    s = [head(W, H, "The demo run",
              "Eight probes selected, five cracked, full sweep with zero leaks, "
              "thirteen dependents all confirmed.")]
    s.append(defs(p, W, H))
    s.append('<rect width="%d" height="%d" fill="%s"/>' % (W, H, p["ground"]))

    s.append(text(72, 74, "The commit said the callers were checked and all tests pass.",
                  p["dim"], "17", MONO, ls="0.3"))

    # the five cracks, drawn as broken rings
    for i in range(13):
        x = 78 + i * 40
        y = 140
        cracked = i in (2, 5, 6, 9, 11)
        if cracked:
            s.append('<circle cx="%d" cy="%d" r="13" fill="none" stroke="%s" '
                     'stroke-width="2.2" stroke-dasharray="14 6" opacity="0.95"/>'
                     % (x, y, p["bright"]))
            s.append('<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" '
                     'stroke-width="2.2"/>' % (x - 8, y + 7, x + 8, y - 7, p["mid"]))
        else:
            s.append('<circle cx="%d" cy="%d" r="13" fill="none" stroke="%s" '
                     'stroke-width="1.2" opacity="0.7"/>' % (x, y, p["faint"]))
    s.append(text(78 + 13 * 40 + 14, 146, "13 dependents", p["dim"], "13", MONO))

    s.append('<line x1="72" y1="196" x2="1128" y2="196" stroke="%s" stroke-width="1"/>'
             % p["rule"])

    figs = [
        (72, "8", "probes selected"),
        (306, "5", "cracked"),
        (540, "0", "leaks in the full sweep"),
        (774, "13", "confirmed by the graph"),
    ]
    for (x, n, label) in figs:
        s.append(text(x, 268, n, p["ink"], "58", SERIF))
        s.append(text(x, 296, label, p["dim"], "13", MONO, ls="0.3"))

    names = ("test_apply_refund_caps_at_paid, test_apply_refund_normal, "
             "test_empty_is_zero, test_negative, test_rounding")
    s.append(text(72, 348, "every one in a file the session never opened",
                  p["mid"], "14", MONO, ls="0.3"))
    s.append(text(72, 372, names, p["faint"], "12", MONO, ls="0.2"))
    s.append("</svg>")
    return "\n".join(s)


# --------------------------------------------------------------- evidence

def evidence(p, night):
    W, H = 1200, 300
    s = [head(W, H, "Three kinds of evidence",
              "Confirmed, heuristic, and needs verification, shown as three "
              "eclipse orbs at falling certainty.")]
    s.append(defs(p, W, H))
    s.append('<rect width="%d" height="%d" fill="%s"/>' % (W, H, p["ground"]))
    s.append(text(72, 62, "Graph is evidence, not an oracle. Every dependent now "
                  "carries how well the graph resolved it.",
                  p["dim"], "15", MONO, ls="0.3"))

    orbs = [
        (222, 1.00, "confirmed",
         "the graph resolved every relation on this path"),
        (600, 0.46, "heuristic",
         "the graph derived this, it may be wrong"),
        (978, 0.16, "needs verification",
         "check it against the source; the command is printed"),
    ]
    for (cx, strength, name, meaning) in orbs:
        cy, r = 168, 40
        rng_seed = 990 + int(strength * 1000)
        s.append('<g opacity="%.2f">' % (0.30 + 0.70 * strength))
        s.append(eclipse(p, cx, cy, r, seed=rng_seed,
                         count=int(12 + 34 * strength), flare=False))
        s.append("</g>")
        s.append(text(cx, 248, name, p["ink"], "18", SERIF, anchor="middle"))
        s.append(text(cx, 272, meaning, p["dim"], "12", MONO, anchor="middle", ls="0.2"))
    s.append("</svg>")
    return "\n".join(s)


# ------------------------------------------------------------------ write

BUILDERS = {
    "hero": hero,
    "subtraction": subtraction,
    "result": result,
    "evidence": evidence,
}

for name, fn in BUILDERS.items():
    for scheme, pal in (("night", NIGHT), ("day", DAY)):
        path = os.path.join(OUT, "%s-%s.svg" % (name, scheme))
        with open(path, "w") as fh:
            fh.write(fn(pal, scheme == "night"))
        print("wrote", path)
