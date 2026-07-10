# Deployment

Every push to `main` (including merged PRs) triggers the
[Release & Deploy workflow](../.github/workflows/release-deploy.yml), which:

1. **Versions the release** using TurfBuilder's CalVer scheme:
   `YY.M.D_<build number for that day>` — e.g. `26.7.9_0`, and `26.7.9_1` for
   a second release the same day. No zero padding, matching TurfBuilder's tags.
2. **Builds the container** and pushes it to GitHub Container Registry as
   `ghcr.io/turfbuilder/overture-download-service:<version>` (and `:latest`).
3. **Tags the commit and creates a GitHub release** with generated notes.
4. **Deploys to Kubernetes**: applies [.k8s/deployment.yaml](../.k8s/deployment.yaml)
   with the image pinned to the new version and waits for the rollout to
   finish. The Deployment runs **2 replicas**; both pull work from the same
   NATS consumer, so requests are load-balanced between them and a rolling
   update never drops the service.

Runs are serialized, so two quick pushes can't race on the build number or
deploy out of order. Releasing (steps 1–3) and deploying (step 4) are separate
jobs: a failed deploy never blocks or revokes the release, and using
**Re-run failed jobs** on the run redeploys that exact version rather than
minting a new one. If the rollout doesn't become healthy within 5 minutes, the
deploy job fails — the previous ReplicaSet keeps serving.

## Required GitHub secrets

Set these under **Settings → Secrets and variables → Actions**:

| Secret | Contents |
|---|---|
| `KUBE_CONFIG` | Base64-encoded kubeconfig for the cluster (`base64 < kubeconfig` on macOS, `base64 -w0 < kubeconfig` on Linux). Use a service account scoped to the `turfbuilder` namespace if possible. |
| `S3_ENDPOINT` | Spaces/S3 endpoint hostname, e.g. `nyc3.digitaloceanspaces.com`. |
| `S3_BUCKET` | Bucket that receives result files. |
| `S3_ACCESS_KEY` | Spaces access key. |
| `S3_SECRET_KEY` | Spaces secret key. |

The four `S3_*` secrets are synced into the cluster as the
`overture-downloader-s3` Kubernetes secret on every deploy, so rotating them
in GitHub takes effect on the next push to `main`.

## Cluster assumptions

- A NATS server with JetStream reachable at `nats.nats.svc.cluster.local:4222`.
  If yours lives elsewhere, edit the `--host` argument in
  [.k8s/deployment.yaml](../.k8s/deployment.yaml).
- The REST API is exposed inside the cluster as the `overture-downloader`
  service in the `turfbuilder` namespace on port 23363 (ClusterIP — the
  endpoint is unauthenticated, so it is deliberately not exposed externally).
- JWT validation is on by default; the pods use the production JWKS URL baked
  into the binary's `--jwks` default. Add a `--jwks=...` argument in the
  manifest to point elsewhere.
