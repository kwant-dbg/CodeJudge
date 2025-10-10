#!/bin/bash

# Problem 1: Sum of Series
echo "Creating Problem 1: Sum of Series..."
curl -X POST http://localhost:8080/api/problems/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjAxNzcwNzksInVzZXJfaWQiOjZ9.TQxPLZ47d5-1t43_7eEJy9oxaZT8uR0uN3f4TaQZ6sQ" \
  -d '{
    "title": "Sum of Series",
    "description": "Write a program that calculates the sum of the first $n$ natural numbers.\n\nThe formula is: $$\\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}$$\n\n**Input:** A single integer $n$ ($1 \\leq n \\leq 1000$)\n\n**Output:** The sum of first $n$ natural numbers\n\n**Example:**\n- Input: $5$\n- Output: $15$ (because $1+2+3+4+5=15$)",
    "difficulty": "easy",
    "test_cases": [
      {"input": "5", "output": "15"},
      {"input": "1", "output": "1"},
      {"input": "10", "output": "55"},
      {"input": "100", "output": "5050"}
    ]
  }'

echo -e "\n\n"

# Problem 2: Quadratic Equation
echo "Creating Problem 2: Quadratic Equation..."
curl -X POST http://localhost:8080/api/problems/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjAxNzcwNzksInVzZXJfaWQiOjZ9.TQxPLZ47d5-1t43_7eEJy9oxaZT8uR0uN3f4TaQZ6sQ" \
  -d '{
    "title": "Quadratic Equation Solver",
    "description": "Given coefficients $a$, $b$, and $c$ of a quadratic equation $ax^2 + bx + c = 0$, determine if the equation has real roots.\n\nThe discriminant is: $$\\Delta = b^2 - 4ac$$\n\n- If $\\Delta > 0$: Two distinct real roots exist\n- If $\\Delta = 0$: One real root exists (repeated)\n- If $\\Delta < 0$: No real roots\n\n**Input:** Three integers $a$, $b$, $c$ separated by spaces ($a \\neq 0$, $-100 \\leq a,b,c \\leq 100$)\n\n**Output:** \n- Print \"TWO\" if two distinct real roots\n- Print \"ONE\" if one real root\n- Print \"NONE\" if no real roots\n\n**Example:**\n- Input: `1 -3 2` → Output: `TWO` (because $\\Delta = 9 - 8 = 1 > 0$)\n- Input: `1 -2 1` → Output: `ONE` (because $\\Delta = 4 - 4 = 0$)\n- Input: `1 0 1` → Output: `NONE` (because $\\Delta = 0 - 4 = -4 < 0$)",
    "difficulty": "medium",
    "test_cases": [
      {"input": "1 -3 2", "output": "TWO"},
      {"input": "1 -2 1", "output": "ONE"},
      {"input": "1 0 1", "output": "NONE"},
      {"input": "2 -8 6", "output": "TWO"},
      {"input": "1 2 5", "output": "NONE"}
    ]
  }'

echo -e "\n\nProblems created successfully!"
