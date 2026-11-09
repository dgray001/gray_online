Online Games

## Current Version

v0.9m: Christmas theme

## Dev Use

Start go server

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

## Build Instructions

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

When deploying to GCP, update the version export in /backend/static/scripts/version.js

```bash
export const version = '<version>';
```

## Benchmarking Models
Build the model trainer using go build

```bash
cd model_training
go build -o fiddlesticks.live main.go
```

Then create an input yml file in an inputs folder
An example showing my standard benchmarking is given

```bash
mkdir inputs
cp standard_benchmarks.yml inputs/standard_benchmarks.yml
./fiddlesticks.live standard_benchmarks
```

Outputs will be written to the outputs folder

## Model Training

TODO: document
