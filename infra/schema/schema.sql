-- perfect-cat-streaming database schema (desired state).
--
-- Applied declaratively via mysqldef: it diffs this file against the live
-- database and emits the ALTER TABLE / CREATE TABLE statements required to
-- converge. Safe to re-run; no-op when already in sync.
--
-- Local: `make schema-apply` (uses the `schema` compose service)
-- Prod:  run mysqldef from your CI/admin host against the RDS endpoint with
--        the credentials from Secrets Manager.

CREATE TABLE `videos` (
    `id`           VARCHAR(64)  NOT NULL,
    `title`        VARCHAR(255) NOT NULL,
    `description`  TEXT         NOT NULL,
    `cat_name`     VARCHAR(64)  NOT NULL,
    `breed`        VARCHAR(32)  NOT NULL,
    `tags`         JSON         NOT NULL,
    `duration_sec` DOUBLE       NOT NULL DEFAULT 0,
    `status`       VARCHAR(32)  NOT NULL,
    `error_msg`    TEXT         NOT NULL,
    `playlist_url` TEXT         NOT NULL,
    `created_at`   DATETIME(6)  NOT NULL,
    `updated_at`   DATETIME(6)  NOT NULL,
    PRIMARY KEY (`id`),
    INDEX `videos_created_at_idx` (`created_at`),
    INDEX `videos_status_idx`     (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
