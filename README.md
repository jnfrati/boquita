# Boquita 🟦🟨🟦

> Fully WIP not ready for production usage

Job orchestrator for Unikraft cloud.


## What is Boquita?

Boquita is a really simple job orchestator to be able to define a job through a manifest, and execute it based on a schedule you provide.
It can run natively on your [Unikraft cloud](https://unikraft.cloud) account or execute it on a third party service (like a VM in AWS/GCP)

## Installation

To be able to use Boquita, you'll need to deploy it somewhere, because we build images for it, you should be able to run it on any container based runtime like kubernetes, dockploy, etc.

Because the purpose of Boquita was to be able to run it as an add-on, this guide will cover on how to deploy it on your [Unikraft cloud](https://unikraft.cloud) account.

First, you'll need to have your [kraft CLI](https://unikraft.cloud/docs/cli/) installed and working, to double check that you're properly connected, run `kraft cloud quota`.

> TODO: Install script to deploy on kraft cloud

## CLI Usage

> TODO: CLI Usage

## Job Manifest

Before running into how to use the CLI, Boquita uses yaml manifests (similar to k8s) to be able to define a job. The job manifest tells Boquita:
- What you want to run
- Where you want to run it
- When you want to run it

The yaml file is a living component right now but it should look something like this:

```yaml
version: 0

name: string # Unique name to idenfity the cron job
manifest:
  type: "job.manifest/v1/cron" # Currently only cron supported, but future options might be "single_run" and "schedule"
  cron: string
  image: string
  entrypoint: string
  memory_mb: number
  args:
    - arg1
    - arg2
  env_map:
    - token: string
    - some_other: string
```

> TODO: Work other options like "job.manifest/v1/schedule"


## History

Boquita was born after realizing that the current offering the Unikraft was not offering any kind of "schedule a job" solution, where an instance is executed only to perform some background job and die.

If you're interested in learning more on knowing what is the status, feel free to reach [nicofrati@gmail.com](emailto:nicofrati@gmail.com)