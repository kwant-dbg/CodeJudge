# Sample Solutions for Testing

## Problem 1: Sum of Two Numbers

### Python Solution
```python
# Read input
a, b = map(int, input().split())

# Calculate sum
result = a + b

# Print output
print(result)
```

### C++ Solution
```cpp
#include <iostream>
using namespace std;

int main() {
    int a, b;
    cin >> a >> b;
    cout << (a + b) << endl;
    return 0;
}
```

### Java Solution
```java
import java.util.Scanner;

public class Main {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int a = sc.nextInt();
        int b = sc.nextInt();
        System.out.println(a + b);
        sc.close();
    }
}
```

---

## Problem 2: Quadratic Equation Roots

### Python Solution
```python
import math

# Read input
a, b, c = map(int, input().split())

# Calculate discriminant
discriminant = b * b - 4 * a * c

if discriminant < 0:
    # Complex roots
    print("COMPLEX")
elif discriminant == 0:
    # One repeated root
    root = -b / (2 * a)
    print(f"{root:.2f}")
else:
    # Two distinct real roots
    sqrt_discriminant = math.sqrt(discriminant)
    root1 = (-b - sqrt_discriminant) / (2 * a)
    root2 = (-b + sqrt_discriminant) / (2 * a)
    
    # Output in ascending order
    if root1 > root2:
        root1, root2 = root2, root1
    
    print(f"{root1:.2f}")
    print(f"{root2:.2f}")
```

### C++ Solution
```cpp
#include <iostream>
#include <cmath>
#include <iomanip>
using namespace std;

int main() {
    int a, b, c;
    cin >> a >> b >> c;
    
    // Calculate discriminant
    int discriminant = b * b - 4 * a * c;
    
    if (discriminant < 0) {
        // Complex roots
        cout << "COMPLEX" << endl;
    } else if (discriminant == 0) {
        // One repeated root
        double root = -b / (2.0 * a);
        cout << fixed << setprecision(2) << root << endl;
    } else {
        // Two distinct real roots
        double sqrt_discriminant = sqrt(discriminant);
        double root1 = (-b - sqrt_discriminant) / (2.0 * a);
        double root2 = (-b + sqrt_discriminant) / (2.0 * a);
        
        // Output in ascending order
        if (root1 > root2) {
            swap(root1, root2);
        }
        
        cout << fixed << setprecision(2) << root1 << endl;
        cout << fixed << setprecision(2) << root2 << endl;
    }
    
    return 0;
}
```

### Java Solution
```java
import java.util.Scanner;

public class Main {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int a = sc.nextInt();
        int b = sc.nextInt();
        int c = sc.nextInt();
        
        // Calculate discriminant
        int discriminant = b * b - 4 * a * c;
        
        if (discriminant < 0) {
            // Complex roots
            System.out.println("COMPLEX");
        } else if (discriminant == 0) {
            // One repeated root
            double root = -b / (2.0 * a);
            System.out.printf("%.2f%n", root);
        } else {
            // Two distinct real roots
            double sqrtDiscriminant = Math.sqrt(discriminant);
            double root1 = (-b - sqrtDiscriminant) / (2.0 * a);
            double root2 = (-b + sqrtDiscriminant) / (2.0 * a);
            
            // Output in ascending order
            if (root1 > root2) {
                double temp = root1;
                root1 = root2;
                root2 = temp;
            }
            
            System.out.printf("%.2f%n", root1);
            System.out.printf("%.2f%n", root2);
        }
        
        sc.close();
    }
}
```

---

## Testing Instructions

1. Navigate to the problem page
2. Copy one of the solutions above for your preferred language
3. Paste it into the code editor
4. Click "Submit Solution"
5. The judge will run your code against all test cases (both sample and hidden)
6. You'll see the verdict: ACCEPTED or details about which test cases failed

## Expected Results

Both solutions should produce:
- **Verdict: ACCEPTED** ✅
- All 6 test cases passing (3 sample + 3 hidden)
