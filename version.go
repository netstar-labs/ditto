package ditto

// PipelineVersion identifies the fingerprint pipeline: [Normalize], the shingling
// in [Featurizer], and the feature hash together. Fingerprints are only comparable
// across documents processed by the same version. Bump it on any change to that
// pipeline, and store it alongside any persisted fingerprint (e.g. a capture-time
// SimHash on a captured record) so a version mismatch is detectable rather than a
// silent comparison of incomparable hashes.
const PipelineVersion = "ditto/2"
