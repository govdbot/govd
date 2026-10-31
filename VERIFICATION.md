# Verification provenance

The Facebook/TikTok update was built and tested from commit `8dc1c3ac03e5c65f6b837d3bcd41fb37055b9727` in [GitHub Actions run 41242029556](https://github.com/mrjacock/govd/actions/runs/41242029556), then merged through [PR #1](https://github.com/mrjacock/govd/pull/1). Formatted source files exported from that same build are committed here; source formatting and documentation do not change the compiled application behavior.

The complete Go unit suite and targeted race checks passed, including platform parsers, URL/identity/privacy/cookie boundaries, download-header consistency and replacement of partial output when a ranged transfer falls back to an ordinary GET.

A separate bounded live matrix completed 16 cases: 13 successful downloads, one expected size-limit rejection, and two Facebook links unavailable to anonymous retrieval. The latter two are limitations and are not counted as successful downloads. Real Telegram delivery of a TikTok app share and a Facebook app share sent in one message was confirmed. Public test scopes do not imply support for every private/deleted/restricted asset or every configured group.

CI artifact archive SHA256:

```text
5533e9caa2c38154c0f18ac957090f15ebe83bf15f378d5e6d8e9adb1544530a
```

Compiled Linux amd64 application SHA256:

```text
8b6ed3b6f1de1d1d676de6a5585236a86a49c84974d614af53995f2d8377d258
```

The CI artifact is short-lived. The repository distributes source and does not promise a permanent binary or container release. Deployment configuration, authentication material, operational logs and host firewall rules are excluded.
