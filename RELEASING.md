# Releasing

Releases are built and signed only by the tag-triggered GitHub Actions workflow in
`.github/workflows/publish-release.yml`. Do not build or upload provider packages
manually, and never move or reuse a published version tag.

1. Merge a fully reviewed, green pull request to `main` and record the exact merge
   commit.
2. Run the **Release Doctor** workflow on `main` and require it to pass at that
   exact commit.
3. Create a new annotated, `v`-prefixed semantic-version tag at the exact commit,
   verify that the tag peels to it, and push only that tag.
4. Require the **Publish Release** workflow to complete successfully. It builds
   Linux AMD64 and ARM64 archives, generates deterministic checksums and the
   Terraform Registry manifest, signs the checksums with the registered GPG key,
   verifies every artifact, and creates the GitHub Release.
5. Verify the release has exactly the two platform archives, manifest,
   `SHA256SUMS`, and detached `SHA256SUMS.sig`; independently verify the
   checksums and signing fingerprint.
6. Confirm the public Terraform Registry lists the new version and both Linux
   platforms before updating any consuming repository.

If a workflow attempt fails, rerun only the failed jobs when the tagged workflow
is correct. If the workflow itself must change, merge that fix and publish a new
version from a new tag; do not retarget the failed tag.
