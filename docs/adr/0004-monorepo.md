# One repository for the iOS app, Go server, API contract and docs

`ios/`, `server/`, `api/` and `docs/` live in one repo, so a change to the API contract, the server and the app can land in one PR and be reviewed together. CI uses path filters so each change runs only the jobs it affects.
