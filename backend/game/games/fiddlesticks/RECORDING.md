# Production recordings

Recording runs only when `GAE_SERVICE` and `FIDDLESTICKS_RECORDING_BUCKET` are set.
Set the bucket name without `gs://` in `backend/app.yaml` before deploying.

Create a private bucket in the same region as App Engine:

```bash
gcloud storage buckets create gs://YOUR_BUCKET --project=dwg-games --location=YOUR_REGION --uniform-bucket-level-access --public-access-prevention
gcloud storage buckets add-iam-policy-binding gs://YOUR_BUCKET --member=serviceAccount:dwg-games@appspot.gserviceaccount.com --role=roles/storage.objectCreator
```

Deploy from `backend/` using `gcloud app deploy`.
No service account key is needed.

Play a round and check `gs://YOUR_BUCKET/fiddlesticks/` for JSON objects.
Upload failures appear in application logs as `Fiddlesticks upload`.

Objects use `fiddlesticks/RECORDING_UUID/EVENT_COUNT.json`.
Each contains `recording_id` and the complete ordered `events` array.
Use the highest event count per recording when processing checkpoints.
Each event has `sequence`, UTC `time`, `kind`, and `data`.

Events cover game settings/deployment, player identities/model parameters,
deals with all hands, AI predictions, accepted actions, trick winners,
round results, disconnections/reconnections, and the final result.
Predictions contain the raw bid and sampled bid, or legal hand indices,
their unnormalized weights, and the sampled card index.
Actions distinguish `human`, `bot`, and `timeout` sources.

The HttpOnly browser cookie persists for one year, renewed on connection.
It identifies a browser profile, not an authenticated person.
Identity records include IP, user agent, nickname, and connection time.
Checkpoints upload asynchronously at each deal and on game termination.
Abrupt instance termination can lose the current round and pending uploads.
Uploads retry three times; later checkpoints include earlier events.
