-- name: ListSalaries :many
SELECT *
FROM salary
ORDER BY name, id;

-- name: GetSalary :one
SELECT *
FROM salary
WHERE id = ?;

-- name: UpsertSalary :one
INSERT INTO salary (
    id,
    name,
    to_account_id,
    pension_account_id,
    priority,
    recurrence,
    budget_category_id,
    enabled,
    kommun,
    forsamling,
    church_member,
    is_gross,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
  to_account_id = EXCLUDED.to_account_id,
  pension_account_id = EXCLUDED.pension_account_id,
  priority = EXCLUDED.priority,
  recurrence = EXCLUDED.recurrence,
  budget_category_id = EXCLUDED.budget_category_id,
  enabled = EXCLUDED.enabled,
  kommun = EXCLUDED.kommun,
  forsamling = EXCLUDED.forsamling,
  church_member = EXCLUDED.church_member,
  is_gross = EXCLUDED.is_gross,
  updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteSalary :exec
DELETE FROM salary
WHERE id = ?;

-- name: ListSalaryAmounts :many
SELECT *
FROM salary_amount
WHERE salary_id = ?
ORDER BY start_date, id;

-- name: GetSalaryAmount :one
SELECT *
FROM salary_amount
WHERE id = ?;

-- name: UpsertSalaryAmount :one
INSERT INTO salary_amount (
    id,
    salary_id,
    amount,
    start_date,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET salary_id = EXCLUDED.salary_id,
  amount = EXCLUDED.amount,
  start_date = EXCLUDED.start_date,
  updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteSalaryAmount :exec
DELETE FROM salary_amount
WHERE id = ?;

-- name: ListAllSalaryAmounts :many
SELECT *
FROM salary_amount
ORDER BY salary_id, start_date, id;
