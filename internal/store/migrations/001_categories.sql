CREATE TABLE categories (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER DEFAULT NULL,
  name TEXT NOT NULL CHECK (length(trim(name)) > 0 AND name = trim(name)),
  category_key TEXT NOT NULL UNIQUE CHECK (length(trim(category_key)) > 0),
  sort_order INTEGER NOT NULL DEFAULT 0
    CHECK (typeof(sort_order) = 'integer' AND sort_order >= 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (parent_id IS NULL OR parent_id <> id),
  FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_categories_root_name
  ON categories (name) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX idx_categories_child_name
  ON categories (parent_id, name) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_categories_parent_sort
  ON categories (parent_id, sort_order, id);

CREATE TRIGGER categories_check_parent_insert
BEFORE INSERT ON categories
WHEN NEW.parent_id IS NOT NULL
BEGIN
  SELECT RAISE(ABORT, 'parent must be a root category')
  WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE id = NEW.parent_id AND parent_id IS NULL
  );
END;

CREATE TRIGGER categories_check_parent_update
BEFORE UPDATE OF parent_id ON categories
WHEN NEW.parent_id IS NOT NULL
BEGIN
  SELECT RAISE(ABORT, 'parent must be a root category')
  WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE id = NEW.parent_id AND parent_id IS NULL
  );
  SELECT RAISE(ABORT, 'category with children must remain a root')
  WHERE EXISTS (SELECT 1 FROM categories WHERE parent_id = OLD.id);
END;

CREATE TRIGGER categories_keep_key
BEFORE UPDATE OF category_key ON categories
WHEN NEW.category_key IS NOT OLD.category_key
BEGIN
  SELECT RAISE(ABORT, 'category_key is immutable');
END;
