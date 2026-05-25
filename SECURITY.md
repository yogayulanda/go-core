# Security Policy

`go-core` is an open source engineering library, not a hosted SaaS product or managed runtime service. Security reports should focus on repository code, examples, documentation, dependency behavior, and unsafe defaults that could affect consuming services.

## Reporting a Vulnerability

Please report suspected vulnerabilities privately through GitHub Security Advisories for this repository when available. If advisories are not available, contact the repository maintainer through a private channel and include only the minimum technical detail needed to reproduce the issue.

Do not open a public issue for active vulnerabilities, leaked secrets, or exploit details before maintainers have had a reasonable chance to investigate.

## What to Include

- affected package, file, or behavior
- reproduction steps or proof of concept using synthetic data
- expected impact
- affected versions or commits if known
- suggested mitigation if available

## Operational Security Expectations

- Do not include production credentials, private keys, customer data, internal URLs, or proprietary infrastructure details in reports, issues, PRs, examples, or tests.
- Redact tokens, passwords, DSNs, certificates, and authorization headers.
- Use local or synthetic fixtures when demonstrating security behavior.
- Rotate any real secret that may have been committed, logged, or shared publicly.

Security fixes should preserve transport-safe error responses and keep sensitive details in protected logs or traces only.
