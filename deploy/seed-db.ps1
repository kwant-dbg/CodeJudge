# Problem 1: Sum of Series
Write-Host "Creating Problem 1: Sum of Series..." -ForegroundColor Cyan

$problem1 = @{
    title = "Sum of Series"
    description = @"
Write a program that calculates the sum of the first `$`n`$` natural numbers.

The formula is: `$$\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$$`

**Input:** A single integer `$`n`$` (`$`1 \leq n \leq 1000`$`)

**Output:** The sum of first `$`n`$` natural numbers

**Example:**
- Input: `$`5`$`
- Output: `$`15`$` (because `$`1+2+3+4+5=15`$`)
"@
    difficulty = "easy"
    test_cases = @(
        @{input = "5"; output = "15"},
        @{input = "1"; output = "1"},
        @{input = "10"; output = "55"},
        @{input = "100"; output = "5050"}
    )
} | ConvertTo-Json -Depth 10

$response1 = Invoke-RestMethod -Uri "http://localhost:8080/api/problems/" `
    -Method POST `
    -ContentType "application/json" `
    -Headers @{Authorization = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjAxNzcwNzksInVzZXJfaWQiOjZ9.TQxPLZ47d5-1t43_7eEJy9oxaZT8uR0uN3f4TaQZ6sQ"} `
    -Body $problem1

Write-Host "Problem 1 created with ID: $($response1.id)" -ForegroundColor Green

# Problem 2: Quadratic Equation
Write-Host "`nCreating Problem 2: Quadratic Equation..." -ForegroundColor Cyan

$problem2 = @{
    title = "Quadratic Equation Solver"
    description = @"
Given coefficients `$`a`$`, `$`b`$`, and `$`c`$` of a quadratic equation `$`ax^2 + bx + c = 0`$`, determine if the equation has real roots.

The discriminant is: `$$\Delta = b^2 - 4ac$$`

- If `$`\Delta > 0`$`: Two distinct real roots exist
- If `$`\Delta = 0`$`: One real root exists (repeated)
- If `$`\Delta < 0`$`: No real roots

**Input:** Three integers `$`a`$`, `$`b`$`, `$`c`$` separated by spaces (`$`a \neq 0`$`, `$`-100 \leq a,b,c \leq 100`$`)

**Output:** 
- Print "TWO" if two distinct real roots
- Print "ONE" if one real root
- Print "NONE" if no real roots

**Example:**
- Input: ``1 -3 2`` → Output: ``TWO`` (because `$`\Delta = 9 - 8 = 1 > 0`$`)
- Input: ``1 -2 1`` → Output: ``ONE`` (because `$`\Delta = 4 - 4 = 0`$`)
- Input: ``1 0 1`` → Output: ``NONE`` (because `$`\Delta = 0 - 4 = -4 < 0`$`)
"@
    difficulty = "medium"
    test_cases = @(
        @{input = "1 -3 2"; output = "TWO"},
        @{input = "1 -2 1"; output = "ONE"},
        @{input = "1 0 1"; output = "NONE"},
        @{input = "2 -8 6"; output = "TWO"},
        @{input = "1 2 5"; output = "NONE"}
    )
} | ConvertTo-Json -Depth 10

$response2 = Invoke-RestMethod -Uri "http://localhost:8080/api/problems/" `
    -Method POST `
    -ContentType "application/json" `
    -Headers @{Authorization = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjAxNzcwNzksInVzZXJfaWQiOjZ9.TQxPLZ47d5-1t43_7eEJy9oxaZT8uR0uN3f4TaQZ6sQ"} `
    -Body $problem2

Write-Host "Problem 2 created with ID: $($response2.id)" -ForegroundColor Green

Write-Host "`nBoth problems created successfully!" -ForegroundColor Yellow
