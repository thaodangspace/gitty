# Vim Motion Support

Gitty now supports vim-style keyboard navigation throughout the UI!

## Enabling Vim Mode

Press `v` to toggle vim mode on/off. When enabled, you'll see a **VIM** indicator in the status bar at the bottom of the screen.

## Keybindings

### Global Controls
- `v` - Toggle vim mode on/off
- `Esc` - Exit vim mode
- `Tab` - Switch between header and content areas

### Header Navigation (Shift + H/L)
Use `Shift+H` (left) and `Shift+L` (right) to navigate between all header buttons:
- Repository selector
- View tabs (Changes, Branches, History, Tree)
- Refresh button
- Settings button

Press `Enter` to activate the currently focused button.

### List Navigation (j/k)

#### In Commit History View
- `j` - Move down to next commit
- `k` - Move up to previous commit
- `Enter` - View commit details

#### In Branch List View
- `j` - Move down to next branch
- `k` - Move up to previous branch
- `Enter` - Switch to the selected branch

#### In Changes View (Working Directory)
- `j` - Move down through all file changes (staged, modified, untracked)
- `k` - Move up through all file changes
- `Enter` - View diff for the selected file
- `Space` - Stage/unstage the selected file (coming soon)

### File Tree Navigation (j/k/h/l)
- `j` - Move down to next file/folder
- `k` - Move up to previous file/folder
- `h` - Collapse the currently focused folder
- `l` - Expand the currently focused folder
- `Enter` - Open file or toggle folder

### Context Switching
Vim navigation automatically switches between contexts as you interact with different parts of the UI:
- **Tab**: Toggle between header and content (most efficient way!)
- Click on a list to activate that context
- Press `Shift+H` or `Shift+L` to switch to header context
- Navigate to a different view to switch contexts automatically

**Pro Tip**: Use `Tab` to quickly jump between navigating the header menu and the content area. The system remembers which content context you were last in (commits, branches, files, etc.)

## Visual Feedback

When vim mode is active:
1. **Status Bar**: Shows "VIM - [Context] [current/total]" indicator
2. **Focus Ring**: Currently focused element has a blue ring around it
3. **Background Highlight**: Focused list items have a light blue background

## Tips

- **Tab is your friend**: The fastest way to switch between navigating the header menu and content area
- **Context-aware navigation**: The j/k/h/l keys work differently depending on which part of the UI you're in
- **Quick access**: Use `Shift+H/L` to quickly navigate between major UI sections via the header
- **Tree navigation**: In the file tree, use h/l to collapse/expand folders without moving focus
- **Efficiency**: Once you learn the keybindings, you can navigate the entire application without touching the mouse!
- **Smart context memory**: When you Tab back to content, you return to the same type of content you were viewing (commits, branches, etc.)

## Example Workflows

### Reviewing Commits (using Tab)
1. Press `v` to enable vim mode
2. Press `Tab` to switch to content area (if not already there)
3. Press `Shift+L` until "History" is highlighted in header, then `Enter`
4. Press `Tab` to return to content area
5. Use `j/k` to navigate through commits
6. Press `Enter` to view commit details
7. Press `Esc` to close details and return to list

### Quick View Switching (Power User Method)
1. Press `v` to enable vim mode
2. Press `Tab` to jump to header
3. Use `Shift+H/L` to select a view (Changes, Branches, History, etc.)
4. Press `Enter` to switch to that view
5. Press `Tab` to jump back to content
6. Navigate content with `j/k` (or `h/l` for file tree)

### Staging Files for Commit
1. Press `v` to enable vim mode
2. If you're in the header, press `Tab` to switch to content
3. Use `j/k` to navigate through modified files
4. Press `Enter` to view the diff
5. Click the "Stage" button or use mouse to stage files

### Browsing Code
1. Press `v` to enable vim mode
2. Press `Tab` to ensure you're in content area
3. Use `j/k` to move through files and folders
4. Use `h` to collapse a folder, `l` to expand it
5. Press `Enter` to open a file

## Future Enhancements

Planned improvements:
- `Space` to stage/unstage files in Changes view
- `gg` to jump to top of list
- `G` to jump to bottom of list
- `/` to search within current context
- `n/N` for next/previous search result
- Custom keybinding configuration

## Troubleshooting

**Vim mode not working?**
- Make sure you're not focused in an input field or text area
- Try pressing `Esc` to reset, then `v` to re-enable

**Lost focus?**
- Click on the list/panel you want to navigate
- Or use `Shift+J/K` to return to header navigation

**Keys not responding?**
- Check that the VIM indicator is visible in the status bar
- Some keys only work in specific contexts (e.g., h/l for file tree)
