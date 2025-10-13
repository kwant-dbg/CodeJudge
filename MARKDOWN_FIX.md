# Markdown Formatting Fix

## Issue
The problem descriptions were showing raw markdown syntax (like `###` for headings) instead of rendering them as proper HTML elements.

## Root Cause
The markdown processing in `problem.html` was only handling bold (`**text**`) and italic (`*text*`) formatting, but not:
- Headings (`### Title`)
- Lists (`- item`)
- Code blocks (`` `code` ``)

## Solution
Updated the markdown processing in `problem.html` to include:

### 1. Heading Conversion
```javascript
// Convert headings (### h3, ## h2, # h1)
html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>');
html = html.replace(/^# (.+)$/gm, '<h3>$1</h3>');
```

### 2. List Conversion
```javascript
// Convert unordered lists (lines starting with - or *)
html = html.replace(/^[\-\*] (.+)$/gm, '<li>$1</li>');
// Wrap consecutive list items in <ul>
html = html.replace(/(<li>.*<\/li>\n?)+/g, (match) => '<ul>' + match + '</ul>');
```

### 3. Inline Code
```javascript
// Convert backticks to code
html = html.replace(/`([^`]+?)`/g, '<code>$1</code>');
```

### 4. Paragraph Cleanup
```javascript
// Clean up empty paragraphs and misplaced tags
html = html.replace(/<p>\s*<h3>/g, '<h3>');
html = html.replace(/<\/h3>\s*<\/p>/g, '</h3>');
html = html.replace(/<p>\s*<ul>/g, '<ul>');
html = html.replace(/<\/ul>\s*<\/p>/g, '</ul>');
```

## Result
Now problem descriptions properly display:
- ✅ **Headings** - `### The Quadratic Formula` renders as an `<h3>` element
- ✅ **Lists** - `- Item` renders as proper `<ul><li>` elements
- ✅ **Inline code** - `` `code` `` renders with code styling
- ✅ **Bold/Italic** - `**bold**` and `*italic*` work correctly
- ✅ **LaTeX** - Math equations like `$x^2$` and `$$\frac{a}{b}$$` still render properly (protected during markdown processing)

## Files Modified
- `d:\dev\codejudge\monolith\static\problem.html` - Enhanced markdown processing

## Testing
Refresh the problem pages to see the changes:
- Problem 1: http://localhost:8080/problem.html?id=2
- Problem 2: http://localhost:8080/problem.html?id=3

Both problems now display with proper formatting including:
- Heading sections for "The Quadratic Formula", "Discriminant", "Task", "Constraints"
- Bulleted lists for conditions and constraints
- Proper LaTeX math rendering
- Clean paragraph breaks
