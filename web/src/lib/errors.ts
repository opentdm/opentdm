// errMessage narrows an unknown thrown value (the type of a `catch` binding) to a
// displayable string. The api client throws Error, but callers must not assume so —
// see the project's TS coding-style rule: use `unknown`, then narrow.
export function errMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
