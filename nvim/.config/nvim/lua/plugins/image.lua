-- Inline image rendering in markdown. Uses image.nvim rather than snacks.image
-- because we run inside tmux: image.nvim places images by absolute position
-- (works through tmux's `allow-passthrough on`), whereas snacks relies on Kitty
-- unicode placeholders, which tmux does not pass through (so snacks falls back
-- to a cursor-only float). Backend is Ghostty's Kitty graphics protocol; the
-- "magick_cli" processor shells out to ImageMagick's `convert` (no luarocks).
return {
  "3rd/image.nvim",
  ft = { "markdown" },
  opts = {
    backend = "kitty",
    processor = "magick_cli",
    integrations = {
      markdown = {
        enabled = true,
        -- Render every image in the buffer, always - not just at the cursor.
        only_render_image_at_cursor = false,
        -- Keep images visible while editing.
        clear_in_insert_mode = false,
        filetypes = { "markdown" },
      },
    },
    max_width = 100,
    max_height = 20,
    -- Hide images when another window covers them or when this tmux window
    -- isn't active, so they don't bleed across panes/windows.
    window_overlap_clear_enabled = true,
    tmux_show_only_in_active_window = true,
  },
}
