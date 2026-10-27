# Sentinel Learnings

- When retrieving files via dynamic path segments, such as reading `custom` maps using a filename passed via user input in `backend/dev_routes.go`, always sanitize the parameter using `filepath.Base(filepath.Clean(...))` before passing it to file loading or deletion operations.
