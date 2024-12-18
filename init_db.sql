USE mydb;

CREATE TABLE IF NOT EXISTS users  (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    token VARCHAR(225),
    `role` INT default 0
);

INSERT INTO users(username, password, `role`)
VALUES ('staff1', 'staff1', 0);

INSERT INTO users(username, password, `role`)
VALUES ('maanger1', 'maanger1', 0);

INSERT INTO users(username, password, `role`)
VALUES ('maanger2', 'maanger2', 0);

INSERT INTO users(username, password, `role`)
VALUES ('maanger3', 'maanger3', 0);