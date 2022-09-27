# Slack Queue

A simple, per-channel queueing app for Slack.

## Quickstart

1. Create a new Slack app at https://api.slack.com/apps
2. Add a bot user to the app
3. Set the env vars `SLACK_CLIENT_ID`, `SLACK_CLIENT_SECRET`, and `SLACK_SIGNING_SECRET` to the values from the app's Basic Information page
4. Set the env var `SLACK_BOT_TOKEN` to the bot user's OAuth token. This only needs to happen for testing / if not publicly distributing the app.
5. Set the env var `FINAL_OAUTH_URL` to any url. This is used to redirect the user to after the OAuth flow is complete (It is _not_ part of the auth flow itself).
6. Run the service, either via `go run` or or through Docker.

### Running on Google Cloud Platform

1. Set up Firestore in Native Mode.
2. Enable the Cloud Build API.
3. `gcloud builds submit --tag gcr.io/[PROJECT_ID]/slack-queue`
4. Create a Cloud Build service.
   1. Set the env vars similar to the Quickstart section.
   2. Set the `GOOGLE_PROJECT_ID` env var to match your project ID.
