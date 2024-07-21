# youtube-creator-control
Content creators currently need to share credentials to their youtube channels in order for any employees (editors or collaborators) to make a post to the channel.
As a work around, editors instead need to send the completed video files over to the channel owner. The owner then has to download the video, and then upload it to youtube.

This application aims to simplify the collaboration process between channel owners and their editors. Owners are able to create an account and authenticate with their youtube channel.
Their editors also create an account, and the owners invite them to be a collaborator. Editors are then able to upload video files to a space where the owner can review the content and
if ready for publishing, the onwer can choose to publish the video to youtube directly instead of needing to download and reupload the file.
One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

run all make commands with clean tests
```bash
make all build
```

build the application
```bash
make build
```

run the application
```bash
make run
```

Create DB container
```bash
make docker-run
```

Shutdown DB container
```bash
make docker-down
```

live reload the application
```bash
make watch
```

run the test suite
```bash
make test
```

clean up binary from the last build
```bash
make clean
```
