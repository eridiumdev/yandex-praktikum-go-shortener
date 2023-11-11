CREATE TABLE IF NOT EXISTS shortlinks(
   id serial PRIMARY KEY,
   link_uid VARCHAR (8) UNIQUE NOT NULL,
   user_uid VARCHAR (32) NULL,
   short VARCHAR (32) NOT NULL,
   long VARCHAR (512) NOT NULL
);
