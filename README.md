# Recreate User Schemes

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-recreate-user-schemes?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-recreate-user-schemes/releases)

Recreate User Schemes

<details>
<summary>Description</summary>

This step recreates default user schemes.

If no shared schemes exist in the project/workspace, step will recreate default user schemes, just like Xcode does.
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `project_path` | A `.xcodeproj/.xcworkspace` path. | required | `$BITRISE_PROJECT_PATH` |
</details>

<details>
<summary>Outputs</summary>
There are no outputs defined in this step
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-recreate-user-schemes/pulls) and [issues](https://github.com/bitrise-steplib/steps-recreate-user-schemes/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
