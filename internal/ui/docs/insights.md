## Insights - Reading Your Session

* **Start here** : Insights ranks observations worth inspecting; it does not decide that a large result is wasteful.
  * **Open and navigate** : Press 5 for Insights. Use h/l or Left/Right to change category, j/k or Up/Down to select a row, and Enter to inspect its linked step in History.
  * **Find this guide** : Press 4 for Docs, then / and type Insights. Enter finishes the search; j/k scrolls the guide.

* **Choose the question** : Task rows span one user step through the step before the next user step. A task may contain several model calls, tool calls and compactions.
  * **Task usage** : Which user tasks made the model process the most tokens? Every observed model call contributes cached input, uncached input, thinking and content output. Repeated context is counted on every call because this is processing volume, not unique context size.
  * **Task output** : Which user tasks accumulated the most visible model-output tokens across all observed model calls?
  * **Task thinking** : Which user tasks accumulated the most observed thinking-output tokens? More thinking can indicate difficulty, exploration, confusion or retries; it does not establish quality.
  * **Tool outputs** : Which local tool results contain the most text? TOKENS is a local estimate using cl100k_base, with a byte-length fallback if unavailable. It excludes model thinking and visible model output.
  * **Compaction** : A checkpoint does not start a new user task. Task details show its count and the starting, peak and ending observed context states. Those states are not subtracted into a net-growth claim.

* **Read the chart** : RELATIVE compares each value with the largest value in the current category. A full bar means largest here, not 100% of the session, context capacity or budget.
  * **Example** : If values are 5,000 and 2,500 tokens, the first bar is full and the second is half-length. It does not mean the first item wasted 5,000 tokens.
  * **Changing categories** : Each category has its own scale; equal bar lengths in different categories do not represent equal quantities. New data can rescale the bars.
  * **Small values and zeros** : Partial blocks keep small positive values visible. Read the number for precision; zero is different from unavailable.

* **Read the row and evidence** : USER STEP identifies a task's opening prompt; STEPS counts persisted history records in the task. A Tool output is one step. The highlighted row drives SELECTED EVIDENCE on the right in wide windows or below the ranking in narrow windows.
  * **Source** : The filename and locator identify the supporting user step, tool result or generation record. Task details show the full step span, call coverage, compactions and token breakdown.
  * **Enter** : Opens the linked History step and clears History filters so the evidence is visible. An unlinked or missing step produces a message instead of jumping to an unrelated event.
  * **Top N / total** : Insights exposes up to ten ranked items. A short terminal shows fewer rows at once; j/k scrolls that list. JSON export includes the full rankings.

* **Check coverage before conclusions** : Task totals contain available observations, not invented values. Repeated model-call input is counted repeatedly because it represents processing work.
  * **Partial task** : At least one model call lacks a required usage component. Treat the displayed amount as a lower bound over available fields.
  * **Context start / peak / end** : These are recorded context states within the task. They do not attribute growth to one prompt or tool result, and a compaction can make the ending state smaller than the starting state.
  * **Usage x/y turns** : The session summary still reports generation-level input coverage. It does not promise that all split output fields are present.
  * **Ready and last-read time** : These describe source monitoring, not whether the agent is thinking. Read interrupted retains the last successful data while retrying.

* **A practical inspection loop** : Start with Task usage, open a large task, and inspect its step and model-call counts. Compare Task output with Task thinking, then inspect large Tool outputs inside the same step span.
  * **Save the evidence** : In Insights, type :report and press Enter. A new JSON file in the current directory preserves the displayed revision, full rankings and measurement methods.
  * **Privacy** : The report omits raw prompts and tool output text, but includes session identifiers, task step numbers, model/tool names and evidence locations. Review it before sharing.
