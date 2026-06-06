# Limitations & deviations

A few places where this project couldn't copy the two videos exactly:

- **`iddfs` / `idastar` / `prune` — distance estimate.** The videos guess "how far
  from solved?" by roughly counting wrong pieces. That guess was too rough here —
  the solver chased millions of dead ends and one mode ran 10+ minutes. So these
  three use pre-computed lookup tables instead, each more accurate than the last.

- **`sandwich` — success rate.** Rates come from actually running each solver, not
  the on-screen numbers, so some are lower (e.g. `sandwich` ~45% here vs ~72%).

- **`domino` / `iddfs` / `idastar` / `prune` / `multi` — move count.** Solving in
  two stages ("get halfway, then finish") is reliable but not the shortest path, so
  solutions run ~21–23 moves vs the video's ~20.
