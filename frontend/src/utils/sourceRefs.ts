// 把回答里的来源引用转成指向引用来源的可点击链接（href=#src-N）。
// 兼容两种常见引用格式：片段[N]（或片段【N】）与独立的【N】。
// 同时也处理完整的 "（来源：片段[N]）" 格式，将其替换为纯数字链接（NotebookLM 风格芯片）。
// \n? 吃掉前面的换行，让芯片紧跟在句子末尾而不是另起一行
const PAREN_SOURCE_RE = /\n?（来源：片段[【[](\d+)[】\]]）|\n?（来源：【(\d+)】）/g;
const SOURCE_REF_RE = /片段[【[](\d+)[】\]]|【(\d+)】/g;

export function linkSourceRefs(content: string): string {
  // 先处理 "（来源：片段[N]）" 完整格式 → 仅保留数字链接
  content = content.replace(
    PAREN_SOURCE_RE,
    (_match, a: string | undefined, b: string | undefined) => {
      const n = a ?? b;
      return `[${n}](#src-${n})`;
    },
  );
  // 再处理剩余的 "片段[N]" 或 "【N】"
  return content.replace(SOURCE_REF_RE, (_match, a: string | undefined, b: string | undefined) => {
    const n = a ?? b;
    return `[${n}](#src-${n})`;
  });
}
