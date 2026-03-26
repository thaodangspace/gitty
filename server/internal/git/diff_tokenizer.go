package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitweb/server/internal/models"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// ─── DARK THEME (matches RN design tokens) ───

var darkTheme = map[chroma.TokenType]string{
	// Keywords
	chroma.Keyword:            "#d2a8ff",
	chroma.KeywordConstant:    "#ffab70",
	chroma.KeywordDeclaration: "#d2a8ff",
	chroma.KeywordNamespace:   "#d2a8ff",
	chroma.KeywordType:        "#ffa657",
	chroma.KeywordReserved:    "#d2a8ff",

	// Names
	chroma.Name:              "#e6edf3",
	chroma.NameBuiltin:       "#79c0ff",
	chroma.NameClass:         "#ffa657",
	chroma.NameFunction:      "#79c0ff",
	chroma.NameDecorator:     "#ffab70",
	chroma.NameException:     "#ffa657",
	chroma.NameTag:           "#f85149",
	chroma.NameAttribute:     "#ffab70",
	chroma.NameVariable:      "#f85149",
	chroma.NameConstant:      "#ffab70",
	chroma.NameOther:         "#e6edf3",
	chroma.NameProperty:      "#f85149",
	chroma.NameEntity:        "#ffab70",
	chroma.NameLabel:         "#79c0ff",
	chroma.NameNamespace:     "#ffa657",
	chroma.NameBuiltinPseudo: "#ffa657",

	// Literals
	chroma.LiteralString:          "#7ee787",
	chroma.LiteralStringDouble:    "#7ee787",
	chroma.LiteralStringSingle:    "#7ee787",
	chroma.LiteralStringBacktick:  "#7ee787",
	chroma.LiteralStringEscape:    "#76e3ea",
	chroma.LiteralStringRegex:     "#76e3ea",
	chroma.LiteralStringInterpol:  "#ffab70",
	chroma.LiteralNumber:          "#ffab70",
	chroma.LiteralNumberFloat:     "#ffab70",
	chroma.LiteralNumberHex:       "#ffab70",
	chroma.LiteralNumberInteger:   "#ffab70",
	chroma.LiteralNumberOct:       "#ffab70",

	// Operators
	chroma.Operator:     "#76e3ea",
	chroma.OperatorWord: "#d2a8ff",
	chroma.Punctuation:  "#8b949e",

	// Comments
	chroma.Comment:          "#8b949e",
	chroma.CommentSingle:    "#8b949e",
	chroma.CommentMultiline: "#8b949e",
	chroma.CommentHashbang:  "#8b949e",
	chroma.CommentPreproc:   "#d2a8ff",

	// Generic (fallback)
	chroma.GenericEmph:    "#e6edf3",
	chroma.GenericStrong:  "#e6edf3",
	chroma.GenericHeading: "#79c0ff",

	// Text
	chroma.Text: "#e6edf3",
}

const defaultColor = "#e6edf3"

func colorForToken(tokenType chroma.TokenType) string {
	// Walk up the token type hierarchy to find a matching color
	for tt := tokenType; tt != chroma.Error; tt = tt.Parent() {
		if color, ok := darkTheme[tt]; ok {
			return color
		}
	}
	return defaultColor
}

// ─── LEXER DETECTION ───

func lexerForFile(filename string) chroma.Lexer {
	// Try by filename first
	lexer := lexers.Match(filename)
	if lexer != nil {
		return chroma.Coalesce(lexer)
	}

	// Fallback by extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".ts", ".tsx":
		lexer = lexers.Get("typescript")
	case ".js", ".jsx":
		lexer = lexers.Get("javascript")
	case ".go":
		lexer = lexers.Get("go")
	case ".py":
		lexer = lexers.Get("python")
	case ".rs":
		lexer = lexers.Get("rust")
	case ".rb":
		lexer = lexers.Get("ruby")
	case ".java":
		lexer = lexers.Get("java")
	case ".kt":
		lexer = lexers.Get("kotlin")
	case ".swift":
		lexer = lexers.Get("swift")
	case ".css":
		lexer = lexers.Get("css")
	case ".scss":
		lexer = lexers.Get("scss")
	case ".html":
		lexer = lexers.Get("html")
	case ".json":
		lexer = lexers.Get("json")
	case ".yaml", ".yml":
		lexer = lexers.Get("yaml")
	case ".md":
		lexer = lexers.Get("markdown")
	case ".sh", ".bash":
		lexer = lexers.Get("bash")
	case ".sql":
		lexer = lexers.Get("sql")
	case ".tf":
		lexer = lexers.Get("hcl")
	case ".proto":
		lexer = lexers.Get("protobuf")
	case ".toml":
		lexer = lexers.Get("toml")
	case ".xml":
		lexer = lexers.Get("xml")
	case ".c", ".h":
		lexer = lexers.Get("c")
	case ".cpp", ".hpp", ".cc":
		lexer = lexers.Get("c++")
	}

	if lexer == nil {
		lexer = lexers.Get("plaintext")
	}
	return chroma.Coalesce(lexer)
}

// ─── TOKENIZE A SINGLE LINE ───

// tokenizeLine runs the lexer on a single line of code and returns colored tokens.
// We re-lex per line which is slightly less accurate than full-file lexing for
// multi-line constructs, but avoids having to correlate full-file token offsets
// back to individual diff lines. For the diff use-case this is a good tradeoff.
func tokenizeLine(lexer chroma.Lexer, line string) []models.Token {
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		// Fallback: return the whole line as plain text
		return []models.Token{{Text: line, Color: defaultColor}}
	}

	tokens := []models.Token{}
	for _, tok := range iterator.Tokens() {
		text := tok.Value
		if text == "" {
			continue
		}
		// Skip trailing newlines from the tokenizer
		text = strings.TrimRight(text, "\n")
		if text == "" {
			continue
		}

		color := colorForToken(tok.Type)
		// Merge adjacent tokens with the same color
		if len(tokens) > 0 && tokens[len(tokens)-1].Color == color {
			tokens[len(tokens)-1].Text += text
		} else {
			tokens = append(tokens, models.Token{Text: text, Color: color})
		}
	}

	if len(tokens) == 0 {
		tokens = append(tokens, models.Token{Text: line, Color: defaultColor})
	}

	return tokens
}

// ─── FULL-FILE TOKENIZATION ───
// For better accuracy with multi-line strings/comments, tokenize the full
// source text and then split the result into per-line token slices.

func tokenizeFullSource(lexer chroma.Lexer, lines []string) [][]models.Token {
	fullSource := strings.Join(lines, "\n")

	iterator, err := lexer.Tokenise(nil, fullSource)
	if err != nil {
		// Fallback to per-line tokenization
		result := make([][]models.Token, len(lines))
		for i, line := range lines {
			result[i] = tokenizeLine(lexer, line)
		}
		return result
	}

	// Build per-line token arrays
	result := make([][]models.Token, len(lines))
	for i := range result {
		result[i] = []models.Token{}
	}

	lineIdx := 0
	for _, tok := range iterator.Tokens() {
		if tok.Value == "" {
			continue
		}

		color := colorForToken(tok.Type)

		// A single token can span multiple lines (e.g. multi-line strings).
		// Split it at newline boundaries and distribute to the right line.
		parts := strings.Split(tok.Value, "\n")
		for pi, part := range parts {
			if pi > 0 {
				lineIdx++
			}
			if lineIdx >= len(lines) {
				break
			}
			if part == "" {
				continue
			}

			row := &result[lineIdx]
			// Merge adjacent tokens with the same color
			if len(*row) > 0 && (*row)[len(*row)-1].Color == color {
				(*row)[len(*row)-1].Text += part
			} else {
				*row = append(*row, models.Token{Text: part, Color: color})
			}
		}
	}

	// Ensure no line is empty (provide placeholder)
	for i, line := range result {
		if len(line) == 0 {
			result[i] = []models.Token{{Text: lines[i], Color: defaultColor}}
		}
	}

	return result
}

// ─── BLOCK ASSEMBLY HELPERS ───

const collapseThreshold = 6

// flushBlock appends the current block to the hunk and handles collapse logic.
// Returns nil to clear the block pointer.
func flushBlock(block *models.DiffBlock, hunk *models.DiffHunkTokenized) *models.DiffBlock {
	if block == nil || len(block.Lines) == 0 {
		return nil
	}
	// Auto-collapse context blocks >= 6 lines
	if block.Type == "context" && len(block.Lines) >= collapseThreshold {
		block.Collapsed = true
	}
	hunk.Blocks = append(hunk.Blocks, *block)
	return nil
}

// newBlock creates a new block from a parsed diff line.
func newBlock(dl rawDiffLine, tokens []models.Token) *models.DiffBlock {
	return &models.DiffBlock{
		Type:     dl.lineType,
		Lines:    []models.DiffLineTokenized{{Type: dl.lineType, Tokens: tokens, OldNum: dl.oldNum, NewNum: dl.newNum}},
		StartOld: dl.oldNum,
		EndOld:   dl.oldNum,
		StartNew: dl.newNum,
		EndNew:   dl.newNum,
	}
}

// appendToBlock adds a line to an existing block and updates line number ranges.
func appendToBlock(block *models.DiffBlock, dl rawDiffLine, tokens []models.Token) {
	block.Lines = append(block.Lines, models.DiffLineTokenized{
		Type:   dl.lineType,
		Tokens: tokens,
		OldNum: dl.oldNum,
		NewNum: dl.newNum,
	})
	if dl.oldNum > 0 {
		block.EndOld = dl.oldNum
	}
	if dl.newNum > 0 {
		block.EndNew = dl.newNum
	}
}

// ─── PARSE UNIFIED DIFF INTO HUNKS ───

type rawDiffLine struct {
	lineType string // "added", "deleted", "context", "header"
	content  string // code content (without +/- prefix)
	oldNum   int    // line number in old file (0 if N/A)
	newNum   int    // line number in new file (0 if N/A)
}

// parseDiffContent takes a raw unified diff string and returns structured lines.
func parseDiffContent(diffText string) ([]rawDiffLine, string) {
	lines := strings.Split(diffText, "\n")
	result := []rawDiffLine{}

	// Extract the file header (first few lines before @@)
	headerEnd := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "@@") {
			headerEnd = i
			break
		}
	}

	oldLine := 0
	newLine := 0

	for i := headerEnd; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "@@") {
			// Parse hunk header: @@ -a,b +c,d @@
			result = append(result, rawDiffLine{
				lineType: "header",
				content:  line,
			})
			// Extract line numbers
			// Go's Sscanf doesn't support %*d (discard), so we parse with dummy variables
			var oldCount, newCount int
			fmt.Sscanf(line, "@@ -%d,%d +%d,%d", &oldLine, &oldCount, &newLine, &newCount)
			// Handle alternate format without counts: @@ -a +b @@
			if newLine == 0 {
				fmt.Sscanf(line, "@@ -%d +%d", &oldLine, &newLine)
			}
			continue
		}

		if line == "" && i == len(lines)-1 {
			// Skip trailing empty line
			continue
		}

		if strings.HasPrefix(line, "+") {
			result = append(result, rawDiffLine{
				lineType: "added",
				content:  line[1:],
				newNum:   newLine,
			})
			newLine++
		} else if strings.HasPrefix(line, "-") {
			result = append(result, rawDiffLine{
				lineType: "deleted",
				content:  line[1:],
				oldNum:   oldLine,
			})
			oldLine++
		} else if strings.HasPrefix(line, " ") {
			result = append(result, rawDiffLine{
				lineType: "context",
				content:  line[1:],
				oldNum:   oldLine,
				newNum:   newLine,
			})
			oldLine++
			newLine++
		} else if line == "\\ No newline at end of file" {
			// Skip this marker
			continue
		} else {
			// Treat as context
			result = append(result, rawDiffLine{
				lineType: "context",
				content:  line,
				oldNum:   oldLine,
				newNum:   newLine,
			})
			oldLine++
			newLine++
		}
	}

	// Build a filename from the diff header
	filename := ""
	for i := 0; i < headerEnd && i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "+++ b/") {
			filename = strings.TrimPrefix(lines[i], "+++ b/")
			break
		} else if strings.HasPrefix(lines[i], "+++ ") {
			filename = strings.TrimPrefix(lines[i], "+++ ")
			break
		}
	}

	return result, filename
}

// ─── GIT DIFF COMMAND OPTIMIZATION ───

// GetFileDiffUsingGitDiff executes "git diff HEAD <file>" directly using the git CLI.
// This is faster than using go-git library for large files as it leverages git's
// optimized diff algorithm and avoids loading full file contents into memory.
func (s *Service) GetFileDiffUsingGitDiff(repoPath, filePath string) (string, error) {
	cmd := exec.Command("git", "diff", "HEAD", "--", filePath)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Check if it's an exit error (file might be new/untracked)
		if exitErr, ok := err.(*exec.ExitError); ok {
			// If git diff returns no output but exits with error, file might be new
			if len(exitErr.Stderr) == 0 || string(exitErr.Stderr) == "" {
				// Try checking if file exists in working directory
				fullPath := filepath.Join(repoPath, filePath)
				if _, statErr := os.Stat(fullPath); statErr == nil {
					// File exists but git diff returned empty - likely a new untracked file
					// Fall back to manual diff generation for untracked files
					return s.getUntrackedFileDiff(repoPath, filePath)
				}
			}
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	diffText := string(output)
	// If diff is empty, check if file exists (might be untracked)
	if strings.TrimSpace(diffText) == "" {
		fullPath := filepath.Join(repoPath, filePath)
		if _, statErr := os.Stat(fullPath); statErr == nil {
			// File exists but git diff HEAD returned empty
			// Check if file is untracked (not in git)
			isTracked := s.isFileTracked(repoPath, filePath)
			if !isTracked {
				// Untracked file - generate diff showing all content as additions
				return s.getUntrackedFileDiff(repoPath, filePath)
			}
		}
		return "", nil
	}
	return diffText, nil
}

// isFileTracked checks if a file is tracked by git using "git ls-files"
func (s *Service) isFileTracked(repoPath, filePath string) bool {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", "--", filePath)
	cmd.Dir = repoPath
	err := cmd.Run()
	return err == nil
}

// GetStagedDiffUsingGitDiff executes "git diff --cached <file>" directly using the git CLI.
// This gets the diff between the staged content (index) and HEAD.
func (s *Service) GetStagedDiffUsingGitDiff(repoPath, filePath string) (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--", filePath)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if it's because file is not staged
			if len(exitErr.Stderr) > 0 {
				return "", fmt.Errorf("git diff --cached failed: %s", string(exitErr.Stderr))
			}
		}
		return "", fmt.Errorf("git diff --cached failed: %w", err)
	}

	diffText := string(output)
	// If diff is empty, return empty string
	if strings.TrimSpace(diffText) == "" {
		return "", nil
	}
	return diffText, nil
}

// getUntrackedFileDiff generates a diff for an untracked/new file.
// Shows the entire file content as additions.
func (s *Service) getUntrackedFileDiff(repoPath, filePath string) (string, error) {
	fullPath := filepath.Join(repoPath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	diff.WriteString("new file mode 100644\n")
	diff.WriteString("index 0000000..0000000\n")
	diff.WriteString("--- /dev/null\n")
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 0 {
		diff.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
		for _, line := range lines {
			diff.WriteString("+" + line + "\n")
		}
	}

	return diff.String(), nil
}

// ─── PUBLIC API ───

// TokenizeDiff takes a unified diff string and a filename, and returns a
// fully tokenized diff ready for the RN client to render.
func (s *Service) TokenizeDiff(diffText string, filename string, cursor int, limit int) *models.TokenizedDiff {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	parsed, detectedFile := parseDiffContent(diffText)

	// Use provided filename, fallback to detected
	if filename == "" {
		filename = detectedFile
	}

	lexer := lexerForFile(filename)

	// Collect all code lines (stripping diff markers) for full-file tokenization.
	// We do two passes: one for "old" content (context + deleted) and one for
	// "new" content (context + added). This gives better accuracy for multi-line
	// constructs that appear only on one side.
	var oldCodeLines []string
	var newCodeLines []string
	var oldIndices []int // index into parsed for each old code line
	var newIndices []int

	for i, dl := range parsed {
		switch dl.lineType {
		case "context":
			oldCodeLines = append(oldCodeLines, dl.content)
			newCodeLines = append(newCodeLines, dl.content)
			oldIndices = append(oldIndices, i)
			newIndices = append(newIndices, i)
		case "deleted":
			oldCodeLines = append(oldCodeLines, dl.content)
			oldIndices = append(oldIndices, i)
		case "added":
			newCodeLines = append(newCodeLines, dl.content)
			newIndices = append(newIndices, i)
		}
	}

	// Tokenize both sides
	oldTokenized := tokenizeFullSource(lexer, oldCodeLines)
	newTokenized := tokenizeFullSource(lexer, newCodeLines)

	// Build lookup: parsedIndex → tokens
	tokenMap := make(map[int][]models.Token)
	for i, idx := range oldIndices {
		tokenMap[idx] = oldTokenized[i]
	}
	for i, idx := range newIndices {
		// For context lines, both sides should produce the same tokens,
		// so overwriting is fine.
		tokenMap[idx] = newTokenized[i]
	}

	// Build the output with block grouping
	result := &models.TokenizedDiff{
		Filename: filename,
		Hunks:    []models.DiffHunkTokenized{},
	}

	var allHunks []models.DiffHunkTokenized
	var currentHunk *models.DiffHunkTokenized
	var currentBlock *models.DiffBlock
	totalAdd := 0
	totalDel := 0

	for i, dl := range parsed {
		if dl.lineType == "header" {
			// Flush current block and hunk
			currentBlock = flushBlock(currentBlock, currentHunk)
			if currentHunk != nil {
				allHunks = append(allHunks, *currentHunk)
			}
			// Start new hunk
			currentHunk = &models.DiffHunkTokenized{
				Header: dl.content,
				Blocks: []models.DiffBlock{},
			}
			currentBlock = nil
			continue
		}

		// Create hunk if needed (shouldn't happen with valid diffs)
		if currentHunk == nil {
			currentHunk = &models.DiffHunkTokenized{
				Header: "@@ @@",
				Blocks: []models.DiffBlock{},
			}
		}

		// Get tokens for this line
		tokens := tokenMap[i]
		if tokens == nil {
			tokens = []models.Token{{Text: dl.content, Color: defaultColor}}
		}

		// Handle block assembly
		if currentBlock == nil {
			currentBlock = newBlock(dl, tokens)
		} else if currentBlock.Type == dl.lineType {
			appendToBlock(currentBlock, dl, tokens)
		} else {
			// Type changed - flush and start new block
			currentBlock = flushBlock(currentBlock, currentHunk)
			currentBlock = newBlock(dl, tokens)
		}

		// Count additions/deletions
		switch dl.lineType {
		case "added":
			totalAdd++
		case "deleted":
			totalDel++
		}
	}

	// Flush remaining block and hunk
	currentBlock = flushBlock(currentBlock, currentHunk)
	if currentHunk != nil {
		allHunks = append(allHunks, *currentHunk)
	}

	result.TotalHunks = len(allHunks)
	result.Additions = totalAdd
	result.Deletions = totalDel

	// Apply pagination on hunks
	startIdx := cursor
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(allHunks) {
		// Out of bounds, return empty hunks
		result.HasMore = false
		result.NextCursor = 0
		return result
	}

	endIdx := startIdx + limit
	if endIdx >= len(allHunks) {
		endIdx = len(allHunks)
		result.HasMore = false
		result.NextCursor = 0
	} else {
		result.HasMore = true
		result.NextCursor = endIdx
	}

	result.Hunks = allHunks[startIdx:endIdx]

	return result
}

// TokenizeDiffFromPatch is a convenience method that gets the diff using optimized
// git diff commands and returns a fully tokenized diff ready for rendering.
func (s *Service) TokenizeDiffFromPatch(repoPath, filePath string, staged bool, cursor int, limit int) (*models.TokenizedDiff, error) {
	var diffText string
	var err error

	// Use optimized git diff path
	if staged {
		diffText, err = s.GetStagedDiffUsingGitDiff(repoPath, filePath)
	} else {
		diffText, err = s.GetFileDiffUsingGitDiff(repoPath, filePath)
	}
	if err != nil {
		return nil, err
	}

	// If no diff, return empty result
	if diffText == "" {
		return &models.TokenizedDiff{
			Filename:  filePath,
			Hunks:     []models.DiffHunkTokenized{},
			TotalHunks: 0,
			Additions: 0,
			Deletions: 0,
			HasMore:   false,
			NextCursor: 0,
		}, nil
	}

	return s.TokenizeDiff(diffText, filePath, cursor, limit), nil
}

// TokenizeCommitDiff tokenizes all file diffs in a commit detail.
func (s *Service) TokenizeCommitDiff(repoPath, commitHash string) (*models.TokenizedCommitDiff, error) {
	detail, err := s.GetCommitDetails(repoPath, commitHash)
	if err != nil {
		return nil, err
	}

	result := &models.TokenizedCommitDiff{
		Hash:    detail.Hash,
		Message: detail.Message,
		Author:  detail.Author,
		Date:    detail.Date,
		Files:   []models.TokenizedFileDiff{},
		Stats:   detail.Stats,
	}

	for _, change := range detail.Changes {
		tokenized := s.TokenizeDiff(change.Patch, change.Path, 0, 9999) // don't paginate commit diff files for now
		result.Files = append(result.Files, models.TokenizedFileDiff{
			Path:       change.Path,
			ChangeType: change.ChangeType,
			Diff:       *tokenized,
		})
	}

	return result, nil
}
