# Script to create two sample problems with LaTeX formatting

$baseUrl = "http://localhost:8080"

# First, let's register or login to get a token
Write-Host "Registering/Logging in..." -ForegroundColor Cyan

# Try to register
$registerData = @{
    username = "sampleuser"
    password = "password123"
    email = "sample@example.com"
} | ConvertTo-Json

try {
    $registerResponse = Invoke-RestMethod -Uri "$baseUrl/api/auth/register" -Method POST -Body $registerData -ContentType "application/json"
    Write-Host "Registration successful!" -ForegroundColor Green
} catch {
    Write-Host "User might already exist, trying to login..." -ForegroundColor Yellow
}

# Login to get token
$loginData = @{
    username = "sampleuser"
    password = "password123"
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "$baseUrl/api/auth/login" -Method POST -Body $loginData -ContentType "application/json"
$token = $loginResponse.token

Write-Host "Login successful! Token obtained." -ForegroundColor Green
Write-Host ""

# Problem 1: Sum of Two Numbers with LaTeX
Write-Host "Creating Problem 1: Sum of Two Numbers..." -ForegroundColor Cyan

$problem1 = @{
    title = "Sum of Two Numbers"
    description = "Given two integers " + '$a$' + " and " + '$b$' + ", compute their sum " + '$c = a + b$' + ".`n`nThis is a simple arithmetic problem to test basic input/output.`n`n**Mathematical Notation:** The sum can be expressed as " + '$$c = a + b$$' + " where " + '$a, b \in \mathbb{Z}$' + " (integers).`n`n### Constraints`n- " + '$-10^9 \le a, b \le 10^9$' + "`n- The answer will fit in a 32-bit signed integer"
    difficulty = "easy"
    input_format = "Two space-separated integers a and b"
    output_format = "A single integer representing a + b"
    test_cases = @(
        @{ input = "2 3"; output = "5"; sample = $true }
        @{ input = "10 20"; output = "30"; sample = $true }
        @{ input = "-5 8"; output = "3"; sample = $true }
        @{ input = "0 0"; output = "0"; sample = $false }
        @{ input = "1000000000 -1000000000"; output = "0"; sample = $false }
        @{ input = "999999999 1"; output = "1000000000"; sample = $false }
    )
} | ConvertTo-Json -Depth 10

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

try {
    $response1 = Invoke-RestMethod -Uri "$baseUrl/api/problems" -Method POST -Body $problem1 -Headers $headers
    Write-Host "Problem 1 created successfully! ID: $($response1.id)" -ForegroundColor Green
    Write-Host "View at: http://localhost:8080/problem.html?id=$($response1.id)" -ForegroundColor Yellow
    Write-Host ""
} catch {
    Write-Host "Error creating Problem 1: $_" -ForegroundColor Red
    Write-Host "Response: $($_.Exception.Response)" -ForegroundColor Red
}

# Problem 2: Quadratic Equation Solver with Advanced LaTeX
Write-Host "Creating Problem 2: Quadratic Equation Roots..." -ForegroundColor Cyan

$problem2 = @{
    title = "Quadratic Equation Roots"
    description = "Given a quadratic equation in the standard form " + '$$ax^2 + bx + c = 0$$' + " where " + '$a \neq 0$' + ", determine the nature of its roots and compute them.`n`n### The Quadratic Formula`n`nThe roots of a quadratic equation can be found using the quadratic formula:`n`n" + '$$x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$$' + "`n`n### Discriminant`n`nThe discriminant " + '$\Delta$' + " determines the nature of the roots:`n`n" + '$$\Delta = b^2 - 4ac$$' + "`n`n- If " + '$\Delta > 0$' + ": Two distinct real roots`n- If " + '$\Delta = 0$' + ": One repeated real root`n- If " + '$\Delta < 0$' + ": Two complex conjugate roots`n`n### Task`n`nGiven coefficients " + '$a$' + ", " + '$b$' + ", and " + '$c$' + ", output the roots rounded to 2 decimal places.`n`n- If there are two distinct real roots, output them in ascending order on separate lines`n- If there is one repeated root, output it once`n- If there are complex roots, output COMPLEX`n`n### Constraints`n- " + '$1 \le a \le 100$' + "`n- " + '$-1000 \le b, c \le 1000$' + "`n- All coefficients are integers"
    difficulty = "medium"
    input_format = "Three space-separated integers a, b, and c representing the coefficients of the quadratic equation"
    output_format = "The roots of the equation according to the rules described above, rounded to 2 decimal places"
    test_cases = @(
        @{ 
            input = "1 -3 2"
            output = "1.00`n2.00"
            sample = $true 
        }
        @{ 
            input = "1 -2 1"
            output = "1.00"
            sample = $true 
        }
        @{ 
            input = "1 0 1"
            output = "COMPLEX"
            sample = $true 
        }
        @{ 
            input = "1 -5 6"
            output = "2.00`n3.00"
            sample = $false 
        }
        @{ 
            input = "2 -8 8"
            output = "2.00"
            sample = $false 
        }
        @{ 
            input = "1 2 5"
            output = "COMPLEX"
            sample = $false 
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response2 = Invoke-RestMethod -Uri "$baseUrl/api/problems" -Method POST -Body $problem2 -Headers $headers
    Write-Host "Problem 2 created successfully! ID: $($response2.id)" -ForegroundColor Green
    Write-Host "View at: http://localhost:8080/problem.html?id=$($response2.id)" -ForegroundColor Yellow
    Write-Host ""
} catch {
    Write-Host "Error creating Problem 2: $_" -ForegroundColor Red
    Write-Host "Response: $($_.Exception.Response)" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Sample Problems Created Successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "You can now view the problems with sample test cases at:" -ForegroundColor Yellow
Write-Host "- Problem 1: http://localhost:8080/problem.html?id=$($response1.id)" -ForegroundColor White
Write-Host "- Problem 2: http://localhost:8080/problem.html?id=$($response2.id)" -ForegroundColor White
Write-Host ""
