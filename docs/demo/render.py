"""Turn the frames written by TestDemoFrames into docs/demo.gif.

    python docs/demo/render.py FRAMES_DIR OUT_DIR

Each frame is the TUI's own output with ANSI colours; this draws it with
a monospace font and writes one PNG per frame plus an ffmpeg concat list
with the hold times. See README.md in this directory for the ffmpeg step.
"""
import glob
import os
import re
import sys

from PIL import Image, ImageDraw, ImageFont

frames_dir, out_dir = sys.argv[1], sys.argv[2]
os.makedirs(out_dir, exist_ok=True)

FONT_PATH = os.environ.get("DEMO_FONT", r"C:\Windows\Fonts\CascadiaMono.ttf")
FONT = ImageFont.truetype(FONT_PATH, 20)
COLS = 100
PAD = 24
BG = (30, 30, 46)
FG = (205, 214, 244)

CW = FONT.getlength("M")
ascent, descent = FONT.getmetrics()
CH = ascent + descent + 2


def xterm256(n):
    base = [(0, 0, 0), (205, 49, 49), (13, 188, 121), (229, 229, 16), (36, 114, 200), (188, 63, 188), (17, 168, 205), (229, 229, 229),
            (102, 102, 102), (241, 76, 76), (35, 209, 139), (245, 245, 67), (59, 142, 234), (214, 112, 214), (41, 184, 219), (255, 255, 255)]
    if n < 16:
        return base[n]
    if n < 232:
        n -= 16
        levels = [0, 95, 135, 175, 215, 255]
        return (levels[n // 36], levels[(n // 6) % 6], levels[n % 6])
    v = 8 + (n - 232) * 10
    return (v, v, v)


def cells(text):
    """Yield (row, col, char, fg, bg, bold) for every visible character."""
    fg, bg, bold, dim = FG, None, False, False
    row = col = 0
    i = 0
    while i < len(text):
        c = text[i]
        if c == "\x1b" and text[i + 1:i + 2] == "[":
            j = text.index("m", i)
            codes = [int(x) if x else 0 for x in text[i + 2:j].split(";")]
            k = 0
            while k < len(codes):
                code = codes[k]
                if code == 0:
                    fg, bg, bold, dim = FG, None, False, False
                elif code == 1:
                    bold = True
                elif code == 2:
                    dim = True
                elif code == 22:
                    bold = dim = False
                elif code == 38 and codes[k + 1] == 5:
                    fg = xterm256(codes[k + 2])
                    k += 2
                elif code == 48 and codes[k + 1] == 5:
                    bg = xterm256(codes[k + 2])
                    k += 2
                elif code == 38 and codes[k + 1] == 2:
                    fg = tuple(codes[k + 2:k + 5])
                    k += 4
                elif code == 48 and codes[k + 1] == 2:
                    bg = tuple(codes[k + 2:k + 5])
                    k += 4
                elif 30 <= code <= 37:
                    fg = xterm256(code - 30)
                elif 90 <= code <= 97:
                    fg = xterm256(code - 90 + 8)
                elif code == 39:
                    fg = FG
                elif code == 49:
                    bg = None
                k += 1
            i = j + 1
            continue
        if c == "\n":
            row, col = row + 1, 0
        elif c != "\r":
            shown = tuple(int(v * 0.55 + BG[n] * 0.45) for n, v in enumerate(fg)) if dim else fg
            yield row, col, c, shown, bg, bold
            col += 1
        i += 1


def render(text, path, rows):
    img = Image.new("RGB", (int(COLS * CW) + PAD * 2, rows * CH + PAD * 2), BG)
    draw = ImageDraw.Draw(img)
    for row, col, ch, fg, bg, bold in cells(text):
        if row >= rows or col >= COLS:
            continue
        x, y = PAD + col * CW, PAD + row * CH
        if bg:
            draw.rectangle([x, y, x + CW, y + CH], fill=bg)
        if ch != " ":
            draw.text((x, y + 1), ch, font=FONT, fill=fg)
    img.save(path)


# Every frame of a session has the terminal's height; take it from the tallest.
frames = sorted(glob.glob(os.path.join(frames_dir, "*.txt")))
rows = max(open(f, encoding="utf-8").read().count("\n") + 1 for f in frames)
entries = []
for f in frames:
    stem = os.path.splitext(os.path.basename(f))[0]
    hold = int(stem.split("_")[1]) / 100
    text = open(f, encoding="utf-8").read()
    # The sample project lives in a temporary directory; show it at a plain path.
    text = re.sub(r"[A-Za-z]:\\[^ \n]*?TestDemoFrames\d+\\001\\api", r"C:\\work\\api", text)
    assert "AppData" not in text and "/tmp/" not in text, "a temporary path is still visible in " + f
    png = os.path.join(out_dir, stem + ".png")
    render(text, png, rows)
    entries.append((os.path.abspath(png).replace("\\", "/"), hold))

with open(os.path.join(out_dir, "list.txt"), "w", encoding="utf-8") as fh:
    for png, hold in entries:
        fh.write("file '%s'\nduration %.2f\n" % (png, hold))
    fh.write("file '%s'\n" % entries[-1][0])
print("frames:", len(entries))
