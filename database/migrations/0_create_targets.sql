CREATE TABLE targets (
  pk integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  id uuid NOT NULL,
  name varchar(255),
  uri varchar(255) NOT NULL,
  period unsigned int NOT NULL,
  config json NOT NULL DEFAULT '{}',
  created_at integer NOT NULL,
  updated_at integer NOT NULL
);

CREATE UNIQUE INDEX targets_on_id ON targets (id);
