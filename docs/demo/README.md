# Regenerating the gifs in docs/

The gifs are the TUI's own output, not screen captures: a test renders a
sample project after each key of two short sessions, and this directory
turns those frames into gifs. Re-run it whenever the screens change.

```
DEMO_FRAMES_DIR=/tmp/frames go test ./internal/tui/ -run TestDemoFrames
for s in run add; do
  python docs/demo/render.py /tmp/frames/$s /tmp/png-$s
  ffmpeg -y -f concat -safe 0 -i /tmp/png-$s/list.txt \
    -vf "fps=10,scale=1000:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=64[p];[b][p]paletteuse=dither=none" \
    -loop 0 docs/demo-$s.gif
done
mv docs/demo-run.gif docs/demo.gif
```

- `run` is docs/demo.gif (pick two commands, answer a placeholder, reach
  the review) and `add` is docs/demo-add.gif (write a command in the
  editor). Each session is a table of keys in
  `internal/tui/demo_frames_test.go`; the second number of a frame file
  name is its hold time in 1/100 s.
- `render.py` needs Pillow and a monospace font with box-drawing glyphs
  (`DEMO_FONT` overrides the default Cascadia Mono path). The image
  height follows the session's terminal height.
- The sample project lives in a temporary directory while recording;
  the renderer shows it as `C:\work\api` and refuses to draw a frame
  that still contains a temporary path.
