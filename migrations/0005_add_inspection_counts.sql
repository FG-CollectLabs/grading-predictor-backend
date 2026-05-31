ALTER TABLE inspections
  ADD COLUMN IF NOT EXISTS corners_defective_cut   SMALLINT,
  ADD COLUMN IF NOT EXISTS corners_major_whitening  SMALLINT,
  ADD COLUMN IF NOT EXISTS corners_minor_whitening  SMALLINT,
  ADD COLUMN IF NOT EXISTS corners_micro_whitening  SMALLINT,
  ADD COLUMN IF NOT EXISTS edges_whitening          SMALLINT,
  ADD COLUMN IF NOT EXISTS surface_dead_pixels      SMALLINT,
  ADD COLUMN IF NOT EXISTS surface_dimples          SMALLINT,
  ADD COLUMN IF NOT EXISTS surface_print_lines      SMALLINT;
