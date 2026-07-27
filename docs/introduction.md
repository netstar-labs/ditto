# Meet ditto — the office that says "same as above"

The ditto mark — a pair of small quotes, `″` — is the oldest deduplication
notation there is: written under an entry, it means *repeat what's above without
writing it again*. That is precisely this library's job. Show it a stream of
documents and it tells you which ones are, for all practical purposes, the same as
one already seen — even when a word was swapped, a punctuation mark added, or a
line reflowed.

## What it actually is

ditto is a pure-Go implementation of Charikar's **SimHash**: it maps each document
to a 64-bit fingerprint with the deliberate, un-cryptographic property that
**similar inputs produce similar fingerprints**. Flip one input character and a
SHA hash changes half its bits; a ditto fingerprint changes a handful. So the
question *"are these two nearly the same?"* collapses to counting the differing
bits between two 64-bit integers — a single machine instruction — and *"which of
these millions are near-duplicates of each other?"* is answered by a banded index
that never has to compare most pairs at all.

## The capability

It needs **no tokenizer and no model**. The default featurizer shingles character
4-grams, which is language-agnostic and shrugs off light obfuscation — `c4sino`
still shares grams with `casino`. That makes ditto the right tool for the messy,
adversarial, multilingual text of a spam or crawl corpus, where a word tokenizer
would trip and an LLM tokenizer would fragment meaning. One dependency-free
library, fingerprints in microseconds, near-duplicate lookup in better than O(n²).

## The scope it keeps

ditto answers one question — *is this a near-duplicate?* — and only that. It does
not classify, categorize, or judge content; it has no opinion on whether a
document is spam, gambling, or a newsletter. Those are other tools' jobs. Its
discipline is the source of its speed and its honesty: a fingerprint is a
statement about *sameness*, nothing more.

*Read next:* [executive-summary.md](executive-summary.md) ·
[architecture.md](architecture.md) · [userguide.md](userguide.md) ·
[../example/README.md](../example/README.md)
