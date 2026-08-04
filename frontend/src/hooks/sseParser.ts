export type RawSSEEvent = { event: string; data: string; id?: string };

export function parseFrames(buffer: string): { events: RawSSEEvent[]; rest: string } {
  const events: RawSSEEvent[] = [];
  const sep = '\n\n';
  let rest = buffer;

  while (true) {
    const idx = rest.indexOf(sep);
    if (idx === -1) break;
    const raw = rest.slice(0, idx);
    rest = rest.slice(idx + sep.length);

    const lines = raw.split('\n');
    let event = 'message';
    const dataLines: string[] = [];
    let id: string | undefined;
    for (const line of lines) {
      if (line.startsWith('event:')) event = line.slice(6).trim();
      else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
      else if (line.startsWith('id:')) id = line.slice(3).trim();
    }
    if (lines.length === 0 || (dataLines.length === 0 && !id)) continue;
    events.push({ event, data: dataLines.join('\n'), id });
  }
  return { events, rest };
}
