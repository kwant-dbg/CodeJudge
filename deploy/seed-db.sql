-- Problem 1: Sum of Series
WITH new_problem AS (
    INSERT INTO problems (title, description, difficulty) VALUES (
        'Sum of Series',
        'Write a program that calculates the sum of the first $n$ natural numbers. The formula is: $$\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$$

---INPUT---
A single integer $n$ ($1 \leq n \leq 1000$)

---OUTPUT---
The sum of first $n$ natural numbers

---EXAMPLES---
###EXAMPLE1###
INPUT: 5
OUTPUT: 15
NOTE: Because $1+2+3+4+5=15$

###EXAMPLE2###
INPUT: 10
OUTPUT: 55

###EXAMPLE3###
INPUT: 1
OUTPUT: 1',
        'easy'
    ) RETURNING id
)
INSERT INTO test_cases (problem_id, input, output) 
SELECT id, input, output FROM new_problem, (VALUES
    ('5', '15'),
    ('1', '1'),
    ('10', '55'),
    ('100', '5050')
) AS t(input, output);

-- Problem 2: Quadratic Equation Solver
WITH new_problem AS (
    INSERT INTO problems (title, description, difficulty) VALUES (
        'Quadratic Equation Solver',
        'Given coefficients $a$, $b$, and $c$ of a quadratic equation $ax^2 + bx + c = 0$, determine if the equation has real roots.

The discriminant is: $$\Delta = b^2 - 4ac$$

- If $\Delta > 0$: Two distinct real roots exist
- If $\Delta = 0$: One real root exists (repeated)  
- If $\Delta < 0$: No real roots

---INPUT---
Three integers $a$, $b$, $c$ separated by spaces ($a \neq 0$, $-100 \leq a,b,c \leq 100$)

---OUTPUT---
- Print "TWO" if two distinct real roots
- Print "ONE" if one real root  
- Print "NONE" if no real roots

---EXAMPLES---
###EXAMPLE1###
INPUT: 1 -3 2
OUTPUT: TWO
NOTE: Because $\Delta = 9 - 8 = 1 > 0$

###EXAMPLE2###
INPUT: 1 -2 1
OUTPUT: ONE
NOTE: Because $\Delta = 4 - 4 = 0$

###EXAMPLE3###
INPUT: 1 0 1
OUTPUT: NONE
NOTE: Because $\Delta = 0 - 4 = -4 < 0$',
        'medium'
    ) RETURNING id
)
INSERT INTO test_cases (problem_id, input, output) 
SELECT id, input, output FROM new_problem, (VALUES
    ('1 -3 2', 'TWO'),
    ('1 -2 1', 'ONE'),
    ('1 0 1', 'NONE'),
    ('2 -8 6', 'TWO'),
    ('1 2 5', 'NONE')
) AS t(input, output);
