# govd — public media and group hardening

Independent review snapshot based on [govdbot/govd](https://github.com/govdbot/govd), upstream commit `c49232db414f2a1132c505b0e5833813beaed3ea` (March 2026). Original MIT license and attribution are preserved. These changes have not been merged or endorsed by the upstream maintainers.

This version adds bounded anonymous Instagram/Facebook extraction and native TikTok mobile/embed parsing, along with task ownership, scoped cookies, concurrency limits and streamed download limits. The tested public Instagram, Facebook and TikTok cases do not require a personal account. TikTok uses only the temporary anonymous cookies issued by its own public page, kept in memory with their domain scopes.

## Build the reviewed version

For an explicit Compose deployment of this fork, see [installation and migration instructions in Italian](INSTALL-IT.md) and `compose.review.yaml`. This new deployment example has not yet been tested end to end on a fresh host; it does not change an existing installation automatically.

The reviewed Dockerfile is **Dockerfile.group**, target **runtime**. The original Dockerfile and Compose files are retained for reference and do not automatically deploy this reviewed image. In particular, the original Compose image still points to upstream.

Place the official [yt-dlp 2026.10.26 musllinux archive](https://github.com/yt-dlp/yt-dlp/releases/download/2026.10.26/yt-dlp_musllinux.zip) in the repository root as `yt-dlp_musllinux.zip`. Verify its SHA256 before building:

```text
c3a67f7c14d4cbf876e7801039237fc16b929a629c72f9c785ec3fdc8aa40748
```

```sh
docker build -f Dockerfile.group --target runtime -t govd:review .
```

The tested helper is Linux amd64. Other architectures have not been validated. Use the resulting local image in your own Compose deployment, preserving your database and configuring your bot token, administrator IDs and allowed chats privately. The inherited `.env.example` is an upstream example, not the reviewed production configuration; do not copy its example credentials or resource settings blindly.

[Deployment notes and limits (Italian)](REVIEW-IT.md) describe the tested resource settings, migration precautions and separate network isolation. No personal deployment files or host firewall installer are included. The Go request checks do not cover every connection made by the external yt-dlp helper; constrain that process at the container/network level too.

## Validation

- Full Go test suite passed using Go 1.26.10; targeted race tests passed.
- Live Instagram matrix: four Reels, a three-photo album and a share link downloaded; one oversized video was correctly rejected. Temporary-file cleanup was checked.
- Facebook/TikTok matrix: 13 successful download cases, covering six distinct Facebook videos, two distinct TikTok videos, an Instagram Reel and a three-photo album; one oversized file was correctly rejected. Actual app shares, mobile Reel/watch URLs, video.php, TikTok short redirects, empty handles and embed URLs were covered. Two additional Facebook links could not be retrieved anonymously and are not counted as downloads.
- Real Telegram delivery checked in a private test chat, including a 64-second TikTok and a 27-second Facebook share sent together. Bounded sequential fallback, header consistency and temporary-file cleanup are covered by tests.
- Updated govulncheck: no reachable vulnerable symbols or additional imported vulnerable packages; two module-only AWS S3 Crypto advisories remain in unused transitive functionality.

These are bounded tests, not a guarantee for every platform, private/deleted/restricted content, or every group configuration. This snapshot has not yet passed the upstream maintainer's full contribution/lint review. Inherited publishing and notification workflows are manual-only; the public-media test workflow runs for relevant pull requests and can be started manually.

For upstream documentation see [README.upstream.md](README.upstream.md). Please do not post bot tokens, cookies, personal configuration or operational logs in issues. This repository currently provides source code; it does not publish a prebuilt container image.

## Optional extractor configuration

The public source intentionally contains no embedded Twitter web-client bearer value or iGram signing key. Set `TWITTER_WEB_BEARER_TOKEN` only if you use the legacy Twitter extractor, and `IGRAM_HMAC_KEY` only if you use the optional iGram fallback. Obtain any required values independently and keep them in your private configuration. Without them, those legacy paths are unavailable; the tested anonymous Instagram yt-dlp fallback does not need either value. Example bot tokens and database/password defaults have also been removed; configure your own before startup.

## Anonymous access limitations

A video playable in a signed-in browser may still be unavailable to an anonymous downloader. Removed/restricted content and platform anti-automation responses are not treated as a guarantee of public access. No browser session is imported automatically, and no promise is made that every Facebook or TikTok link will work. See [verification provenance](VERIFICATION.md) for the tested build.
