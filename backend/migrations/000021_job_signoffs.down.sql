-- Reverts 000021_job_signoffs. The stored signatures and reports are not removed.
DROP TABLE IF EXISTS job_signoffs;
