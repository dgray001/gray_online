Online Games

## Current Version

v0.2ah: Drag card to play

## Dev Use

Ensure DEV flags are true

Backend: /backend/main.go::DEV

```bash
var DEV = true
```

Frontend: /frontend/src/scripts/util.ts::DEV

```bash
export const DEV = true;
```

Then start go server

```bash
cd backend
go run .
```

Then start webpack server

```bash
cd frontend
npm run dev
```

Navigate to localhost:8080

To create a new frontend component or page

```bash
cd frontend
npm run add
```

## Build instructions

Set the aforementioned frontend and backend flags to false

Build the js bundle

```bash
cd frontend
npm run build
```

Run go server

```bash
cd backend
go run .
```

If deploying to GCP

```bash
gcloud app deploy
```