# Research: hosting, transactional email, iOS platform facts, macOS CI runners

Status: Research note · Retrieved 2026-10-01 · Feeds open questions Q25 (staging host), Q33 (email provider), and ADR-0008 (FCM push)

Context: Splits is an iOS client, a single Go 1.26 HTTP API container, and Postgres 18. Traffic is very low (<100 Users at first), most Users are likely in India, and a public App Store launch must stay possible (Q1). Terms follow `GLOSSARY.md`.

Every fact below cites the primary source it came from. All URLs were retrieved on **2026-10-01**. "Computed" means I derived the number from published unit prices; "Unverified" means I couldn't confirm it from a primary source. The Recommendation section at the end is opinion and is kept separate from the facts.

---

## 1. Hosting: one Go container + managed Postgres

### 1.1 Summary table (facts)

Monthly USD for the smallest setup that makes sense: always-on API + smallest paid managed Postgres with backups. Prices are list prices before tax.

| Provider | Smallest sensible setup (≈ USD/mo) | Managed PG 18? | Backups / PITR | Region nearest India |
|---|---|---|---|---|
| **Fly.io** | App ≈ $4.05 (shared-cpu-1x 512 MB in `sin`, computed) + MPG Basic $38 + storage $0.28/GB ≈ **$42+** | **No**: MPG offers 16 and 17 only | Automatic backups; PITR restore into a new cluster | `sin` (Singapore). No India region listed |
| **Render** | Starter web $7 + Postgres Basic-256mb $6 + storage $0.30/GB ≈ **$13–14** (Hobby workspace $0) | **Yes** (13–18) | PITR: 3 days (Hobby), 7 days (Pro). Logical backups kept 7 days. Free PG has no backups | Singapore. No India |
| **Railway** | Hobby $5/mo, which includes $5 of usage; PG runs as a template service on usage billing ≈ **$5–10** (estimate) | Image tag `:18` exists, but PG is an **unmanaged template** | Volume backups (daily/weekly/monthly); opt-in PITR via pgBackRest, ~4-week window | Singapore. No India |
| **Cloud Run + Cloud SQL** | Cloud Run ≈ $0 inside the free tier + Cloud SQL `db-f1-micro` $0.0105/h ≈ $7.67 + 10 GB SSD ≈ $1.70 ≈ **$9–10**, **plus** custom-domain front door (see below) | **Yes** (18 is the default) | Automated backups; PITR log retention up to 7 days (Enterprise) / 35 days (Enterprise Plus) | **Mumbai `asia-south1`, Delhi `asia-south2`** |
| **Cloud Run + Neon** | Cloud Run ≈ $0 + Neon Free $0 (100 CU-h, 0.5 GB) or Launch, usage-based ($0.106/CU-h, $0.35/GB-mo; Neon cites "typical spend $15") | **Yes** (14–18) | Instant-restore history: Free up to 6 h / 1 GB of changes; Launch up to 7 days | Neon: Singapore `aws-ap-southeast-1` only (no India) |
| **AWS App Runner** | n/a | n/a | n/a | **Closed to new customers** |
| **AWS ECS Express Mode + RDS** | Fargate 0.25 vCPU/0.5 GB ≈ $9.47 + ALB ≈ $17.45 + LCU + public IPv4 $3.65 each + RDS `db.t4g.micro` ≈ $15.33 + 20 GB gp3 $2.62 ≈ **$50+** (computed, Mumbai) | **Yes** (RDS 18.x) | RDS automated backups, retention 1–35 days (PITR inside it) | **Mumbai `ap-south-1`** |
| **AWS Lightsail** | Container Nano $7 + DB Standard $15 ≈ **$22** | **No**: Lightsail PG goes up to 16 | Point-in-time restore supported | Mumbai |
| **DigitalOcean App Platform + Managed PG** | App $5 (512 MiB) + Managed PG 1 GiB $15.15 ≈ **$20** (or $5 + $7 dev database ≈ $12, see caveats) | **Yes** (Managed 14–18; dev DB 15–18) | Managed: daily backups + WAL, PITR within 7 days | **Bangalore `BLR1`** (App Platform dynamic apps + Managed PG) |

### 1.2 Fly.io

- **Compute**: Machine price = `cpu × $0.00000075/s + (RAM_GB − 0.25) × $0.00000193/s` for shared CPUs, times a regional markup. `sin` markup is 1.269. For shared-cpu-1x: 256 MB ≈ $1.94 (iad) / $2.47 (sin); 512 MB ≈ $3.19 / $4.05; 1 GB ≈ $5.70 / $7.23 per 30 days (**computed** from the constants published in the page source). https://docs.fly.io/about/pricing/ (source: https://docs.fly.io/about/pricing.md)
- Dedicated IPv4 $2/mo (shared IPv4 + Anycast IPv6 free). TLS: first 10 single-hostname certs per org free, then $0.10/mo. Egress to India $0.12/GB. https://docs.fly.io/about/pricing/
- **Managed Postgres (MPG)** plans: Basic (shared-2x, 1 GB) $38, Starter $72, Launch $282, and up. Storage $0.28/GB per 30 days. "All plans include high availability, backups, and connection pooling." Security patches and version upgrades are listed under "What's not there yet". https://docs.fly.io/mpg/overview/
- **PG version**: `fly mpg create --pg-major-version`: "Supported versions are 16 and 17. (default 16)". **No 18.** https://docs.fly.io/flyctl/cmd/fly_mpg_create.md
- **PITR**: `fly mpg restore --pitr-time` restores into a new cluster; `--backup-id` restores a snapshot. Retention window: **Unverified** (not stated in the docs I read). https://docs.fly.io/flyctl/cmd/fly_mpg_restore.md
- **Regions**: ams, arn, cdg, dfw, ewr, fra, gru, iad, jnb, lax, lhr, nrt, ord, sin, sjc, syd, yyz. MPG is available in `sin`. No Mumbai (`bom`) region is listed. https://docs.fly.io/reference/regions/
- **Custom domain/TLS**: `fly certs add example.com`, with verification via AAAA, `_acme-challenge` CNAME, or `_fly-ownership` TXT. https://docs.fly.io/networking/custom-domain/
- **Secrets**: `fly secrets set`. Values are kept in an encrypted vault, and the API servers "can only encrypt; they cannot decrypt". Secrets reach the app as env vars. https://docs.fly.io/apps/secrets/
- **GitHub Actions**: `superfly/flyctl-actions/setup-flyctl` + a `FLY_API_TOKEN` repo secret + `flyctl deploy`. https://docs.fly.io/launch/continuous-deployment-with-github-actions/

### 1.3 Render

- **Workspace**: Hobby $0/mo + compute; Pro $25/mo + compute. https://render.com/pricing
- **Web services**: Free $0 (512 MB), Starter $7 (512 MB, 0.5 CPU), Standard $25 (2 GB, 1 CPU). https://render.com/pricing
- **Postgres**: Free $0 (256 MB), Basic-256mb $6, Basic-1gb $19, 1c-2g $40. Storage $0.30/GB. PITR window: 3 days on Hobby, 7 days on Pro and above. https://render.com/pricing
- **Free tier caveats**: free web services spin down after 15 min idle (about 1 min to wake). Free Postgres expires after 30 days, has a 1 GB cap, and "Free Render Postgres databases don't support any form of backups." https://render.com/docs/free
- **Backups**: PITR Hobby "past 3 days", Pro "past 7 days". Logical backups kept 7 days. https://render.com/docs/postgresql-backups
- **PG versions**: "18, 17, 16, 15, 14, 13" fully supported. https://render.com/docs/postgresql-upgrading
- **Regions**: Oregon, Ohio, Virginia, Frankfurt, Singapore. No India. https://render.com/docs/regions
- **TLS**: automatic certs for all custom domains, including wildcards, and HTTP is redirected to HTTPS. Hobby includes 2 custom domains, then $0.25/domain/mo. https://render.com/docs/custom-domains, https://render.com/pricing
- **Secrets**: env vars, secret files, and environment groups. https://render.com/docs/configure-environment-variables
- **GitHub Actions**: deploy hooks (one HTTP POST, including "deploy a registered image") or `render deploys create --wait` from the CLI. https://render.com/docs/deploy-hooks, https://render.com/docs/cli

### 1.4 Railway

- **Plans**: Free $0 ($1 credit/mo), Hobby $5/mo (includes $5 of usage), Pro $20/mo (includes $20). Usage: RAM $10/GB-mo, CPU $20/vCPU-mo, egress $0.05/GB, volumes $0.15/GB-mo. Hobby caps volume storage at 5 GB. https://docs.railway.com/reference/pricing/plans
- **Postgres** is a template service built on the official Docker image. The docs call these templates "unmanaged, meaning you have total control over their configuration and maintenance". https://docs.railway.com/databases/postgresql
- **PG 18**: the template image repo ships `Dockerfile.13` … `Dockerfile.18`. Its README says `:latest` "currently points to PostgreSQL 16", so pin `:18`. https://github.com/railwayapp-templates/postgres-ssl (README: https://raw.githubusercontent.com/railwayapp-templates/postgres-ssl/main/README.md). Which version a new template deploy picks by default: **Unverified**.
- **Backups**: volume backups, manual or scheduled daily/weekly/monthly, restorable only into the same project and environment. https://docs.railway.com/volumes/backups
- **PITR**: opt-in. pgBackRest archives WAL to a Railway bucket, with weekly full and daily differential backups. "The last 4 full backups are retained, giving you a restore window of roughly 4 weeks." Restore creates a new service. Cost is ordinary egress plus storage. https://docs.railway.com/volumes/point-in-time-recovery.md
- **Regions**: US West, US East, EU West (Amsterdam), Southeast Asia (Singapore). No India. https://docs.railway.com/reference/regions
- **Domains/TLS**: automatic SSL for custom domains. Hobby allows 2 custom domains per service. https://docs.railway.com/networking/domains/working-with-domains.md
- **CI**: GitHub autodeploy with "Wait for CI", or `railway up --ci` from Actions. https://docs.railway.com/deployments/github-autodeploys.md, https://docs.railway.com/cli/deploying.md

### 1.5 Google Cloud Run + Cloud SQL (or Neon)

- **Cloud Run pricing** (request-based billing): free tier of 180,000 vCPU-s, 360,000 GiB-s, and 2M requests per month, aggregated per billing account. Instance-based billing has a different free tier (240,000 vCPU-s / 450,000 GiB-s). `asia-south1` (Mumbai) is a Tier 1 price region. https://cloud.google.com/run/pricing, https://cloud.google.com/run/docs/locations
- **Regions**: `asia-south1` Mumbai and `asia-south2` Delhi are both available. https://cloud.google.com/run/docs/locations
- **Custom domain caveat**: Cloud Run domain mapping is Preview, "not production-ready", and **not available in `asia-south1`/`asia-south2`** (only asia-east1, asia-northeast1, asia-southeast1, and some EU/US regions). The production options are a global external Application Load Balancer or Firebase Hosting. Firebase Hosting rewrites to Cloud Run need the Blaze plan. https://cloud.google.com/run/docs/mapping-custom-domains, https://firebase.google.com/docs/hosting/cloud-run
- **Load balancer**: global forwarding rules cost $0.025/h for the first 5, which is ≈ **$18.25/mo** (computed). https://cloud.google.com/load-balancing/pricing
- **Secrets**: Secret Manager secrets exposed as env vars (resolved at instance start) or volume mounts (fetched on read). https://cloud.google.com/run/docs/configuring/services/secrets
- **GitHub Actions**: `google-github-actions/deploy-cloudrun`, authenticated via Workload Identity Federation or a service account key. https://github.com/google-github-actions/deploy-cloudrun
- **Cloud SQL PG versions**: "PostgreSQL 18 (default)", minor 18.6. https://cloud.google.com/sql/docs/db-versions
- **Cloud SQL pricing** (default region shown on the page, likely us-central1): `db-f1-micro` $0.0105/h (0.6 GB), `db-g1-small` $0.035/h (1.7 GB). SSD $0.000232877/GiB-h (≈ $0.17/GB-mo). Backups $0.000109589/GiB-h. Shared-core types "are not covered by the Cloud SQL SLA". https://cloud.google.com/sql/pricing. **Mumbai-specific Cloud SQL rates: Unverified** (the page loads the regional table client-side).
- **Cloud SQL PITR**: log retention "Up to 35 days" (Enterprise Plus) / "Up to 7 days" (Enterprise). Shared-core machines are in the Enterprise edition. https://cloud.google.com/sql/docs/editions-intro, https://cloud.google.com/sql/docs/postgres/backup-recovery/backups
- **Neon**: supports Postgres 14, 15, 16, 17, 18 (18.6 as of Aug 2026). https://neon.com/docs/postgresql/postgres-version-policy
- **Neon plans**: Free $0 (100 CU-h per project, 0.5 GB, scale to zero after 5 min, history up to 6 h or 1 GB of changes). Launch is usage-based: $0.106/CU-h, $0.35/GB-mo, history $0.20/GB-mo, window up to 7 days. https://neon.com/pricing
- **Neon regions**: AWS us-east-1, us-east-2, us-west-2, eu-central-1, eu-west-2, ap-southeast-1 (Singapore), ap-southeast-2, sa-east-1. Azure regions are deprecated. No India. https://neon.com/docs/introduction/regions

### 1.6 AWS

- **App Runner**: "AWS App Runner is no longer open to new customers." AWS recommends **Amazon ECS Express Mode** instead, which provisions Fargate + an Application Load Balancer at no extra charge beyond those resources. https://docs.aws.amazon.com/apprunner/latest/dg/apprunner-availability-change.html
- **Mumbai unit prices** (AWS Price List API, publication 2026-10-01): RDS `db.t4g.micro` PostgreSQL Single-AZ $0.021/h; gp3 storage $0.131/GB-mo; ALB $0.0239/h + $0.008/LCU-h; public IPv4 $0.005/h; Fargate $0.04256/vCPU-h + $0.004655/GB-h. https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonRDS/current/ap-south-1/index.csv (and the AWSELB, AmazonVPC, AmazonECS equivalents)
- **RDS PG 18**: available (18.1 through 18.6 listed). https://docs.aws.amazon.com/AmazonRDS/latest/PostgreSQLReleaseNotes/postgresql-release-calendar.html
- **RDS backups**: retention can be set "between 1 and 35 days". The default is 1 day via API/CLI (the console default differs). https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithAutomatedBackups.BackupRetention.html
- **Free tier** (accounts from 15 Jul 2025 on): up to $100 in credits at sign-up plus up to $100 earned. The Free plan covers `db.t3.micro`/`db.t4g.micro` for PostgreSQL. https://aws.amazon.com/rds/free/
- **Lightsail**: container Nano $7 (0.25 vCPU, 512 MB), Micro $10. Managed DB Standard $15 (1 GB, 40 GB SSD), HA $30. Mumbai rates are the same per the Price List API. https://aws.amazon.com/lightsail/pricing/. Lightsail PostgreSQL: "PostgreSQL 12, 13, 14, 15, and 16 are available". **No 17/18.** https://docs.aws.amazon.com/lightsail/latest/userguide/amazon-lightsail-choosing-a-database.html

### 1.7 DigitalOcean

- **App Platform**: $5/mo (512 MiB, 50 GiB transfer). Dev database add-on $7/mo (512 MiB). Extra egress $0.02/GiB. https://www.digitalocean.com/pricing/app-platform
- **Regions**: dynamic apps are available in BLR (Bangalore) (table "Generated on 1 Oct 2026"). https://docs.digitalocean.com/products/app-platform/details/availability/. BLR1 = Bangalore, India. https://docs.digitalocean.com/platform/regional-availability/
- **Managed PostgreSQL**: smallest single node 1 GiB/1 vCPU $15.15/mo, storage 10–30 GiB at $0.215/GiB. https://www.digitalocean.com/pricing/managed-databases. Versions: "We currently support PostgreSQL major versions 14, 15, 16, 17, and 18 on Standard Edition clusters." https://docs.digitalocean.com/products/databases/postgresql/how-to/create/
- **Backups/PITR**: "Full cluster backups are taken daily and write-ahead-logs are maintained to allow you to restore to any point-in-time within the previous seven days." https://docs.digitalocean.com/products/databases/postgresql/details/features/ (also https://docs.digitalocean.com/products/databases/postgresql/details/limits/)
- **Dev database caveats**: one size only, default database only (no creating extra databases), region-locked to the app. Spec versions 15, 16, 17, 18. https://docs.digitalocean.com/products/app-platform/how-to/manage-databases/. Whether dev databases have backups/PITR: **Unverified** (not stated on that page).
- **TLS**: automatic certs from Let's Encrypt or Google Trust Services. If you use CAA records, allow both `letsencrypt.org` and `pki.goog`. https://docs.digitalocean.com/products/app-platform/how-to/manage-domains/
- **Secrets**: app-spec env vars with `type: SECRET` are stored encrypted (shown as `EV[...]`). Same page as above.
- **GitHub Actions**: `digitalocean/app_action/deploy@v2` with a DO API token secret. https://docs.digitalocean.com/products/app-platform/how-to/deploy-from-github-actions/

---

## 2. Transactional email

### 2.1 Summary table (facts)

| Provider | Free / low-volume pricing | Go | API | Domain auth |
|---|---|---|---|---|
| **Postmark** | Free: 100 emails/mo (testing). Basic $15/mo for 10k, extra $1.80/1k | **No official Go library**. Postmark lists 3 community Go libs (e.g. `mrz1836/postmark`, "Unofficial") | `POST https://api.postmarkapp.com/email`, header `X-Postmark-Server-Token` | DKIM TXT record (1024-bit keys) + custom Return-Path. New accounts need manual approval (<24 h weekdays); until then you can send only to your own verified domains |
| **Resend** | Free: 3,000/mo, 100/day, 3 domains, 30-day retention. Pro $20/mo for 50k, overage $0.90/1k | **Official**: `github.com/resend/resend-go` | `POST https://api.resend.com/emails`, `Authorization: Bearer re_…` | DNS records per domain. Recommends a sending subdomain, custom Return-Path, DMARC |
| **Amazon SES** | $0.10 per 1,000 (à la carte). Free-tier *credits* for new AWS accounts (up to $200), not a permanent free quota | AWS first-party SDK for Go v2 (`sesv2` package); SDK page not fetched (see §5) | SES API/SMTP; endpoint `email.ap-south-1.amazonaws.com` in Mumbai | Sandbox per region: verified recipients only, 200 msgs/24 h, 1 msg/s, until you request production access |
| **SendGrid** | **Free plan retired**: a 60-day trial at 100/day, then Essentials from $19.95/mo | Official `sendgrid/sendgrid-go` ("maintained and funded by Twilio SendGrid") | Web API v3 / SMTP | SPF/DKIM via domain authentication (not fetched) |
| **Mailgun** | Free: 100 emails/day, 1 domain, 1-day log retention. Basic $15/mo for 10k, extra $1.80/1k | Official `mailgun/mailgun-go` (active, last push 2026-09-07) | REST + SMTP | SPF + DKIM records, auto key rotation |

Sources:
- Postmark: https://postmarkapp.com/pricing, https://postmarkapp.com/developer/user-guide/send-email-with-api, https://postmarkapp.com/developer/integration/community-libraries, https://postmarkapp.com/developer/integration/official-libraries (no Go entry), https://postmarkapp.com/support/article/1091-how-do-i-set-up-dkim-for-postmark, https://postmarkapp.com/support/article/1084-how-does-the-account-approval-process-work. Postmark says transactional and broadcast messages travel on separate IP infrastructure: https://postmarkapp.com/developer
- Resend: https://resend.com/pricing, https://resend.com/docs/sdks, https://resend.com/docs/api-reference/emails/send-email, https://resend.com/docs/dashboard/domains/introduction
- SES: https://aws.amazon.com/ses/pricing/, https://docs.aws.amazon.com/ses/latest/dg/request-production-access.html, https://docs.aws.amazon.com/general/latest/gr/ses.html
- SendGrid: https://www.twilio.com/en-us/products/email-api/pricing, https://support.sendgrid.com/hc/en-us/articles/35270136965403-Twilio-SendGrid-Trial-Account-Plan, https://github.com/sendgrid/sendgrid-go
- Mailgun: https://www.mailgun.com/pricing/, https://documentation.mailgun.com/docs/mailgun/user-manual/domains/domains-verify, https://github.com/mailgun/mailgun-go

**Deliverability reputation: Unverified.** No vendor-neutral primary source exists, and the claims on vendor sites are marketing. Facts that do bear on it: Postmark separates transactional and broadcast traffic onto different IP pools and vets every account by hand. Resend offers dedicated IPs only on Scale and above. SES starts every account in a sandbox. For the low volume Splits expects, correct SPF/DKIM/DMARC on a sending subdomain matters more than which provider you pick, but that is general practice, not a cited fact.

### 2.2 Local development SMTP catcher: Mailpit

- Image `axllent/mailpit`. SMTP on **1025**, web UI on **8025**. A Compose example sets `MP_SMTP_AUTH_ACCEPT_ANY=1`, `MP_SMTP_AUTH_ALLOW_INSECURE=1`, and `MP_DATABASE=/data/mailpit.db`. https://mailpit.axllent.org/docs/install/docker/
- Features include a REST API for integration tests, an HTML compatibility check, a link check, and a spam check. https://mailpit.axllent.org/docs/
- Latest release v1.31.3 (2026-09-27). https://github.com/axllent/mailpit/releases

Implication (not a cited fact): the Go server should send through an interface with an SMTP implementation (Mailpit locally) and a provider implementation (HTTP API) for staging and production.

---

## 3. iOS Universal Links and FCM → APNs

### 3.1 apple-app-site-association (AASA) hosting

- File name `apple-app-site-association`, **no extension**, served at `https://<fully qualified domain>/.well-known/apple-app-site-association`. "You must host the file using `https://` with a valid certificate and with no redirects." https://developer.apple.com/documentation/xcode/supporting-associated-domains
- Every subdomain needs its own entitlement entry and its own AASA file. `appIDs` use the format `<Application Identifier Prefix>.<Bundle Identifier>` (Team ID prefix). Universal-link paths go in `applinks.details[].components`. Same source.
- Since iOS 14, devices fetch the AASA from **Apple's CDN**, not from your server. The CDN fetches within 24 hours, and devices check for updates about once a week. For a server that isn't publicly reachable during development, use `?mode=developer` (alternate mode) on the entitlement. Same source.
- The file must be reachable from all IP ranges and geographies, must not redirect (301/302), and must not be blocked (403/404) by access policies. Test with `sudo swcutil dl -d <domain>` / `swcutil verify`, or on device via Settings > Developer > Associated Domains Development (requires Developer Mode). https://developer.apple.com/documentation/technotes/tn3155-debugging-universal-links
- Content-Type `application/json` and a size limit: **Unverified** (neither is stated on the pages read).
- Implication (not a cited fact): the Invite Link domain (Q11) needs a real hostname with a valid TLS cert serving that path. Every staging host above can serve it from the Go server.

### 3.2 Paid Apple Developer Program, APNs auth key, FCM

- Apple's capability table lists **Associated domains** and **Push notifications** as available to Apple Developer Program (ADP) and ADEP members, and **not** to a free "Apple Developer" account. https://developer.apple.com/help/account/reference/supported-capabilities-ios
- ADP costs **99 USD per membership year** (or local currency). https://developer.apple.com/programs/whats-included/
- **APNs auth key (.p8)**: you request it in your developer account and get a 10-character Key ID and a `.p8` signing key. Tokens are ES256 JWTs carrying Key ID + Team ID and must be refreshed at least once an hour. Team-scoped keys are now restricted to either Sandbox or Production, with up to 2 per environment. Topic-specific keys are also available. https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns
- **FCM**: upload the APNs authentication key (development, production, or both; at least one required) with its Key ID (and Team ID) under Project settings > Cloud Messaging > iOS app configuration. The FCM SDK uses method swizzling by default to map the APNs token to the FCM token. https://firebase.google.com/docs/cloud-messaging/ios/certs
- Implication (not a cited fact): because the key is environment-scoped, staging (TestFlight/debug, Sandbox) and production may need separate keys in Firebase. Check this when setting up ADR-0008.

---

## 4. GitHub Actions macOS runners (Xcode 27 / iOS 26–27 simulators)

Source for labels: https://github.com/actions/runner-images (README). Per-image software lists: https://github.com/actions/runner-images/tree/main/images/macos

| Label (arm64, standard) | Image | Xcode | iOS simulator runtimes installed |
|---|---|---|---|
| `macos-latest`, `macos-26` | macOS 26.6.2, image 20260907 | 26.6 (default), 26.5, 26.4.1, 26.3, 26.2, 26.1.1, 26.0.1. **No Xcode 27** | iOS 26.2, 26.4, 26.5 |
| `macos-15` | macOS 15.7.9, image 20260907 | 16.4 (default), 16.0–16.3, 26.0.1–26.3 | iOS 18.5, 18.6, 26.0, 26.1, 26.2 |
| **`xcode-27`** (public preview) | macOS 27.0, image 20260921 | **27.0 (default)**, 27.1, 27.2 (beta) | **iOS 27.0** (SDKs for simulator 27.0–27.2 are present) |

- Xcode 27 ships **only** on the `xcode-27` preview label. Since 2026-09-16 its base OS is macOS 27. "Preview" means software can be unstable and jobs may queue. https://github.com/actions/runner-images/issues/14404, https://github.com/actions/runner-images/blob/main/images/macos/xcode-27-arm64-Readme.md
- `macos-26` arm64 software list: https://github.com/actions/runner-images/blob/main/images/macos/macos-26-arm64-Readme.md. `macos-15` arm64: https://github.com/actions/runner-images/blob/main/images/macos/macos-15-arm64-Readme.md
- Policy: one Xcode major version per macOS image, and betas only on the latest image. `-latest` labels migrate gradually over 1–2 months. Runner-images README, "Software and image support".
- **Billing**: "GitHub Actions usage is free for self-hosted runners and for public repositories that use standard GitHub-hosted runners." GitHub's standard-runner list includes `macos-latest`, `macos-15`, `macos-26`, and `xcode-27` (public preview). "Larger runners are always charged for, even when used by public repositories", which covers the `-large` / `-xlarge` labels. https://docs.github.com/en/billing/concepts/product-billing/github-actions (source: https://github.com/github/docs/blob/main/content/billing/concepts/product-billing/github-actions.md), https://docs.github.com/en/actions/reference/runners/github-hosted-runners

---

## 5. Couldn't verify (flagged)

- Fly MPG PITR retention window. Fly's rendered price table (I computed prices from the constants in the page source instead).
- Cloud SQL **Mumbai** regional rates (I quoted the page's default-region rates).
- Which Postgres version Railway's template deploys by default.
- Whether DigitalOcean App Platform dev databases have backups or PITR.
- AASA Content-Type and size limit.
- Email deliverability rankings (no neutral primary source exists).
- SendGrid and Resend SPF/DKIM record specifics (pages not fully read). AWS SDK for Go v2 SES module page (not fetched; known AWS first-party SDK).
- Latency figures (for example Mumbai↔Singapore). None were measured or cited.

---

## 6. Recommendation (opinion, not fact)

**Staging host (Q25): DigitalOcean App Platform (BLR) + DigitalOcean Managed PostgreSQL 18 (BLR), ≈ $20/mo.** It is the only option that is all of: in India, a real managed Postgres 18 with 7-day PITR, automatic TLS, encrypted app secrets, and a first-party GitHub Action, at a flat, predictable price. Use the $15.15 managed cluster, not the $7 dev database, so staging exercises the same backup/PITR path production will use. The same setup scales directly to production.

- **Runner-up for cost: Render (Singapore), ≈ $13/mo.** PG 18 and PITR (3 days on Hobby), with the simplest developer experience, but no India region.
- **Runner-up for "big cloud" credibility: Cloud Run (Mumbai) + Cloud SQL PG 18.** The compute is cheap, but custom domains in Mumbai need an ≈ $18/mo load balancer or Firebase Hosting, and IAM/Workload Identity add setup work. A good portfolio talking point, but more moving parts for a low-traffic app.
- **Rule out**: Fly MPG (no PG 18, $38 minimum, no India region), Lightsail (PG ≤ 16), App Runner (closed to new customers), and ECS Express + RDS (≈ $50+/mo for this scale). Railway is workable but its Postgres is a self-managed template, which weakens the "managed DB" story.

**Transactional email (Q33): Resend now, behind a `Mailer` interface. Keep Postmark as the upgrade path.** Resend's free tier (3,000/mo, 100/day) covers verification codes and password resets for <100 Users, and it has an official Go SDK. If deliverability problems show up, or the app outgrows the free tier, switching to Postmark ($15/mo) is a one-adapter change; its transactional-only IP pools and account vetting suit verification email. Amazon SES is the cheapest at volume but adds sandbox exit and AWS account overhead. SendGrid no longer has a free plan. **Local dev: Mailpit** in Docker Compose (SMTP 1025, UI 8025).

**iOS**: budget the **$99/yr Apple Developer Program** before M2. Universal Links and push both need it. Create the APNs `.p8` key(s) per environment and upload them to Firebase. Serve the AASA from the API's domain with no redirects.

**CI**: run iOS jobs on `macos-26` (Xcode 26.x, iOS 26 simulators). Add an optional, non-blocking `xcode-27` job if you want to test against Xcode 27 while it is in preview. Keep the repo public and avoid `-large`/`-xlarge` labels, and the macOS minutes stay free.
