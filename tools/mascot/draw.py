"""Draws omnivore's bear mascot in several expressions, as transparent PNGs.

Usage: python3 tools/mascot/draw.py internal/ui/mascot/frames

Shapes are drawn on a 1000x1000 design grid at 4x supersampling, then downscaled, so edges are
smooth. Features are deliberately big and outlined so they read at ~70px in a terminal corner.
"""
import math
import os
import sys
from PIL import Image, ImageDraw

SS = 2  # supersampling: the canvas is 2000x2000 for a 1000-unit design
OUT = 128

INK = (43, 26, 16, 255)
FUR = (176, 116, 64, 255)
FUR_DARK = (138, 88, 48, 255)
EAR_IN = (232, 178, 128, 255)
MUZZLE = (244, 220, 186, 255)
BLUSH = (236, 140, 120, 150)
MOUTH = (96, 28, 30, 255)
TONGUE = (232, 102, 100, 255)
TOOTH = (255, 252, 244, 255)
WHITE = (255, 255, 255, 255)
ACCENT = (188, 140, 255, 255)
OUTLINE = 30


def S(v):
    return int(v * SS)


class Pen:
    def __init__(self):
        self.im = Image.new('RGBA', (S(1000), S(1000)), (0, 0, 0, 0))
        self.d = ImageDraw.Draw(self.im)

    def ell(self, cx, cy, rx, ry, fill=None, outline=INK, width=OUTLINE):
        self.d.ellipse([S(cx - rx), S(cy - ry), S(cx + rx), S(cy + ry)], fill=fill,
                       outline=outline if width else None, width=S(width) if width else 0)

    def blob(self, cx, cy, rx, ry, fill):
        """Filled ellipse with an outline drawn as a bigger ink ellipse behind it (cleaner joins)."""
        self.ell(cx, cy, rx + OUTLINE / 2, ry + OUTLINE / 2, fill=INK, width=0)
        self.ell(cx, cy, rx - OUTLINE / 2, ry - OUTLINE / 2, fill=fill, width=0)

    def line(self, pts, width=OUTLINE, fill=INK):
        self.d.line([(S(x), S(y)) for x, y in pts], fill=fill, width=S(width), joint='curve')
        r = width / 2
        for x, y in (pts[0], pts[-1]):
            self.ell(x, y, r, r, fill=fill, width=0)

    def arc(self, cx, cy, rx, ry, start, end, width=OUTLINE, fill=INK, steps=40):
        pts = []
        for i in range(steps + 1):
            a = math.radians(start + (end - start) * i / steps)
            pts.append((cx + rx * math.cos(a), cy + ry * math.sin(a)))
        self.line(pts, width=width, fill=fill)

    def poly(self, pts, fill, outline=INK, width=0):
        self.d.polygon([(S(x), S(y)) for x, y in pts], fill=fill)
        if width:
            self.line(list(pts) + [pts[0]], width=width, fill=outline)

    def done(self, rotate=0):
        im = self.im
        if rotate:
            im = im.rotate(rotate, resample=Image.Resampling.BICUBIC, center=(S(500), S(560)))
        return im.resize((OUT, OUT), Image.Resampling.LANCZOS)


def base(p, muzzle_drop=0):
    # Ears.
    for ex in (215, 785):
        p.blob(ex, 235, 135, 130, FUR_DARK)
        p.ell(ex + (35 if ex < 500 else -35), 255, 68, 66, fill=EAR_IN, width=0)
    # Head.
    p.blob(500, 560, 400, 355, FUR)
    # Muzzle.
    p.blob(500, 705 + muzzle_drop / 2, 190, 140 + muzzle_drop / 2, MUZZLE)
    # Blush.
    for bx in (255, 745):
        p.ell(bx, 650, 58, 34, fill=BLUSH, width=0)


def eyes(p, kind='open', look=(0, 0)):
    for i, ex in enumerate((350, 650)):
        if kind == 'open':
            p.ell(ex + look[0], 520 + look[1], 44, 52, fill=INK, width=0)
            p.ell(ex + look[0] + 14, 502 + look[1], 15, 15, fill=WHITE, width=0)
        elif kind == 'happy':
            p.arc(ex, 545, 52, 50, 200, 340, width=34)
        elif kind == 'angry':
            p.ell(ex, 540, 40, 34, fill=INK, width=0)
            p.ell(ex + 12, 528, 11, 11, fill=WHITE, width=0)
        elif kind == 'curious':
            big = i == 1
            p.ell(ex, 515, 50 if big else 40, 60 if big else 46, fill=INK, width=0)
            p.ell(ex + 15, 494, 17 if big else 13, 17 if big else 13, fill=WHITE, width=0)


def nose(p, y=620):
    p.ell(500, y, 72, 48, fill=INK, width=0)
    p.ell(478, y - 16, 20, 11, fill=(255, 255, 255, 170), width=0)


def mouth_closed(p):
    nose(p)
    p.line([(500, 665), (500, 705)], width=24)
    p.arc(455, 700, 45, 38, 10, 170, width=24)
    p.arc(545, 700, 45, 38, 10, 170, width=24)


def mouth_chomp(p):
    nose(p)
    # Open "nom" mouth.
    p.blob(500, 745, 88, 70, MOUTH)
    p.ell(500, 785, 58, 30, fill=TONGUE, width=0)
    p.poly([(455, 690), (485, 690), (470, 725)], TOOTH)
    p.poly([(515, 690), (545, 690), (530, 725)], TOOTH)


def mouth_roar(p):
    nose(p, y=600)
    p.blob(500, 760, 150, 120, MOUTH)
    p.ell(500, 820, 95, 45, fill=TONGUE, width=0)
    # Fangs top and bottom.
    for x in (395, 605):
        p.poly([(x - 28, 665), (x + 28, 665), (x, 740)], TOOTH)
    for x in (430, 570):
        p.poly([(x - 22, 860), (x + 22, 860), (x, 805)], TOOTH)


def mouth_happy(p):
    nose(p)
    # A wide "D" grin: the bottom half of an ellipse, outlined.
    box = [S(392), S(620), S(608), S(820)]
    p.d.chord([S(392 - 15), S(620 - 15), S(608 + 15), S(820 + 15)], 0, 180, fill=INK)
    p.d.chord(box, 0, 180, fill=MOUTH)
    p.d.chord([S(440), S(730), S(560), S(812)], 0, 180, fill=TONGUE)
    p.line([(385, 720), (615, 720)], width=26)


def mouth_o(p):
    nose(p)
    p.blob(535, 735, 36, 40, MOUTH)


def brows(p, kind):
    if kind == 'angry':
        p.line([(268, 430), (420, 478)], width=40)
        p.line([(732, 430), (580, 478)], width=40)
    elif kind == 'curious':
        p.line([(285, 455), (410, 450)], width=34)  # flat
        p.arc(650, 455, 80, 55, 200, 340, width=34)  # raised


def question(p, bob=0):
    # A "?" floating to the upper right.
    x, y = 915, 110 + bob
    p.arc(x, y, 55, 55, 190, 400, width=38, fill=ACCENT)
    p.line([(x + 28, y + 52), (x, y + 85), (x, y + 120)], width=38, fill=ACCENT)
    p.ell(x, y + 180, 23, 23, fill=ACCENT, width=0)


def frame(name):
    p = Pen()
    if name == 'chomp1':
        base(p); eyes(p); mouth_closed(p)
    elif name == 'chomp2':
        base(p, muzzle_drop=20); eyes(p); mouth_chomp(p)
    elif name == 'roar':
        base(p, muzzle_drop=60); eyes(p, 'angry'); brows(p, 'angry'); mouth_roar(p)
    elif name == 'happy':
        base(p); eyes(p, 'happy'); mouth_happy(p)
    elif name == 'curious1':
        base(p); eyes(p, 'curious', look=(10, -6)); brows(p, 'curious'); mouth_o(p)
        return p.done(rotate=-10), p
    elif name == 'curious2':
        base(p); eyes(p, 'curious', look=(14, -10)); brows(p, 'curious'); mouth_o(p)
        return p.done(rotate=-14), p
    return p.done(), p


if __name__ == '__main__':
    out = sys.argv[1] if len(sys.argv) > 1 else '.'
    for n in ['chomp1', 'chomp2', 'roar', 'happy', 'curious1', 'curious2']:
        im, _ = frame(n)
        if n.startswith('curious'):
            # Draw the "?" after rotating, so it stays upright.
            q = Pen()
            question(q, bob=0 if n == 'curious1' else -30)
            im = Image.alpha_composite(im, q.done())
        # A small palette keeps each frame to a few KB, since some terminals need it resent.
        im = im.quantize(colors=48, method=Image.Quantize.FASTOCTREE)
        im.save(os.path.join(out, n + '.png'), optimize=True)
