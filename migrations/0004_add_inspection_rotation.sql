ALTER TABLE inspections
  ADD COLUMN IF NOT EXISTS centering_front_rotation NUMERIC(5,2),
  ADD COLUMN IF NOT EXISTS centering_back_rotation  NUMERIC(5,2);
