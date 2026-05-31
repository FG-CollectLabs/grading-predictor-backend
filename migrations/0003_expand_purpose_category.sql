-- +goose Up

-- Add grading_tracker to purpose values
ALTER TABLE certifications DROP CONSTRAINT certifications_purpose_check;
ALTER TABLE certifications ADD CONSTRAINT certifications_purpose_check
  CHECK (purpose IN ('analytics','buy_and_grade','crack_and_regrade','grading_tracker'));

-- Add BGS grades to category values
ALTER TABLE certifications DROP CONSTRAINT certifications_category_check;
ALTER TABLE certifications ADD CONSTRAINT certifications_category_check
  CHECK (category IN ('raw','psa9','psa10','cgc9','cgc10','bgs9','bgs9pt5','bgs10'));

-- +goose Down

ALTER TABLE certifications DROP CONSTRAINT certifications_purpose_check;
ALTER TABLE certifications ADD CONSTRAINT certifications_purpose_check
  CHECK (purpose IN ('analytics','buy_and_grade','crack_and_regrade'));

ALTER TABLE certifications DROP CONSTRAINT certifications_category_check;
ALTER TABLE certifications ADD CONSTRAINT certifications_category_check
  CHECK (category IN ('raw','psa9','psa10','cgc9','cgc10'));
