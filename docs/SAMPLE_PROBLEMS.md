# Sample Problems Created

Two sample problems have been successfully created with proper LaTeX formatting, test cases, and sample inputs/outputs.

## Problem 1: Sum of Two Numbers
**Difficulty:** Easy  
**URL:** http://localhost:8080/problem.html?id=2

### Description
A simple arithmetic problem that demonstrates:
- Basic LaTeX math notation using `$a$`, `$b$`, `$c$`
- Display math equations using `$$c = a + b$$`
- Mathematical sets notation: `$a, b \in \mathbb{Z}$`
- Constraint notation: `$-10^9 \le a, b \le 10^9$`

### Test Cases
- **3 Sample test cases** (visible to users):
  - Input: `2 3` → Output: `5`
  - Input: `10 20` → Output: `30`
  - Input: `-5 8` → Output: `3`
  
- **3 Hidden test cases** (for validation only):
  - Input: `0 0` → Output: `0`
  - Input: `1000000000 -1000000000` → Output: `0`
  - Input: `999999999 1` → Output: `1000000000`

---

## Problem 2: Quadratic Equation Roots
**Difficulty:** Medium  
**URL:** http://localhost:8080/problem.html?id=3

### Description
A more complex problem demonstrating advanced LaTeX features:
- Quadratic equation formula: `$$ax^2 + bx + c = 0$$`
- The quadratic formula with fractions and square roots: `$$x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$$`
- Greek letters: `$\Delta$` (delta)
- Inequality symbols: `$a \neq 0$`, `$\Delta > 0$`, `$\Delta = 0$`, `$\Delta < 0$`
- Complex mathematical expressions
- Range constraints: `$1 \le a \le 100$`, `$-1000 \le b, c \le 1000$`

### Test Cases
- **3 Sample test cases** (visible to users):
  1. Input: `1 -3 2` → Output: `1.00\n2.00` (two distinct roots)
  2. Input: `1 -2 1` → Output: `1.00` (one repeated root)
  3. Input: `1 0 1` → Output: `COMPLEX` (complex roots)
  
- **3 Hidden test cases** (for validation only):
  1. Input: `1 -5 6` → Output: `2.00\n3.00`
  2. Input: `2 -8 8` → Output: `2.00`
  3. Input: `1 2 5` → Output: `COMPLEX`

---

## Features Demonstrated

### LaTeX Math Rendering
Both problems showcase different LaTeX capabilities:
- **Inline math**: `$variable$` for variables in text
- **Display math**: `$$equation$$` for centered equations
- **Fractions**: `\frac{numerator}{denominator}`
- **Square roots**: `\sqrt{expression}`
- **Subscripts/Superscripts**: `x^2`, `10^9`
- **Greek letters**: `\Delta`, `\mathbb{Z}`
- **Comparison operators**: `\le`, `\ge`, `\neq`
- **Plus-minus**: `\pm`

### Test Cases with Sample Flag
Each problem includes:
- **Sample test cases** (`sample: true`): Visible to users for understanding the problem
- **Hidden test cases** (`sample: false`): Used only for validation during submission

### Clean Display
The problems now properly display:
1. Problem title and metadata (difficulty, ID)
2. Problem description with rendered LaTeX
3. Input/Output format sections
4. **Sample Tests section** showing the sample inputs and outputs in a clean, two-column format
5. Submission panel for code submission

---

## How to View

1. Make sure the application is running: `docker-compose up`
2. Navigate to:
   - Problem 1: http://localhost:8080/problem.html?id=2
   - Problem 2: http://localhost:8080/problem.html?id=3

## How to Test

1. Login with:
   - Username: `sampleuser`
   - Password: `password123`

2. Select a programming language (C++, Python, or Java)

3. Write your solution

4. Submit and see the results!

---

## Script Used

The problems were created using the PowerShell script `create_sample_problems.ps1`, which:
1. Registers/logs in a user
2. Gets an authentication token
3. Creates both problems via the REST API
4. Displays the URLs to view them

You can run it again with: `./create_sample_problems.ps1`
