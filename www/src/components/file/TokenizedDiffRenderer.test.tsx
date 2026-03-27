import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TokenizedDiffRenderer from './TokenizedDiffRenderer';
import type { TokenizedDiff } from '../../types/api';

const mockDiff: TokenizedDiff = {
  filename: 'test.ts',
  hunks: [
    {
      header: '@@ -1,3 +1,4 @@',
      blocks: [
        {
          type: 'context',
          lines: [
            {
              type: 'context',
              tokens: [{ text: 'const x = 1;', color: '#C678DD' }],
              oldNum: 1,
              newNum: 1,
            },
          ],
          startOld: 1,
          endOld: 1,
          startNew: 1,
          endNew: 1,
          collapsed: false,
        },
        {
          type: 'added',
          lines: [
            {
              type: 'added',
              tokens: [{ text: 'const y = 2;', color: '#98C379' }],
              newNum: 2,
            },
          ],
          startOld: 0,
          endOld: 0,
          startNew: 2,
          endNew: 2,
          collapsed: false,
        },
        {
          type: 'deleted',
          lines: [
            {
              type: 'deleted',
              tokens: [{ text: 'const z = 3;', color: '#E06C75' }],
              oldNum: 2,
            },
          ],
          startOld: 2,
          endOld: 2,
          startNew: 0,
          endNew: 0,
          collapsed: false,
        },
      ],
    },
  ],
  additions: 1,
  deletions: 1,
  has_more: false,
  total_hunks: 1,
};

describe('TokenizedDiffRenderer', () => {
  it('renders empty state when no hunks', () => {
    const emptyDiff: TokenizedDiff = {
      filename: 'test.ts',
      hunks: [],
      additions: 0,
      deletions: 0,
      has_more: false,
      total_hunks: 0,
    };

    render(<TokenizedDiffRenderer diff={emptyDiff} />);
    expect(screen.getByText('No changes to display')).toBeInTheDocument();
  });

  it('renders diff hunks with blocks and tokens', () => {
    render(<TokenizedDiffRenderer diff={mockDiff} />);

    // Check for hunk header
    expect(screen.getByText('@@ -1,3 +1,4 @@')).toBeInTheDocument();

    // Check for context line
    expect(screen.getByText('const x = 1;')).toBeInTheDocument();

    // Check for added line
    expect(screen.getByText('const y = 2;')).toBeInTheDocument();

    // Check for deleted line
    expect(screen.getByText('const z = 3;')).toBeInTheDocument();
  });

  it('renders a single line number column per row', () => {
    render(<TokenizedDiffRenderer diff={mockDiff} />);

    const addedRow = screen.getByText('const y = 2;').closest('.flex');
    const deletedRow = screen.getByText('const z = 3;').closest('.flex');

    expect(addedRow?.querySelectorAll('.w-12')).toHaveLength(1);
    expect(deletedRow?.querySelectorAll('.w-12')).toHaveLength(1);
  });

  it('applies correct background colors for line types', () => {
    render(<TokenizedDiffRenderer diff={mockDiff} />);

    // Check for added line with green tint and border
    const addedLineDiv = Array.from(document.querySelectorAll('.flex')).find(div =>
      div.textContent?.includes('const y = 2;') &&
      (div as HTMLElement).style.backgroundColor.includes('46, 160, 67')
    );

    // Check for deleted line with red tint and border
    const deletedLineDiv = Array.from(document.querySelectorAll('.flex')).find(div =>
      div.textContent?.includes('const z = 3;') &&
      (div as HTMLElement).style.backgroundColor.includes('248, 81, 73')
    );

    expect(addedLineDiv).toBeDefined();
    expect(deletedLineDiv).toBeDefined();
  });

  it('handles empty tokens array gracefully', () => {
    const diffWithEmptyTokens: TokenizedDiff = {
      filename: 'test.ts',
      hunks: [
        {
          header: '@@ -1,1 +1,1 @@',
          blocks: [
            {
              type: 'context',
              lines: [
                {
                  type: 'context',
                  tokens: [],
                  oldNum: 1,
                  newNum: 1,
                },
              ],
              startOld: 1,
              endOld: 1,
              startNew: 1,
              endNew: 1,
              collapsed: false,
            },
          ],
        },
      ],
      additions: 0,
      deletions: 0,
      has_more: false,
      total_hunks: 1,
    };

    render(<TokenizedDiffRenderer diff={diffWithEmptyTokens} />);
    // Should render without crashing and show at least one line number.
    const lineNumbers = screen.getAllByText('1');
    expect(lineNumbers.length).toBeGreaterThan(0);
  });

  it('displays line number 0 correctly (not as dash) for the active column', () => {
    const diffWithZeroLineNum: TokenizedDiff = {
      filename: 'test.ts',
      hunks: [
        {
          header: '@@ -0,0 +1,1 @@',
          blocks: [
            {
              type: 'deleted',
              lines: [
                {
                  type: 'deleted',
                  tokens: [{ text: 'old line', color: '#E06C75' }],
                  oldNum: 0,
                },
              ],
              startOld: 0,
              endOld: 0,
              startNew: 0,
              endNew: 0,
              collapsed: false,
            },
          ],
        },
      ],
      additions: 0,
      deletions: 1,
      has_more: false,
      total_hunks: 1,
    };

    render(<TokenizedDiffRenderer diff={diffWithZeroLineNum} />);
    // Line number 0 should display as "0", not "-"
    const lineNumbers = screen.getAllByText('0');
    expect(lineNumbers.length).toBeGreaterThan(0);
  });

  it('handles collapsed blocks with expand toggle', async () => {
    const diffWithCollapsed: TokenizedDiff = {
      filename: 'test.ts',
      hunks: [
        {
          header: '@@ -1,10 +1,10 @@',
          blocks: [
            {
              type: 'context',
              lines: Array(10).fill({
                type: 'context',
                tokens: [{ text: 'line', color: '#E6EDF3' }],
                oldNum: 1,
                newNum: 1,
              }),
              startOld: 1,
              endOld: 10,
              startNew: 1,
              endNew: 10,
              collapsed: true,
            },
          ],
        },
      ],
      additions: 0,
      deletions: 0,
      has_more: false,
      total_hunks: 1,
    };

    render(<TokenizedDiffRenderer diff={diffWithCollapsed} />);

    // Should show collapsed indicator initially
    expect(screen.getByText('... 10 lines ...')).toBeInTheDocument();

    // Click to expand
    await userEvent.click(screen.getByText('... 10 lines ...'));

    // Should now show "Collapse" button
    expect(screen.getByText('Collapse')).toBeInTheDocument();
  });
});
