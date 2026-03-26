import { useState } from 'react';
import { useAtom } from 'jotai';
import { ChevronRight, ChevronDown } from 'lucide-react';
import { themeAtom } from '@/store/atoms/ui-atoms';
import type { TokenizedDiff, DiffHunkTokenized, DiffBlock, DiffLineTokenized } from '../../types/api';

interface TokenizedDiffRendererProps {
  diff: TokenizedDiff;
}

export default function TokenizedDiffRenderer({ diff }: TokenizedDiffRendererProps) {
  const [theme] = useAtom(themeAtom);

  // Handle empty diff
  if (!diff.hunks || diff.hunks.length === 0) {
    return (
      <div className="flex items-center justify-center h-48">
        <p className="text-muted-foreground dark:text-gray-400">No changes to display</p>
      </div>
    );
  }

  return (
    <div className="font-mono text-sm">
      {diff.hunks.map((hunk, hunkIndex) => (
        <DiffHunk key={hunkIndex} hunk={hunk} theme={theme} />
      ))}
    </div>
  );
}

interface DiffHunkProps {
  hunk: DiffHunkTokenized;
  theme: 'light' | 'dark' | 'system';
}

function DiffHunk({ hunk, theme }: DiffHunkProps) {
  return (
    <div className="mb-4">
      {/* Hunk header */}
      <div className="text-gray-500 dark:text-gray-400 px-2 py-1 bg-gray-50 dark:bg-gray-800 border-b dark:border-gray-700">
        {hunk.header}
      </div>

      {/* Blocks */}
      <div>
        {hunk.blocks.map((block, blockIndex) => (
          <DiffBlock key={blockIndex} block={block} theme={theme} />
        ))}
      </div>
    </div>
  );
}

interface DiffBlockProps {
  block: DiffBlock;
  theme: 'light' | 'dark' | 'system';
}

function DiffBlock({ block }: DiffBlockProps) {
  const [isExpanded, setIsExpanded] = useState(!block.collapsed);

  if (block.collapsed && !isExpanded) {
    return (
      <div
        className="flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400"
        onClick={() => setIsExpanded(true)}
      >
        <ChevronRight className="h-4 w-4" />
        <span>... {block.lines.length} lines ...</span>
      </div>
    );
  }

  return (
    <div>
      {block.lines.map((line, lineIndex) => (
        <DiffLine key={lineIndex} line={line} type={block.type} />
      ))}
      {block.collapsed && (
        <div
          className="flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400"
          onClick={() => setIsExpanded(false)}
        >
          <ChevronDown className="h-4 w-4" />
          <span>Collapse</span>
        </div>
      )}
    </div>
  );
}

interface DiffLineProps {
  line: DiffLineTokenized;
  type: 'added' | 'deleted' | 'context';
}

function DiffLine({ line, type }: DiffLineProps) {
  const lineStyle = {
    added: {
      backgroundColor: 'rgba(46, 160, 67, 0.08)',
      borderLeft: '3px solid #2ea043',
    },
    deleted: {
      backgroundColor: 'rgba(248, 81, 73, 0.08)',
      borderLeft: '3px solid #f85149',
    },
    context: {
      backgroundColor: 'transparent',
      borderLeft: '3px solid transparent',
    },
  }[type];

  const prefix = {
    added: '+',
    deleted: '-',
    context: ' ',
  }[type];

  // Handle empty tokens array
  if (!line.tokens || line.tokens.length === 0) {
    return (
      <div style={lineStyle} className="flex hover:bg-gray-100 dark:hover:bg-gray-800">
        <div className="w-12 text-right pr-2 text-gray-400 select-none">
          {line.oldNum !== undefined ? line.oldNum : '-'}
        </div>
        <div className="w-12 text-right pr-2 text-gray-400 select-none">
          {line.newNum !== undefined ? line.newNum : '-'}
        </div>
        <div className="flex-1 px-2 py-0.5">
          <span className="text-gray-400">{prefix}</span>
        </div>
      </div>
    );
  }

  return (
    <div style={lineStyle} className="flex hover:bg-gray-100 dark:hover:bg-gray-800">
      {/* Old line number */}
      <div className="w-12 text-right pr-2 text-gray-400 select-none">
        {line.oldNum !== undefined ? line.oldNum : '-'}
      </div>
      {/* New line number */}
      <div className="w-12 text-right pr-2 text-gray-400 select-none">
        {line.newNum !== undefined ? line.newNum : '-'}
      </div>
      {/* Line content with tokens */}
      <div className="flex-1 px-2 py-0.5 whitespace-pre overflow-x-auto">
        <span className="text-gray-400 select-none">{prefix}</span>
        {line.tokens.map((token, tokenIndex) => (
          <span
            key={tokenIndex}
            style={{ color: token.color }}
            className="font-mono"
          >
            {token.text}
          </span>
        ))}
      </div>
    </div>
  );
}
