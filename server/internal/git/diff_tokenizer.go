package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"gitweb/server/internal/models"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// ─── DARK THEME (matches RN design tokens) ───

var darkTheme = map[chroma.TokenType]string{
	// Keywords
	chroma.Keyword:            "#C678DD",
	chroma.KeywordConstant:    "#D19A66",
	chroma.KeywordDeclaration: "#C678DD",
	chroma.KeywordNamespace:   "#C678DD",
	chroma.KeywordType:        "#E5C07B",
	chroma.KeywordReserved:    "#C678DD",

	// Names
	chroma.Name:              "#E6EDF3",
	chroma.NameBuiltin:       "#61AFEF",
	chroma.NameClass:         "#E5C07B",
	chroma.NameFunction:      "#61AFEF",
	chroma.NameDecorator:     "#D19A66",
	chroma.NameException:     "#E5C07B",
	chroma.NameTag:           "#E06C75",
	chroma.NameAttribute:     "#D19A66",
	chroma.NameVariable:      "#E06C75",
	chroma.NameConstant:      "#D19A66",
	chroma.NameOther:         "#E6EDF3",
	chroma.NameProperty:      "#E06C75",
	chroma.NameEntity:        "#D19A66",
	chroma.NameLabel:         "#61AFEF",
	chroma.NameNamespace:     "#E5C07B",
	chroma.NameBuiltinPseudo: "#E5C07B",

	// Literals
	chroma.LiteralString:          "#98C379",
	chroma.LiteralStringDouble:    "#98C379",
	chroma.LiteralStringSingle:    "#98C379",
	chroma.LiteralStringBacktick:  "#98C379",
	chroma.LiteralStringEscape:    "#56B6C2",
	chroma.LiteralStringRegex:     "#56B6C2",
	chroma.LiteralStringInterpol:  "#D19A66",
	chroma.LiteralNumber:          "#D19A66",
	chroma.LiteralNumberFloat:     "#D19A66",
	chroma.LiteralNumberHex:       "#D19A66",
	chroma.LiteralNumberInteger:   "#D19A66",
	chroma.LiteralNumberOct:       "#D19A66",

	// Operators
	chroma.Operator:     "#56B6C2",
	chroma.OperatorWord: "#C678DD",
	chroma.Punctuation:  "#ABB2BF",

	// Comments
	chroma.Comment:          "#5C6370",
	chroma.CommentSingle:    "#5C6370",
	chroma.CommentMultiline: "#5C6370",
	chroma.CommentHashbang:  "#5C6370",
	chroma.CommentPreproc:   "#C678DD",

	// Generic (fallback)
	chroma.GenericEmph:    "#E6EDF3",
	chroma.GenericStrong:  "#E6EDF3",
	chroma.GenericHeading: "#61AFEF",

	// Text
	chroma.Text: "#E6EDF3",
}

const defaultColor = "#E6EDF3"

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

// TokenizeDiffFromPatch is a convenience method that works with the existing
// GetFileDiff / GetStagedDiff output.
func (s *Service) TokenizeDiffFromPatch(repoPath, filePath string, staged bool, cursor int, limit int) (*models.TokenizedDiff, error) {
	var diffText string
	var err error

	if staged {
		diffText, err = s.GetStagedDiff(repoPath, filePath)
	} else {
		diffText, err = s.GetFileDiff(repoPath, filePath)
	}
	if err != nil {
		return nil, err
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
