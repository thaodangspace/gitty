import React, { memo, useCallback, useState } from 'react';
import {
  View,
  Text,
  FlatList,
  Pressable,
  StyleSheet,
  type ViewStyle,
} from 'react-native';
import type { DiffHunk, DiffLine, Token, TokenizedDiff } from '../types/diff';

// ─── DESIGN TOKENS (from prototype T object) ───

const T = {
  bg: '#0B0E14',
  surface: '#111620',
  surfaceHover: '#161C28',
  border: '#1E2738',
  borderLight: '#2A3548',
  text: '#E6EDF3',
  textMuted: '#7D8CA4',
  textDim: '#4A5568',
  accent: '#58C4DC',
  accentDim: 'rgba(88,196,220,0.12)',
  green: '#3FB950',
  greenDim: 'rgba(63,185,80,0.12)',
  red: '#F85149',
  redDim: 'rgba(248,81,73,0.12)',
  orange: '#D29922',
  orangeDim: 'rgba(210,153,34,0.12)',
} as const;

const LINE_BG = {
  added: 'rgba(63,185,80,0.08)',
  deleted: 'rgba(248,81,73,0.08)',
  context: 'transparent',
} as const;

const LINE_BORDER = {
  added: T.green,
  deleted: T.red,
  context: 'transparent',
} as const;

// ─── TOKEN RENDERER ───
// The core "dumb" renderer — just maps server tokens to colored Text spans.

const TokenSpan = memo(({ token }: { token: Token }) => (
  <Text style={{ color: token.color, fontFamily: 'JetBrainsMono-Regular' }}>
    {token.text}
  </Text>
));
TokenSpan.displayName = 'TokenSpan';

// ─── SINGLE DIFF LINE ───

interface DiffLineRowProps {
  line: DiffLine;
  showSideBySide?: boolean;
}

const DiffLineRow = memo(({ line }: DiffLineRowProps) => {
  return (
    <View style={[styles.lineRow, { backgroundColor: LINE_BG[line.type] }]}>
      {/* Left border indicator */}
      <View
        style={[
          styles.lineBorder,
          { backgroundColor: LINE_BORDER[line.type] },
        ]}
      />

      {/* Old line number */}
      <Text style={styles.lineNum}>
        {line.type !== 'added' ? line.oldNum || '' : ''}
      </Text>

      {/* New line number */}
      <Text style={styles.lineNum}>
        {line.type !== 'deleted' ? line.newNum || '' : ''}
      </Text>

      {/* Code content — just render the tokens */}
      <View style={styles.lineContent}>
        <Text style={styles.codeLine}>
          {line.tokens.map((token, i) => (
            <TokenSpan key={i} token={token} />
          ))}
        </Text>
      </View>
    </View>
  );
});
DiffLineRow.displayName = 'DiffLineRow';

// ─── HUNK COMPONENT ───

interface HunkViewProps {
  hunk: DiffHunk;
  index: number;
  defaultCollapsed?: boolean;
}

const HunkView = memo(({ hunk, index, defaultCollapsed = false }: HunkViewProps) => {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  // Extract the function/class context from the hunk header if present
  const headerContext = hunk.header.replace(/@@ .+? @@\s?/, '').trim();

  return (
    <View style={styles.hunk}>
      {/* Hunk header */}
      <Pressable onPress={toggleCollapse} style={styles.hunkHeader}>
        <Text style={styles.hunkHeaderText}>
          {collapsed ? '▶' : '▼'}{' '}
          <Text style={styles.hunkHeaderRange}>
            {hunk.header.match(/@@ .+? @@/)?.[0] || hunk.header}
          </Text>
          {headerContext ? (
            <Text style={styles.hunkHeaderContext}> {headerContext}</Text>
          ) : null}
        </Text>
      </Pressable>

      {/* Lines */}
      {!collapsed &&
        hunk.lines.map((line, i) => (
          <DiffLineRow key={`${index}-${i}`} line={line} />
        ))}
    </View>
  );
});
HunkView.displayName = 'HunkView';

// ─── FILE DIFF HEADER ───

interface DiffFileHeaderProps {
  filename: string;
  additions: number;
  deletions: number;
  changeType?: string;
}

const DiffFileHeader = memo(
  ({ filename, additions, deletions, changeType }: DiffFileHeaderProps) => {
    const changeColor =
      changeType === 'added'
        ? T.green
        : changeType === 'deleted'
          ? T.red
          : T.orange;

    return (
      <View style={styles.fileHeader}>
        <View style={styles.fileHeaderLeft}>
          {changeType && (
            <View
              style={[
                styles.changeTypeBadge,
                { backgroundColor: changeColor + '20' },
              ]}
            >
              <Text style={[styles.changeTypeBadgeText, { color: changeColor }]}>
                {changeType === 'added' ? 'A' : changeType === 'deleted' ? 'D' : 'M'}
              </Text>
            </View>
          )}
          <Text style={styles.fileHeaderFilename} numberOfLines={1}>
            {filename}
          </Text>
        </View>
        <View style={styles.fileHeaderStats}>
          {additions > 0 && (
            <Text style={styles.statAdd}>+{additions}</Text>
          )}
          {deletions > 0 && (
            <Text style={styles.statDel}>−{deletions}</Text>
          )}
        </View>
      </View>
    );
  },
);
DiffFileHeader.displayName = 'DiffFileHeader';

// ─── MAIN DIFF VIEW COMPONENT ───

interface TokenizedDiffViewProps {
  diff: TokenizedDiff;
  changeType?: string;
  /** Collapse hunks beyond this index (for large diffs) */
  collapseAfter?: number;
  style?: ViewStyle;
}

export const TokenizedDiffView = memo(
  ({ diff, changeType, collapseAfter = 5, style }: TokenizedDiffViewProps) => {
    if (!diff || !diff.hunks || diff.hunks.length === 0) {
      return (
        <View style={[styles.container, style]}>
          <DiffFileHeader
            filename={diff?.filename || 'unknown'}
            additions={0}
            deletions={0}
            changeType={changeType}
          />
          <View style={styles.emptyDiff}>
            <Text style={styles.emptyDiffText}>No changes</Text>
          </View>
        </View>
      );
    }

    return (
      <View style={[styles.container, style]}>
        <DiffFileHeader
          filename={diff.filename}
          additions={diff.additions}
          deletions={diff.deletions}
          changeType={changeType}
        />
        {diff.hunks.map((hunk, i) => (
          <HunkView
            key={i}
            hunk={hunk}
            index={i}
            defaultCollapsed={i >= collapseAfter}
          />
        ))}
      </View>
    );
  },
);
TokenizedDiffView.displayName = 'TokenizedDiffView';

// ─── MULTI-FILE COMMIT DIFF ───
// For rendering an entire commit's worth of file diffs in a FlatList.

interface CommitDiffListProps {
  files: Array<{
    path: string;
    changeType: string;
    diff: TokenizedDiff;
  }>;
  style?: ViewStyle;
}

export const CommitDiffList = memo(({ files, style }: CommitDiffListProps) => {
  const renderItem = useCallback(
    ({ item }: { item: CommitDiffListProps['files'][0] }) => (
      <TokenizedDiffView
        diff={item.diff}
        changeType={item.changeType}
        style={styles.fileDiffItem}
      />
    ),
    [],
  );

  const keyExtractor = useCallback(
    (item: CommitDiffListProps['files'][0]) => item.path,
    [],
  );

  return (
    <FlatList
      data={files}
      renderItem={renderItem}
      keyExtractor={keyExtractor}
      style={style}
      contentContainerStyle={styles.listContent}
      // Performance: don't render off-screen file diffs
      initialNumToRender={3}
      maxToRenderPerBatch={2}
      windowSize={5}
      removeClippedSubviews={true}
    />
  );
});
CommitDiffList.displayName = 'CommitDiffList';

// ─── STYLES ───

const FONT_SIZE = 12;
const LINE_HEIGHT = 18;
const GUTTER_WIDTH = 38;

const styles = StyleSheet.create({
  container: {
    backgroundColor: T.surface,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: T.border,
    overflow: 'hidden',
  },
  // File header
  fileHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 14,
    paddingVertical: 10,
    backgroundColor: T.surfaceHover,
    borderBottomWidth: 1,
    borderBottomColor: T.border,
  },
  fileHeaderLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    flex: 1,
  },
  changeTypeBadge: {
    paddingHorizontal: 6,
    paddingVertical: 1,
    borderRadius: 3,
  },
  changeTypeBadgeText: {
    fontSize: 10,
    fontWeight: '700',
    fontFamily: 'JetBrainsMono-Bold',
  },
  fileHeaderFilename: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 11,
    color: T.textMuted,
    flex: 1,
  },
  fileHeaderStats: {
    flexDirection: 'row',
    gap: 10,
  },
  statAdd: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 11,
    color: T.green,
  },
  statDel: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 11,
    color: T.red,
  },
  // Hunk
  hunk: {
    borderBottomWidth: 1,
    borderBottomColor: T.border,
  },
  hunkHeader: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    backgroundColor: T.accentDim,
  },
  hunkHeaderText: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 11,
    color: T.textDim,
  },
  hunkHeaderRange: {
    color: T.accent,
  },
  hunkHeaderContext: {
    color: T.textMuted,
  },
  // Lines
  lineRow: {
    flexDirection: 'row',
    minHeight: LINE_HEIGHT,
    alignItems: 'stretch',
  },
  lineBorder: {
    width: 3,
  },
  lineNum: {
    width: GUTTER_WIDTH,
    textAlign: 'right',
    paddingRight: 8,
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: FONT_SIZE,
    lineHeight: LINE_HEIGHT,
    color: T.textDim,
    opacity: 0.5,
  },
  lineContent: {
    flex: 1,
    paddingHorizontal: 12,
  },
  codeLine: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: FONT_SIZE,
    lineHeight: LINE_HEIGHT,
  },
  // Empty state
  emptyDiff: {
    padding: 24,
    alignItems: 'center',
  },
  emptyDiffText: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 12,
    color: T.textDim,
  },
  // List
  fileDiffItem: {
    marginBottom: 12,
  },
  listContent: {
    padding: 16,
  },
});

export default TokenizedDiffView;
