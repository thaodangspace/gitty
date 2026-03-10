# Server-Side Diff Tokenization — Architecture & Implementation Plan

## 1. Problem Statement

The mobile Git client needs to display syntax-highlighted diffs. Doing syntax highlighting on the React Native client is problematic:

- **`react-native-syntax-highlighter`** bundles a full JS tokenizer + theme engine, adding ~200KB+ to the bundle and running expensive regex on the JS thread.
- Large diffs (500+ lines) cause visible jank on mid-range Android devices because the JS thread is blocked during tokenization.
- The RN client would need to parse raw unified diff text, strip `+`/`-` prefixes, run the highlighter, then reassemble — duplicating logic that the server already partially does.

**Solution**: Move all syntax highlighting to the Go backend. The server parses diffs, tokenizes code via Chroma, and sends pre-colored token arrays. The RN client becomes a dumb renderer: `token.text` → `<Text style={{color: token.color}}>`.

---

## 2. Pipeline Overview

```
┌──────────────── SERVER (Go) ────────────────┐     ┌──── CLIENT (RN) ────┐
│                                              │     │                     │
│  git diff (unified text)                     │     │                     │
│       │                                      │     │                     │
│       ▼                                      │     │                     │
│  parseDiffContent()                          │     │                     │
│  → split into hunks & lines                  │     │                     │
│  → classify: added / deleted / context       │     │                     │
│  → extract line numbers                      │     │                     │
│  → strip +/- prefixes → pure code            │     │                     │
│       │                                      │     │                     │
│       ▼                                      │     │                     │
│  reconstruct old & new source files          │     │                     │
│  (context+deleted → old, context+added → new)│     │                     │
│       │                                      │     │                     │
│       ▼                                      │     │                     │
│  tokenizeFullSource() via Chroma             │     │                     │
│  → full-file lexing for both sides           │     │                     │
│  → split tokens at \n boundaries             │     │                     │
│  → map back to original diff line indices    │     │                     │
│       │                                      │     │                     │
│       ▼                                      │     │                     │
│  assemble TokenizedDiff JSON                 │     │                     │
│  { filename, hunks: [{ header, lines }] }    │     │                     │
│       │                                      │     │                     │
│       └──────── HTTP JSON ──────────────────────▶  │                     │
│                                              │     │  TanStack Query     │
│                                              │     │  caches response    │
│                                              │     │       │             │
│                                              │     │       ▼             │
│                                              │     │  TokenizedDiffView  │
│                                              │     │  renders tokens as  │
│                                              │     │  <Text> spans       │
│                                              │     │                     │
└──────────────────────────────────────────────┘     └─────────────────────┘
```

---

## 3. Data Model (JSON Contract)

The server and client share an exact type contract. The Go structs serialize to JSON that matches the TypeScript interfaces.

### Token

The atomic unit. One colored text fragment within a line.

```
Token {
  text:  string   // e.g. "const", " ", "handleClick"
  color: string   // hex color, e.g. "#C678DD"
}
```

Design choice: we bake the color directly into the token rather than sending a token-type enum and letting the client resolve colors. This eliminates the need for a theme engine on the client. The tradeoff is slightly larger payloads (~10-15% more bytes), but the simplicity gain on mobile is worth it.

### DiffLine

A single line in the diff, already classified and tokenized.

```
DiffLine {
  type:   "added" | "deleted" | "context"
  tokens: Token[]
  oldNum: int?    // line number in the old file (absent for added lines)
  newNum: int?    // line number in the new file (absent for deleted lines)
}
```

### DiffHunk

A contiguous group of changes with a header (the `@@ -a,b +c,d @@` line).

```
DiffHunk {
  header: string      // e.g. "@@ -14,8 +14,10 @@ function Button()"
  lines:  DiffLine[]
}
```

### TokenizedDiff

The full response for a single file.

```
TokenizedDiff {
  filename:  string
  hunks:     DiffHunk[]
  additions: int
  deletions: int
}
```

### TokenizedCommitDiff

Wraps multiple file diffs for a whole commit.

```
TokenizedCommitDiff {
  hash:    string
  message: string
  author:  { name, email }
  date:    ISO 8601 string
  files:   TokenizedFileDiff[]   // one per changed file
  stats:   { additions, deletions }
}
```

---

## 4. Server Algorithm — Step by Step

### 4.1 Diff Parsing (`parseDiffContent`)

Input: raw unified diff string (from the existing `GetFileDiff` / `GetStagedDiff` / `GetCommitDetails`).

1. **Skip the header lines** — everything before the first `@@` (the `diff --git`, `index`, `---`, `+++` lines). Extract the filename from the `+++ b/...` line.

2. **Parse hunk headers** — each `@@ -old,count +new,count @@` line starts a new hunk and resets the line number counters.

3. **Classify content lines**:
   - `+` prefix → `added`, increment newLine counter
   - `-` prefix → `deleted`, increment oldLine counter
   - ` ` prefix (or no prefix) → `context`, increment both counters
   - `\ No newline at end of file` → skip

4. **Strip the prefix** — store only the code content (without the leading `+`, `-`, or ` `).

Output: `[]rawDiffLine` with type, content, oldNum, newNum.

### 4.2 Source Reconstruction

This is the key insight for accurate multi-line tokenization.

A diff interleaves added, deleted, and context lines. If we tokenize line-by-line, multi-line constructs break — e.g., a template literal that starts on a deleted line and ends on a context line would be mis-tokenized.

Instead, we reconstruct two virtual source files:

- **Old file**: context lines + deleted lines (in order)
- **New file**: context lines + added lines (in order)

We also record which parsed diff line index each reconstructed line came from, so we can map tokens back after tokenization.

```
Diff:                   Old source:         New source:
  context: "import x"   → "import x"        → "import x"
  deleted: "const a"     → "const a"
  added:   "const b"                         → "const b"
  context: "export y"   → "export y"        → "export y"
```

### 4.3 Full-Source Tokenization (`tokenizeFullSource`)

For each reconstructed source file:

1. **Join all lines** with `\n` into a single string.
2. **Run Chroma's lexer** on the full string. Chroma returns a flat stream of `(tokenType, text)` pairs.
3. **Split tokens at newline boundaries** — a single Chroma token can span multiple lines (e.g., a multi-line comment). When we encounter `\n` in a token's text, we advance to the next line's token array.
4. **Map colors** — look up each `tokenType` in our dark theme map. Walk up the token type hierarchy if no exact match (e.g., `LiteralStringDouble` → `LiteralString` → `Literal`).
5. **Merge adjacent same-color tokens** — reduces JSON array size.

Output: `[][]Token` — one token slice per reconstructed source line.

### 4.4 Token Mapping

Build a lookup `map[parsedIndex][]Token` from the reconstructed line arrays:

- For each old source line at index `i` → `tokenMap[oldIndices[i]] = oldTokenized[i]`
- For each new source line at index `i` → `tokenMap[newIndices[i]] = newTokenized[i]`

Context lines appear in both old and new, so both writes produce the same tokens (overwriting is safe).

### 4.5 Assembly

Walk the parsed diff lines in order. For each line, look up its tokens in `tokenMap`, attach the line type and numbers, and group into hunks. Count total additions and deletions.

### 4.6 Lexer Detection

The `lexerForFile` function determines the language:

1. Try Chroma's built-in filename matching (handles shebangs, known filenames like `Makefile`, `Dockerfile`).
2. Fall back to a manual extension map for common types (`.ts`, `.tsx`, `.go`, `.py`, `.rs`, etc.).
3. Final fallback: `plaintext` lexer (returns the whole line as one token).

---

## 5. Color Theme

The server embeds a dark theme map (`darkTheme`) that produces colors matching the app's design system. It's based on One Dark with adjustments to align with the prototype's `T` token palette.

Key color assignments:

| Syntax Element       | Color     | Hex       |
|---------------------|-----------|-----------|
| Keywords            | Purple    | `#C678DD` |
| Strings             | Green     | `#98C379` |
| Numbers             | Orange    | `#D19A66` |
| Functions           | Blue      | `#61AFEF` |
| Types / Classes     | Yellow    | `#E5C07B` |
| Comments            | Gray      | `#5C6370` |
| Operators           | Cyan      | `#56B6C2` |
| Punctuation         | Gray      | `#ABB2BF` |
| Variables / Tags    | Red       | `#E06C75` |
| Plain text          | Light     | `#E6EDF3` |

The fallback color is `#E6EDF3` (the app's `T.text` token), so unrecognized syntax renders as normal text.

To change themes in the future: update the `darkTheme` map on the server and redeploy. No client update needed.

---

## 6. API Endpoints

### `GET /api/diff/file`

Returns the tokenized diff for a single file in the working tree or staging area.

**Query params:**
- `repo` — absolute path to the repository
- `path` — relative file path within the repo
- `staged` — `"true"` for staged diff, omit or `"false"` for working tree

**Response:** `TokenizedDiff` JSON

**Cache guidance:** Client uses `staleTime: 5000` (5 seconds). Working tree diffs change frequently during active development.

### `GET /api/diff/commit`

Returns tokenized diffs for all files changed in a commit.

**Query params:**
- `repo` — absolute path to the repository
- `hash` — commit SHA

**Response:** `TokenizedCommitDiff` JSON

**Cache guidance:** Client uses `staleTime: 60000` (60 seconds). Commits are immutable, so this can be cached aggressively.

---

## 7. RN Client Renderer

### Component Hierarchy

```
CommitDiffList                    (FlatList of files — virtualized)
  └─ TokenizedDiffView            (one per file)
       ├─ DiffFileHeader          (filename, +/- stats, change type badge)
       └─ HunkView[]              (one per hunk, collapsible)
            ├─ Hunk header        (pressable, toggles collapse)
            └─ DiffLineRow[]      (one per line)
                 ├─ Line border   (3px colored left edge)
                 ├─ Old line num  (gutter)
                 ├─ New line num  (gutter)
                 └─ TokenSpan[]   (the actual colored code text)
```

### Rendering Logic

The renderer is intentionally simple — "dumb" by design:

```tsx
// The core of the entire client-side rendering:
<Text style={styles.codeLine}>
  {line.tokens.map((token, i) => (
    <Text key={i} style={{ color: token.color }}>{token.text}</Text>
  ))}
</Text>
```

No parsing. No regex. No syntax grammar. No theme resolution. Just colored text spans.

### Performance Optimizations

1. **`memo()` on every component** — `TokenSpan`, `DiffLineRow`, `HunkView`, `TokenizedDiffView` are all wrapped in `React.memo`. Since token data is immutable from the server, reference equality checks prevent re-renders.

2. **Collapsible hunks** — hunks beyond index 5 start collapsed (`defaultCollapsed`). This avoids rendering 500+ lines of a large diff on mount.

3. **FlatList virtualization** in `CommitDiffList`:
   - `initialNumToRender: 3` — only render 3 file diffs initially
   - `maxToRenderPerBatch: 2` — render 2 more per frame
   - `windowSize: 5` — keep 5 screens worth of content
   - `removeClippedSubviews: true` — native-level view recycling

4. **Token merging on the server** — adjacent tokens with the same color are pre-merged, reducing the number of `<Text>` nodes the client creates. Fewer nodes = less bridge traffic = smoother scrolling.

### Line Backgrounds

Lines get a subtle background color based on type:

- Added: `rgba(63,185,80,0.08)` (faint green)
- Deleted: `rgba(248,81,73,0.08)` (faint red)
- Context: `transparent`

Plus a 3px left border in the saturated color (`T.green`, `T.red`, or `transparent`).

---

## 8. Data Flow: TanStack Query Integration

### Hook: `useFileDiff`

```ts
useFileDiff({ repoPath, filePath, staged: false })
```

- `queryKey`: `['fileDiff', repoPath, filePath, staged]`
- `staleTime`: 5s (re-fetches if user switches back to a file after 5s)
- `gcTime`: 30s (keeps unused diffs in memory for 30s)
- `enabled`: only fires when both repoPath and filePath are set

### Hook: `useCommitDiff`

```ts
useCommitDiff({ repoPath, commitHash })
```

- `queryKey`: `['commitDiff', repoPath, commitHash]`
- `staleTime`: 60s (commits are immutable)
- `gcTime`: 5 min

### Invalidation

When the user performs an action that changes the working tree (commit, stage, pull, checkout), invalidate the relevant queries:

```ts
queryClient.invalidateQueries({ queryKey: ['fileDiff', repoPath] });
```

Commit diffs never need invalidation (immutable data).

---

## 9. Payload Size Estimation

For a typical diff with 50 changed lines:

| Approach | Payload Size |
|----------|-------------|
| Raw unified diff text | ~2 KB |
| Tokenized JSON (this design) | ~8-12 KB |
| Tokenized JSON (gzipped) | ~2-3 KB |

The 4-6x increase in raw size is mitigated by:
- HTTP gzip compression (standard, reduces to ~1.3x of raw diff)
- Elimination of client-side processing time (JS tokenization of 50 lines takes 20-50ms on mobile; this drops to 0)
- The REST polling architecture already sends JSON; adding tokens to the existing response shape adds minimal overhead

For very large diffs (1000+ lines), the tokenized payload reaches ~100-200 KB raw (~20-40 KB gzipped). This is acceptable for a Tailscale LAN connection but worth monitoring. If needed, a future optimization could paginate hunks or lazy-load file diffs within a commit.

---

## 10. Go Dependency

Add Chroma v2 to the Go module:

```bash
go get github.com/alecthomas/chroma/v2
```

Chroma is a pure-Go library (no CGO, no external dependencies). It supports 200+ languages and is the same highlighter used by Hugo, Gitea, and Goldmark.

---

## 11. File Inventory

### Server (Go)

| File | Purpose |
|------|---------|
| `server/internal/git/diff_tokenizer.go` | Core algorithm: diff parsing, Chroma tokenization, theme map, assembly |
| `server/internal/models/tokenized_diff.go` | Shared data models (`Token`, `DiffLine`, `DiffHunk`, etc.) |
| `server/internal/api/diff_handler.go` | HTTP handlers for `/api/diff/file` and `/api/diff/commit` |

### Client (React Native)

| File | Purpose |
|------|---------|
| `app/src/types/diff.ts` | TypeScript interfaces matching Go models |
| `app/src/components/TokenizedDiffView.tsx` | Renderer components (`TokenizedDiffView`, `CommitDiffList`) |
| `app/src/hooks/useDiff.ts` | TanStack Query hooks (`useFileDiff`, `useCommitDiff`) |

---

## 12. Integration Steps

1. **Add Chroma dependency** — `go get github.com/alecthomas/chroma/v2`

2. **Add the Go files** — copy `diff_tokenizer.go`, `tokenized_diff.go`, and `diff_handler.go` into the existing project structure.

3. **Register API routes** — in your router setup (wherever you register HTTP handlers):
   ```go
   diffHandler := api.NewDiffHandler(gitService)
   mux.HandleFunc("/api/diff/file", diffHandler.HandleFileDiff)
   mux.HandleFunc("/api/diff/commit", diffHandler.HandleCommitDiff)
   ```

4. **Add the RN files** — copy `diff.ts`, `TokenizedDiffView.tsx`, and `useDiff.ts` into the app.

5. **Install JetBrains Mono font** — the renderer expects `JetBrainsMono-Regular` and `JetBrainsMono-Bold` to be loaded via Expo. Add to `app.json`:
   ```json
   {
     "expo": {
       "plugins": [
         ["expo-font", {
           "fonts": ["./assets/fonts/JetBrainsMono-Regular.ttf", "./assets/fonts/JetBrainsMono-Bold.ttf"]
         }]
       ]
     }
   }
   ```

6. **Set `API_BASE`** — create `app/src/config.ts` with your Tailscale endpoint:
   ```ts
   export const API_BASE = 'http://your-tailscale-ip:port';
   ```

7. **Use in screens** — replace the current mock `DiffView` with:
   ```tsx
   import { TokenizedDiffView } from '../components/TokenizedDiffView';
   import { useFileDiff } from '../hooks/useDiff';

   function DiffScreen({ repoPath, filePath }) {
     const { data: diff, isLoading } = useFileDiff({ repoPath, filePath });
     if (isLoading) return <ActivityIndicator />;
     return <TokenizedDiffView diff={diff} />;
   }
   ```

---

## 13. Future Considerations

- **Word-level diff highlighting** — currently diffs are line-level. A future enhancement could compare tokens within modified line pairs to highlight the specific changed words/characters (similar to GitHub's intra-line highlighting). This would be done server-side by comparing the token arrays of adjacent deleted+added line pairs.

- **Side-by-side mode** — the data model already carries `oldNum`/`newNum` per line, which is sufficient to render a side-by-side view. The RN component would need a horizontal split layout.

- **Theme switching** — if light mode is ever needed, add a `theme` query param to the API endpoints and a second color map on the server. The client would pass its current theme preference.

- **Streaming for huge diffs** — if a single file diff exceeds 2000 lines, consider streaming hunks via chunked transfer encoding or paginating the response.

- **Binary file detection** — currently the tokenizer assumes text. Add a check for binary content (null bytes in the first 8KB) and return a placeholder response instead of garbled tokens.
