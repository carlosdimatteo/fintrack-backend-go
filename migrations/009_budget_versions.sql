-- Dated budget versions, year-to-date evaluation and category rollups.
-- amount stays the monthly USD budget. A category's budget for a month is its latest row with
-- effective_from <= that month.

-- 1. Version start date. Rows created before 2026 apply from 2025-01-01, 2026 rows from 2026-01-01.
ALTER TABLE public.budgets ADD COLUMN effective_from date;
UPDATE public.budgets
SET effective_from = CASE WHEN created_at < '2026-01-01' THEN DATE '2025-01-01' ELSE DATE '2026-01-01' END;
ALTER TABLE public.budgets ALTER COLUMN effective_from SET NOT NULL;

-- 2. true = judge year-to-date (amount is still monthly)
ALTER TABLE public.budgets ADD COLUMN evaluate_ytd boolean NOT NULL DEFAULT false;

-- 3. When set, this category's spend counts against the rollup category's budget and the category
--    itself is not evaluated.
ALTER TABLE public.budgets ADD COLUMN rollup_category_id bigint REFERENCES public.categories(id);

-- 4.
ALTER TABLE public.budgets ADD COLUMN notes text;

-- 5. One row per category per version
ALTER TABLE public.budgets DROP CONSTRAINT budgets_category_id_key;
ALTER TABLE public.budgets ADD CONSTRAINT budgets_category_id_effective_from_key UNIQUE (category_id, effective_from);

-- 6. id is already an identity column (budgets_id_seq ahead of max(id)): inserts without an id work.

-- 7. Latest version in effect today, per category
CREATE VIEW public.budgets_current WITH (security_invoker = true) AS
SELECT DISTINCT ON (category_id)
    id, created_at, category_id, amount, effective_from, evaluate_ytd, rollup_category_id, notes
FROM public.budgets
WHERE effective_from <= current_date
ORDER BY category_id, effective_from DESC;

-- 8. Version in effect for the month containing m, per category
CREATE FUNCTION public.budget_for_month(m date)
RETURNS TABLE (
    category_id bigint,
    amount double precision,
    evaluate_ytd boolean,
    rollup_category_id bigint,
    notes text,
    effective_from date
)
LANGUAGE sql
STABLE
SET search_path = ''
AS $$
    SELECT DISTINCT ON (b.category_id)
        b.category_id, b.amount, b.evaluate_ytd, b.rollup_category_id, b.notes, b.effective_from
    FROM public.budgets b
    WHERE b.effective_from <= date_trunc('month', m)::date
    ORDER BY b.category_id, b.effective_from DESC
$$;

-- Existing reader: this view joined budgets directly, so a second version would duplicate rows.
-- Same columns as before; now reads this month's version.
CREATE OR REPLACE VIEW public.budget_by_category_current_month AS
WITH lent_by_expense AS (
    SELECT expense_id, SUM(amount) AS lent
    FROM public.debts
    WHERE outbound AND expense_id IS NOT NULL
    GROUP BY expense_id
),
monthlyexpenses AS (
    SELECT e.category_id, e.expense, COALESCE(l.lent, 0) AS lent
    FROM public.expenses e
    LEFT JOIN lent_by_expense l ON l.expense_id = e.id
    WHERE e.created_at >= date_trunc('MONTH', CURRENT_DATE::timestamp with time zone)
      AND e.created_at < date_trunc('MONTH', CURRENT_DATE::timestamp with time zone) + '1 mon'::interval
)
SELECT b.amount,
    c.name AS category_name,
    b.category_id,
    COALESCE(SUM(e.expense - e.lent), 0::double precision) AS spent,
    COALESCE(SUM(e.lent), 0::double precision) AS lent
FROM public.budget_for_month(CURRENT_DATE) b
JOIN public.categories c ON b.category_id = c.id
LEFT JOIN monthlyexpenses e ON b.category_id = e.category_id
GROUP BY b.amount, c.name, b.category_id
ORDER BY b.category_id;
