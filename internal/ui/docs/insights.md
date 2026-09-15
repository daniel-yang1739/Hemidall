## Insights - Reading Your Session

* **Start here** : Insights ranks observations worth inspecting; it does not decide that a large result is wasteful.
  * **Open and navigate** : Press 5 for Insights. Use h/l or Left/Right to change category, j/k or Up/Down to select a row, and Enter to inspect its linked step in History.
  * **Find this guide** : Press 4 for Docs, then / and type Insights. Enter finishes the search; j/k scrolls the guide.

* **Choose the question** : Each category answers a different question. Rows are individual outputs or generations, not totals grouped by tool or model.
  * **Outputs** : Which local tool results contain the most text? TOKENS is a local token estimate using cl100k_base, with a byte-length fallback if unavailable. A large result is not proof that all of it was sent to the model or billed.
  * **Context growth** : Between comparable generations, where did recorded context increase most? GROWTH is a positive difference, not context-window utilization. Model or source changes, missing measurements, numeric generation gaps and checkpoints prevent comparisons.
  * **Tokens** : Which generations have the largest recorded input plus available output? Input usage must be available; missing output components are excluded. This is not a ranking of visible response length.
  * **Cost** : Which generations have the highest reference cost in USD? Calculations use model catalog rates and the pricing-provider assumption. This is not an invoice, subscription charge or guaranteed saving.

* **Read the chart** : RELATIVE compares each value with the largest value in the current category. A full bar means largest here, not 100% of the session, context capacity or budget.
  * **Example** : If outputs contain 5,000 and 2,500 estimated tokens, the first bar is full and the second is half-length. It does not mean the first output wasted 5,000 tokens.
  * **Changing categories** : Each category has its own scale; equal bar lengths in Outputs and Cost do not represent equal quantities. New data can rescale the bars.
  * **Small values and zeros** : Partial blocks keep small positive values visible. Read the number for precision; zero is different from unavailable.

* **Read the row and evidence** : STEP is the transcript step to inspect. The highlighted row drives SELECTED EVIDENCE on the right in wide windows or below the ranking in narrow windows.
  * **Source** : The filename and locator identify the supporting transcript step or database record. Context growth evidence also identifies the generation pair in the wide layout.
  * **Enter** : Opens the linked History step and clears History filters so the evidence is visible. An unlinked or missing step produces a message instead of jumping to an unrelated event.
  * **Top N / total** : Insights exposes up to ten ranked items. A short terminal shows fewer rows at once; j/k scrolls that list. JSON export includes the full rankings.

* **Check coverage before conclusions** : The top strip summarizes the session, not just the selected row or the visible top ten. Input totals accumulate generation usage; they are not the current context-window size.
  * **Usage x/y turns** : x generations have input usage available out of y analyzed generations. This does not promise that all output fields are present.
  * **Priced x/y** : x generations have enough input usage and recognized reference pricing to estimate a cost. Missing output can still make that estimate partial.
  * **Asterisk and Partial estimate** : Some generation pricing or usage is missing. Treat the displayed cost as incomplete; do not compare it as though coverage were complete.
  * **Ready and last-read time** : These describe source monitoring, not whether the agent is thinking. Read interrupted retains the last successful data while retrying. No comparable observations means insufficient eligible data, not proof of no growth or cost.

* **A practical inspection loop** : Start with Outputs, select a large result, and press Enter. Decide whether its content was useful before changing the tool or prompt.
  * **Cross-check** : Inspect nearby Context growth, Tokens and Cost entries. Temporal proximity suggests where to investigate; it does not prove that one tool result caused the increase.
  * **Save the evidence** : In Insights, type :report and press Enter. A new JSON file in the current directory preserves the displayed revision, full rankings, measurement methods and pricing assumptions.
  * **Privacy** : The report omits raw prompts and tool output text, but includes session identifiers, model/tool names and evidence locations. Review it before sharing.
