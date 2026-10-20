#!/usr/bin/env python3
"""Баннер RP Reel для карточки репозитория: 1600x520."""
from PIL import Image, ImageDraw, ImageFont

W, H = 1600, 520
BG = (0xF4, 0xF5, 0xF7)
CARD = (0xFF, 0xFF, 0xFF)
TEXT = (0x1B, 0x1F, 0x24)
DIM = (0x63, 0x6B, 0x75)
ACCENT = (0x2F, 0x5D, 0x8C)
RED = (0xB3, 0x26, 0x1E)
AMBER = (0xB2, 0x6A, 0x00)
BLUE = (0x1A, 0x5F, 0xB4)
PURPLE = (0x6A, 0x3F, 0xA0)

img = Image.new("RGB", (W, H), BG)
d = ImageDraw.Draw(img)


def font(size, bold=False):
    for path in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" if bold else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/dejavu/DejaVuSans.ttf",
    ):
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


# заголовок
d.text((70, 64), "RP Reel", font=font(76, bold=True), fill=TEXT)
d.text((72, 152), "нарезка моментов RP-сервера из видео и журнала событий",
       font=font(30), fill=DIM)

# плашки событий
chips = [
    ("убийство", RED, 70, 230),
    ("разбивка", AMBER, 250, 230),
    ("арест", BLUE, 440, 230),
    ("ограбление", PURPLE, 590, 230),
]
for label, col, x, y in chips:
    f = font(24, bold=True)
    tw = d.textlength(label, font=f)
    d.rounded_rectangle([x, y, x + tw + 36, y + 46], radius=23, fill=col)
    d.text((x + 18, y + 9), label, font=f, fill=(0xFF, 0xFF, 0xFF))

# таймлайн-плашка
d.rounded_rectangle([70, 320, 1530, 470], radius=16, fill=CARD, outline=(0xDF, 0xE3, 0xE8), width=2)
d.text((96, 344), "таймлайн", font=font(22, bold=True), fill=DIM)
# полоска ленты
d.rounded_rectangle([96, 384, 1504, 392], radius=4, fill=(0xEC, 0xF1, 0xF6))
marks = [(210, RED), (430, RED), (470, AMBER), (700, BLUE), (980, PURPLE), (1240, RED), (1400, AMBER)]
for x, col in marks:
    d.ellipse([x - 8, 380, x + 8, 396], fill=col)
d.text((96, 414), "каждая метка - событие с сервера, клик - к кадру, одна кнопка - ролик",
       font=font(24), fill=TEXT)

# правый верхний угол: без перекодирования
f = font(26, bold=True)
note = "без перекодирования: 4K режется мгновенно"
d.rounded_rectangle([W - d.textlength(note, font=f) - 106, 64, W - 40, 118], radius=12, fill=ACCENT)
d.text((W - d.textlength(note, font=f) - 80, 76), note, font=f, fill=(0xFF, 0xFF, 0xFF))

img.save("assets/banner.png")
print("banner ok:", img.size)
