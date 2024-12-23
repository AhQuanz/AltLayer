CREATE DATABASE IF NOT EXISTS mydb;
USE mydb;

CREATE TABLE IF NOT EXISTS users  (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    token VARCHAR(225) NOT NULL default(''),
    `role` INT default 0,
    account_number VARCHAR(255) NOT NULL
);

INSERT INTO users(username, password, `role`, account_number)
VALUES ('staff1', 'staff1', 0, '0x00b27b9756e40353e84e1fb78e764993b1e03d95');

INSERT INTO users(username, password, `role`, account_number)
VALUES ('staff2', 'staff2', 0, '0x3e77cfd8f351b22d642879418c0a9a700ca499dc');

INSERT INTO users(username, password, `role`, account_number)
VALUES ('manager1', 'manager1', 1, '0x44a446b4ca373b108ad6352525fb39df1a3a4dc7');

INSERT INTO users(username, password, `role`, account_number)
VALUES ('manager2', 'manager2', 1, '0x4b6b5998576ecb9dda4773077064323e46d7d447');

INSERT INTO users(username, password, `role`, account_number)
VALUES ('manager3', 'manager3', 1, '0x16ff98eb886ee4811370fa1a3c2d6a3d7957797a');

CREATE TABLE IF NOT EXISTS withdraw_claim  (
    id INT AUTO_INCREMENT PRIMARY KEY,
    claim_user_id INT NOT NULL,
    amount INT NOT NULL,
    claim_status INT default(0),
    transaction_hash VARCHAR(255),
    FOREIGN KEY (claim_user_id) REFERENCES users(id)
);

INSERT INTO withdraw_claim(id, claim_user_id, amount, claim_status)
VALUES (1, 1, 100, 4);

CREATE TABLE IF NOT EXISTS withdraw_claims_approval  (
    id INT AUTO_INCREMENT PRIMARY KEY,
    claim_id INT NOT NULL,
    approve_manager_id INT NOT NULL,
    FOREIGN KEY (approve_manager_id) REFERENCES users(id),
    FOREIGN KEY (claim_id) REFERENCES withdraw_claim(id)
);

INSERT INTO withdraw_claims_approval(id, claim_id, approve_manager_id)
VALUES (1, 1, 3);

INSERT INTO withdraw_claims_approval(id, claim_id, approve_manager_id)
VALUES (2, 1, 5);