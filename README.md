# echo-sense

CLI-first audio fingerprinting experiment (Shazam-style) written in Go.
<img width="1266" height="388" alt="Screenshot 2025-12-21 at 11 50 47 PM" src="https://github.com/user-attachments/assets/1a8c9ff8-abef-4afd-bbe3-1ae5034c0fe9" />

Given a Spotify track URL, `echo-sense`:

- Fetches track metadata from Spotify
- Finds a matching YouTube video
- Downloads audio (`.m4a`)
- Converts to WAV via `ffmpeg`
- Builds a spectrogram, extracts peaks, generates fingerprints
- Stores songs + fingerprints in Postgres (via GORM)

You can then run a local search against a downloaded track file.

## Status

This repo is actively evolving. The download + fingerprint generation path is implemented; match scoring / “print the best match” flow is still a work in progress (matching currently computes intermediate structures but does not output a final match result).

## Project Layout

- `server/`: Go module (the CLI lives here)
- `server/db/`: Postgres connection + GORM models
- `server/internals/recognisingAlgorithm/`: spectrogram + peak picking + fingerprinting + matching logic
- `server/internals/spotify/`: Spotify metadata + YouTube lookup + audio download pipeline
- `songs/`: created at runtime; downloaded `.m4a` files end up here
- `temporary_files/`: created at runtime

## Prerequisites

- Go (see `server/go.mod` for the configured version)
- Postgres (local or remote)
- `ffmpeg` available on your `PATH`

## Setup

1. Install dependencies

- macOS (`brew`):
  - `brew install ffmpeg postgresql@16`
- Linux (example):
  - `sudo apt-get install -y ffmpeg postgresql`

1. Create `server/.env`

`echo-sense` loads environment variables from a `.env` file at startup (`server/main.go`).

Example:

```env
# Postgres (note: current code does not use DB_PASSWORD in the DSN)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_NAME=echo_sense

# Spotify “Client Credentials” flow
SPOTIFY_CLIENT_ID=...
SPOTIFY_CLIENT_SECRET=...
```

Spotify credentials:

- Create an app in the Spotify Developer Dashboard.
- Copy the Client ID + Client Secret into `server/.env`.

1. Start Postgres

Ensure the user in `DB_USER` can connect without a password (the current DSN in `server/db/db.go` does not include `DB_PASSWORD`).

On startup, the app will:

- Connect to Postgres
- Create `DB_NAME` if it doesn’t exist
- Auto-migrate the `songs` and `audio_fingerprints` tables

## Usage

Run everything from `server/`:

```bash
cd server
go run . download "<spotify_track_url>"
```

This creates (if missing) `songs/` and downloads a file named like:

`songs/<Artist> - <Title>.m4a`

It also fingerprints the track and stores results in Postgres.

### Search (local file)

`search` looks for a file in `songs/` by the base filename (without extension):

```bash
cd server
go run . search "Artist - Title"
```

This converts the `.m4a` to WAV, generates fingerprints from the audio, and runs the matching pipeline against fingerprints stored in the database.

## Notes & Troubleshooting

- If you see `ffmpeg error: exec: "ffmpeg": executable file not found in $PATH`, install `ffmpeg` and ensure it’s on your `PATH`.
- If Postgres connection fails, double-check `DB_HOST`, `DB_PORT`, and that `DB_USER` can connect to Postgres without a password (or update `server/db/db.go` to include a password in the DSN).
- Spotify auth uses a cached token file `spotify_token.json` in the working directory; don’t commit it.

## How It Works (High Level)

1. Convert audio to a normalized WAV format (`server/internals/wavService/wav.go`)
2. Compute a spectrogram (`server/internals/recognisingAlgorithm/spectrogram.go`)
3. Extract salient peaks (`server/internals/recognisingAlgorithm/*`)
4. Turn peak pairs into compact fingerprints and store them (`server/internals/recognisingAlgorithm/fingerprinting.go`, `server/db/models/*`)
5. For a query clip, generate fingerprints and look up candidate matches in Postgres (`server/internals/recognisingAlgorithm/matchTracks.go`)
