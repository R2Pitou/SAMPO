# Sentinel's Journal - CRITICAL SECURITY LEARNINGS

## 2025-05-15 - Path Traversal Vulnerability via User-Controlled Object IDs
**Vulnerability:** The Gateway REST API allowed unauthenticated users to POST arbitrary objects with user-defined IDs (e.g., `../../../../etc/passwd`). These IDs were subsequently joined using `filepath.Join` to construct destination paths for storage replication by the Boatman service. Because `filepath.Join` cleans the resulting path, any directory traversal components allowed the final path to escape the intended Storage Provider root directory, leading to Arbitrary File Write (Path Traversal).

**Learning:** Trusting input identifiers (such as virtual Object IDs) as literal relative filesystem paths without validating and sanitizing them opens up critical path traversal risks. Even if the service performing the file operations is decoupled (via event bus) from the API receiving the input, the malicious input propagates transitively.

**Prevention:**
1. Validate object/file identifiers at the API boundary to ensure they do not contain absolute prefix components, Windows drive letters, or directory traversal elements (e.g., `../`, `..`).
2. Use strict base-directory restriction/sandboxing in file transfer and replication engines (like Boatman) by checking if the resolved path starts with the base directory path (defense-in-depth).
