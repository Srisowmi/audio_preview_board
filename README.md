# Audio Preview Board

Search iTunes tracks, build a shortlist, and play 30-second previews in the browser.

| | |
|---|---|
| **Live app (frontend)** | `TODO: https://<your-app>.vercel.app` |
| **Live API (backend)** | `TODO: https://<id>.lambda-url.ap-south-1.on.aws` (try `/api/health`) |
| **Design document** | [docs/DESIGN.md](docs/DESIGN.md) (architecture, data model, API docs, low-level design) |

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

**Tear down** (please do this after the review to avoid charges):

```powershell
terraform destroy
```

## Architecture write-up

**Architecture.** A React SPA on Vercel talks only to my Go API, which runs as a container-image Lambda behind a Function URL (free HTTPS, no load balancer or VPC to pay for). All iTunes calls are made server-side. The Go code is a plain `net/http` handler, so the identical code runs as a normal local server or on Lambda via an adapter (selected by an environment variable); this keeps local development simple and avoids a Lambda-only code path. Responses are normalised into my own `Track` shape (renamed fields, 300px artwork, tracks without a preview dropped), and errors use a single JSON envelope with stable codes. On the frontend, state is split by owner: a debounced-search hook, a localStorage-backed shortlist hook, and a single audio-player hook exposed via context. One shared `Audio` element is what guarantees only one track plays; stale search responses are prevented with debouncing, `AbortController` and a request-id guard. Details and diagrams are in [docs/DESIGN.md](docs/DESIGN.md).

**Caching.** The iTunes API allows only about 20 requests a minute, so the backend keeps a generic in-memory TTL cache (default 1h, configurable via `CACHE_TTL`) guarded by a `sync.RWMutex`. Keys are the trimmed, lowercased query, so `Coldplay` and ` coldplay ` share an entry. Concurrent identical misses are collapsed with `singleflight`, search results also seed the per-track cache (so `/api/tracks/{id}` is usually a hit), errors are never cached, and the cache has a size cap and a background sweep. A known trade-off: on Lambda the cache lives per warm execution environment, so a cold start begins empty and separate environments don't share entries. That is acceptable at this scale and noted in "next steps".

**IaC design.** Terraform is split into `providers.tf`, `variables.tf`, `main.tf`, `github_oidc.tf` and `outputs.tf`, with region, environment, port, cache TTL, allowed origin and sizing all parameterised. IAM is least-privilege: the Lambda role can write only to its own log group, and the CI role can push only to this ECR repository and update only this function. CI uses GitHub OIDC restricted to the `main` branch of this repo, so there are no long-lived keys anywhere. CORS is handled once, in Go, so the Function URL has no CORS block (avoiding duplicate headers). Terraform ignores `image_uri` drift because CI owns image rollouts. State is local and git-ignored, as permitted by the brief.

## What I would do next

- **Shared cache across Lambda instances** (e.g. DynamoDB with TTL, or ElastiCache) so cold starts and parallel instances don't re-hit iTunes; alternatively move to ECS Fargate for a long-lived process.
- **Drag-and-drop reordering** of the shortlist (the up/down buttons are the keyboard-accessible baseline; drag-and-drop would be an enhancement on top).
- **More frontend tests:** component tests for the player and progress bar (seek by keyboard and pointer), and an end-to-end test with Playwright.
- **Terraform remote state** (S3 + locking) and separate environments; a Terraform plan step on pull requests.
- **Backend hardening:** per-IP rate limiting, a CloudWatch alarm on 5xx/throttles, and request metrics.
- **Automatic frontend/CORS wiring:** manage the frontend's domain through a Terraform variable fed by the Vercel project, rather than the manual second `terraform apply`.
- **Pause on tab change / media session controls** (lock-screen play/pause) for the player.
