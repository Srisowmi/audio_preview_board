# Audio Preview Board

Search iTunes tracks, build a shortlist, and play 30-second previews in the browser.

| | |
|---|---|
| **Live app (frontend)** | https://audio-preview-board.vercel.app |
| **Live API (backend)** | https://sucq2bn66m6wh5cx52evqhjld40oultz.lambda-url.ap-south-1.on.aws |
| **Design document** | [docs/DESIGN.md](docs/DESIGN.md) (architecture, data model, API docs, low-level design) |

Example endpoints:

- [/api/health](https://sucq2bn66m6wh5cx52evqhjld40oultz.lambda-url.ap-south-1.on.aws/api/health)
- [/api/search?q=coldplay](https://sucq2bn66m6wh5cx52evqhjld40oultz.lambda-url.ap-south-1.on.aws/api/search?q=coldplay)
- [/api/tracks/1122782283](https://sucq2bn66m6wh5cx52evqhjld40oultz.lambda-url.ap-south-1.on.aws/api/tracks/1122782283)

> **Reviewer reminder:** both deployments stay up until the review is complete. Afterwards, run `terraform destroy` in `terraform/` to remove all AWS resources and stop charges (see [Tear down](#tear-down)).

## Repository layout

```
backend/     Go API (net/http), unit tests, Dockerfile (Lambda container image)
frontend/    React + Vite + TypeScript SPA, Vitest tests
terraform/   ECR, Lambda + Function URL, IAM, CloudWatch, GitHub OIDC role
.github/workflows/deploy.yml   CI (Go + frontend tests) and backend deploy
docs/DESIGN.md
```

## Run locally

Prerequisites: Go 1.22+, Node 20+.

**Backend**

```powershell
cd backend
go run ./cmd/api                      # http://localhost:8080
curl "http://localhost:8080/api/search?q=coldplay"
go test ./...                         # add -race on Linux/macOS/CI (needs gcc on Windows)
```

Environment variables (all optional): `PORT` (8080), `CACHE_TTL` (`1h`), `ALLOWED_ORIGIN` (`http://localhost:5173`), `ITUNES_BASE_URL`.

**Frontend** (in a second terminal)

```powershell
cd frontend
npm install
copy .env.example .env.local          # VITE_API_URL=http://localhost:8080
npm run dev                           # http://localhost:5173
npm test                              # Vitest + React Testing Library
npm run build
```

## Deploy the frontend (Vercel)

1. Import the GitHub repository in Vercel (**Add New → Project**).
2. Set **Root Directory** = `frontend`.
3. Set **Framework Preset** = `Vite` (build `npm run build`, output `dist`; both are auto-filled). No `vercel.json` is needed because the app is a single page with no client-side routes.
4. Add the environment variable `VITE_API_URL` = the backend `api_url` from `terraform output api_url`, **without a trailing slash**. Vite bakes it in at build time, so redeploy after changing it.
5. Deploy, then copy the resulting `https://….vercel.app` URL into Terraform's `allowed_origin` (see below) so the API's CORS accepts it. Vercel redeploys on every push to `main`.

## Deploy the backend (Terraform + GitHub Actions)

Prerequisites: Terraform >= 1.5, AWS CLI configured for your account (`aws configure` or SSO), region `ap-south-1`. No AWS keys are stored in the repo or in GitHub: CI authenticates with **GitHub OIDC** and assumes a least-privilege role that Terraform creates.

A container-image Lambda can't be created before its image exists, and the image is built in CI, so apply in steps:

```powershell
cd terraform
copy terraform.tfvars.example terraform.tfvars    # edit allowed_origin (and github_* IDs if you forked) if needed
terraform init

# 1. ECR repository + GitHub OIDC provider + CI role only
terraform apply -target=aws_ecr_repository.api -target=aws_iam_role_policy.github_actions
terraform output github_actions_role_arn
```

2. In GitHub: **Settings → Secrets and variables → Actions → New repository secret** `AWS_ROLE_ARN` = the role ARN above.
   (GitHub now issues OIDC tokens with an immutable-ID `sub` claim, `repo:<owner>@<owner_id>/<repo>@<repo_id>:ref:refs/heads/main`, so the role's trust policy is built from the `github_owner`, `github_owner_id`, `github_repo_name` and `github_repo_id` variables; find the IDs with `gh api repos/<owner>/<repo> --jq '.owner.id, .id'`.)
   (If your AWS account already has a GitHub OIDC provider, set `create_github_oidc_provider = false` in `terraform.tfvars` first.)
3. Push to `main`. The workflow runs the tests, builds the image, pushes `:latest` and `:<sha>` to ECR, and skips the Lambda update because the function doesn't exist yet.
4. Create the rest:

```powershell
terraform apply
terraform output api_url          # the public backend URL
```

From now on every push to `main` runs the tests, pushes a new image and calls `aws lambda update-function-code`. Pull requests to `main` only run the tests. If the `AWS_ROLE_ARN` secret isn't set yet, the deploy job skips its AWS steps with a notice instead of failing.

After deploying the frontend, set `allowed_origin = "https://<your-app>.vercel.app"` in `terraform.tfvars` and run `terraform apply` again (it only updates a Lambda environment variable).

Key variables (`terraform/variables.tf`): `aws_region`, `environment`, `app_port`, `cache_ttl`, `allowed_origin`, `github_owner(_id)`, `github_repo_name`/`github_repo_id`, `lambda_memory_mb`, `lambda_timeout_seconds`.

### Deployment order used for the live deployment

1. `terraform apply -target=aws_ecr_repository.api -target=aws_iam_role_policy.github_actions` created the ECR repository, the GitHub OIDC provider and the CI role.
2. Saved the `github_actions_role_arn` output as the GitHub secret `AWS_ROLE_ARN`.
3. Pushed to `main`: CI ran the tests, built the image and pushed `:latest` and `:<sha>` to ECR (the Lambda update was skipped because the function didn't exist yet).
4. Ran a full `terraform apply`, which created the Lambda, Function URL, log group and permissions. Took the URL from `terraform output api_url`.
5. Deployed the frontend on Vercel (Root Directory `frontend`, Framework Vite) with `VITE_API_URL` set to the backend URL, no trailing slash.
6. Once the Vercel URL was known, set `allowed_origin = "https://audio-preview-board.vercel.app"` in `terraform.tfvars` and re-applied (only a Lambda environment variable changes).
7. Redeployed the frontend in Vercel so the build picked up `VITE_API_URL`. Every later push to `main` redeploys both sides automatically.

### Tear down

Please run this after the review to avoid ongoing AWS charges:

```powershell
cd terraform
terraform destroy
```

This removes the Lambda, Function URL, ECR repository (including its images), log group and the IAM/OIDC resources. The Vercel project can be deleted separately from the Vercel dashboard.

## Architecture write-up

**Architecture.** A React SPA on Vercel talks only to my Go API, which runs as a container-image Lambda behind a Function URL (free HTTPS, no load balancer or VPC to pay for). All iTunes calls are made server-side. The Go code is a plain `net/http` handler, so the identical code runs as a normal local server or on Lambda via an adapter (selected by an environment variable); this keeps local development simple and avoids a Lambda-only code path. Responses are normalised into my own `Track` shape (renamed fields, 300px artwork, tracks without a preview dropped), and errors use a single JSON envelope with stable codes. On the frontend, state is split by owner: a debounced-search hook, a localStorage-backed shortlist hook, and a single audio-player hook exposed via context. One shared `Audio` element is what guarantees only one track plays; stale search responses are prevented with debouncing, `AbortController` and a request-id guard. Details and diagrams are in [docs/DESIGN.md](docs/DESIGN.md).

**Caching.** The iTunes API allows only about 20 requests a minute, so the backend keeps a generic in-memory TTL cache (default 1h, configurable via `CACHE_TTL`) guarded by a `sync.RWMutex`. Keys are the trimmed, lowercased query, so `Coldplay` and ` coldplay ` share an entry. Concurrent identical misses are collapsed with `singleflight`, search results also seed the per-track cache (so `/api/tracks/{id}` is usually a hit), errors are never cached, and the cache has a size cap and a background sweep. A known trade-off: on Lambda the cache lives per warm execution environment, so a cold start begins empty and separate environments don't share entries. That is acceptable at this scale and noted in "next steps".

**IaC design.** Terraform is split into `providers.tf`, `variables.tf`, `main.tf`, `github_oidc.tf` and `outputs.tf`, with region, environment, port, cache TTL, allowed origin and sizing all parameterised. IAM is least-privilege: the Lambda role can write only to its own log group, and the CI role can push only to this ECR repository and update only this function. CI uses GitHub OIDC restricted to the `main` branch of this repo, so there are no long-lived keys anywhere. CORS is handled once, in Go, so the Function URL has no CORS block (avoiding duplicate headers). Terraform ignores `image_uri` drift because CI owns image rollouts. State is local and git-ignored, as permitted by the brief.

## Notes and decisions

**GitHub OIDC `sub` claim.** The first deploy failed with "Not authorized to perform sts:AssumeRoleWithWebIdentity" even though the trust policy matched the documented `repo:<owner>/<repo>:ref:refs/heads/main` format. A temporary debug step in the workflow (which printed only the token's `sub` and `aud` claims) showed that GitHub now sends the immutable-ID format, `repo:Srisowmi@104438958/audio_preview_board@1423213683:ref:refs/heads/main`. The role's trust policy is now an exact `StringEquals` match built from the owner/repo names and IDs (Terraform variables), so it stays scoped to this repository's `main` branch, and the debug step has been removed.

## What I would do next

- **Drag-and-drop reordering** of the shortlist. The up/down buttons are the keyboard-accessible baseline; drag-and-drop would be an enhancement on top.
- **Shared cache** (DynamoDB with TTL, or ElastiCache) because the Lambda cache is per execution environment: cold starts and parallel instances each start empty and can re-hit iTunes. Alternatively move to ECS Fargate for a long-lived process.
- **Server-side shortlist with authentication**, so a shortlist follows the user across devices instead of living in one browser's localStorage.
- **End-to-end tests with Playwright** (search, shortlist, single-player behaviour, seeking), plus component tests for the player and progress bar.
- **Custom domain** for the API and the app (Route 53 + ACM, CloudFront or API Gateway in front of Lambda), which would also give a stable CORS origin.
- **Remote Terraform state** (S3 bucket with a DynamoDB lock table) and separate environments, plus a `terraform plan` step on pull requests.
- **Backend hardening:** per-IP rate limiting, a CloudWatch alarm on 5xx/throttles, and request metrics.
- **Automatic CORS wiring:** feed the frontend's domain into Terraform rather than the manual second `terraform apply`.
