import os
import time
import sys
import jellyfish
from itertools import combinations

def benchmark_naive_plagiarism(directory_path):
    """
    Performs a naive, O(N^2) plagiarism check on all files in a directory.
    """
    if not os.path.isdir(directory_path):
        print(f"Error: Directory not found at '{directory_path}'")
        sys.exit(1)

    filepaths = [os.path.join(directory_path, f) for f in os.listdir(directory_path) if f.endswith(('.cpp', '.cc', '.cxx'))]
    
    if len(filepaths) < 2:
        print("Need at least two source files to compare.")
        return

    print(f"Found {len(filepaths)} C++ files. Starting naive comparison...")
    
    contents = {}
    for path in filepaths:
        with open(path, 'r', encoding='utf-8', errors='ignore') as f:
            contents[path] = f.read()

    start_time = time.time()
    
    # O(N^2) comparison
    num_comparisons = 0
    for (path1, content1), (path2, content2) in combinations(contents.items(), 2):
        _ = jellyfish.jaro_winkler_similarity(content1, content2)
        num_comparisons += 1

    end_time = time.time()
    print(f"\nFinished!")
    print(f"Total comparisons: {num_comparisons}")
    print(f"Execution time: {end_time - start_time:.4f} seconds")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python benchmark_plagiarism_naive.py <directory_with_cpp_files>")
        sys.exit(1)
    benchmark_naive_plagiarism(sys.argv[1])
