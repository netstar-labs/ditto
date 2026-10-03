# ditto — executive summary

**What it is.** A small, dependency-free Go library that detects near-duplicate
documents. It computes a 64-bit SimHash "fingerprint" per document such that
near-identical documents land a few bits apart, and finds those neighbours across
a large corpus without comparing every pair.

**Why it exists.** Threat corpora are full of near-duplicates: one phishing
template blasted with a changed brand name, the same scam reworded across
thousands of messages, one landing page cloned across domains. Treating each as
unique wastes storage, skews any model trained on the corpus toward whatever the
most prolific actor sent, and hides the fact that ten "different" messages are one
campaign. ditto collapses that redundancy: it tells you *these are the same thing
again*.

**What you get.**
- A 64-bit fingerprint per document — compact enough to store on every record and
  compare with one instruction.
- Near-duplicate lookup and whole-corpus clustering in better than O(n²), via a
  banded index — block-partition bucketing per the pigeonhole guarantee, not a
  bit-permutation scheme.
- **No tokenizer, no model, no dependencies** — character shingles by default,
  standard library only, so it runs inline at capture as easily as offline in
  batch.
- Determinism — a fingerprint is a pure function of the (versioned) pipeline, so
  the same document always yields the same bits.

**Where it fits.** Two jobs in the detection stack: **training hygiene** — dedup a
corpus before a model learns from it, so a prolific template doesn't dominate; and
**campaign correlation** — group messages that are near-copies of each other into
families, the near-duplicate complement to shared-indicator clustering. It is
deliberately *not* a classifier: it answers sameness, not category.

**What it is not.** Not a topic or content classifier, not a semantic-similarity
model, not a plagiarism scorer with document alignment. For "what is this about?"
use a categorizer; for "is this the same as that?" use ditto.
