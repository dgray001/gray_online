import json
import re
import sys

with open('eslint.json', 'r') as f:
    data = json.load(f)

for file in data:
    path = file['filePath']
    messages = [m for m in file['messages'] if m['ruleId'] == '@typescript-eslint/explicit-function-return-type']
    if not messages:
        continue
        
    with open(path, 'r') as f:
        content = f.read()
        
    content = re.sub(r'\b(function\s+[a-zA-Z0-9_]+\s*\([^)]*\))\s*\{', r'\1: void {', content)
    content = re.sub(r'^\s*([a-zA-Z0-9_]+\s*\([^)]*\))\s*\{', r'\1: void {', content, flags=re.MULTILINE)
    content = re.sub(r'(\([^)]*\))\s*=>\s*\{', r'\1: void => {', content)
    content = re.sub(r'\bconstructor(\s*\([^)]*\)): void\s*\{', r'constructor\1 {', content)
    for kw in ['if', 'for', 'while', 'switch', 'catch']:
        content = re.sub(r'\b' + kw + r'(\s*\([^)]*\)): void\s*\{', kw + r'\1 {', content)
        
    with open(path, 'w') as f:
        f.write(content)
