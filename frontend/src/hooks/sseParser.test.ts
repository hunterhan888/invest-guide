import { describe, it, expect } from 'vitest';
import { parseFrames } from './sseParser';

describe('parseFrames', () => {
  it('空行分帧，单帧解析', () => {
    const input = 'event: message\ndata: {"delta":"a"}\n\n';
    const { events, rest } = parseFrames(input);
    expect(events).toEqual([{ event: 'message', data: '{"delta":"a"}' }]);
    expect(rest).toBe('');
  });

  it('多帧拆分，保留不完整尾部', () => {
    const input =
      'event: heartbeat\ndata: {}\n\nevent: message\ndata: {"delta":"b"}\n\nevent: done';
    const { events, rest } = parseFrames(input);
    expect(events).toHaveLength(2);
    expect(events[0]?.event).toBe('heartbeat');
    expect(events[1]?.event).toBe('message');
    expect(rest).toBe('event: done');
  });

  it('多行 data 用 \\n 拼接', () => {
    const input = 'event: message\ndata: line1\ndata: line2\n\n';
    const { events } = parseFrames(input);
    expect(events[0]?.data).toBe('line1\nline2');
  });

  it('无 event 字段默认为 message', () => {
    const input = 'data: hi\n\n';
    const { events } = parseFrames(input);
    expect(events[0]?.event).toBe('message');
  });
});
