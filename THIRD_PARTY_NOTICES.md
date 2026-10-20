# Сторонние компоненты

## ffmpeg + ffprobe

Приложение вызывает их как отдельные программы, внутри себя не несет.
Их нужно поставить самому: https://ffmpeg.org/download.html

ffmpeg распространяется под LGPL/GPL в зависимости от сборки.
Если соберешь свою сборку ffmpeg с GPL-компонентами, распространяй
её по GPL. Стоковые сборки с официального сайта - LGPL.

## Шрифт Inter

В GUI встроен Inter Variable (SIL Open Font License 1.1):
https://github.com/googlefonts/inter/blob/main/OFL.txt

## Gio

Интерфейс рисован на Gio (BSD-3): https://gioui.org

Всё остальное - MIT, см. LICENSE.
