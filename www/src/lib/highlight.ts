const extensionToLanguage: Record<string, string> = {
  js: "javascript",
  jsx: "javascript",
  ts: "typescript",
  tsx: "typescript",
  py: "python",
  go: "go",
  java: "java",
  rb: "ruby",
  php: "php",
  rs: "rust",
  c: "c",
  h: "c",
  cpp: "cpp",
  cxx: "cpp",
  hpp: "cpp",
  cc: "cpp",
  cs: "csharp",
  sh: "bash",
  bash: "bash",
  zsh: "bash",
  json: "json",
  yml: "yaml",
  yaml: "yaml",
  toml: "toml",
  md: "markdown",
  markdown: "markdown",
  html: "html",
  xml: "xml",
  css: "css",
  sql: "sql",
};

export const getLanguageFromPath = (path: string): string | undefined => {
  const ext = path.split(".").pop()?.toLowerCase();
  return ext ? extensionToLanguage[ext] : undefined;
};
