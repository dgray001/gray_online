# Sentinel Learnings

## Insecure Path Joining (Directory Traversal)
- **Vulnerability**: Constructing file paths by joining a base directory with an unbounded or unsanitized set of path parts using `filepath.Join(append([]string{baseDir}, parts...)...)` is vulnerable to directory traversal if the `parts` slice contains sequences like `../`.
- **Modern Go Solution**: As of Go 1.20, `filepath.IsLocal` can be used to securely determine if a path remains local to a directory (i.e. it does not try to traverse up using `..` to escape the root directory). A safe pattern is to first join the variable parts, check if the resulting path is locally safe with `filepath.IsLocal`, and only then join it with the base directory.

```go
func secureJoin(baseDir string, parts ...string) (string, error) {
	p := filepath.Join(parts...)
	if p != "" && !filepath.IsLocal(p) {
		return "", fmt.Errorf("insecure path: %q", p)
	}
	return filepath.Join(baseDir, p), nil
}
```
