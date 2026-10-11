# Regenerating docs/demo.gif

The gif is the TUI's own output, not a screen capture: a test renders a
sample project after each key of a short session, and this directory turns
those frames into a gif. Re-run it whenever the screens change.

```
DEMO_FRAMES_DIR=/tmp/frames go test ./internal/tui/ -run TestDemoFrames
python docs/demo/render.py /tmp/frames /tmp/png
ffmpeg -y -f concat -safe 0 -i /tmp/png/list.txt \
  -vf "fps=10,scale=1000:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=64[p];[b][p]paletteuse=dither=none" \
  -loop 0 docs/demo.gif
```

- The session is the `steps` table in `internal/tui/demo_frames_test.go`;
  the second number of each frame file name is its hold time in 1/100 s.
- `render.py` needs Pillow and a monospace font with box-drawing glyphs
  (`DEMO_FONT` overrides the default Cascadia Mono path).
- The sample project lives in a temporary directory while recording;
  the renderer shows it as `C:\work\api` and refuses to draw a frame
  that still contains a temporary path.
