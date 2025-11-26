# contributing
contributions to govd are welcome, but to ensure a smooth collaboration, please follow these guidelines:
* fork the repository and create a new branch for your feature or bugfix.
* write clear, concise commit messages that describe your changes.
* ensure your code follows the existing coding style and conventions.
* document any new features or changes in the relevant documentation files.
* be respectful and open to feedback from maintainers and other contributors.

## local testing
if you want to test your changes locally, you should use the appropriate `docker-compose.dev.yaml` file, so that your changes are built locally, instead of using the pre-built images from docker hub.
```bash
docker compose -f docker-compose.dev.yaml up --build
```

## code quality
we use `golangci-lint` to maintain code quality. before submitting a pull request, make sure to run the linter and fix any issues reported:
```bash
golangci-lint run --build-tags=lint
```

## localization
we rely on [go-i18n](https://github.com/nicksnyder/go-i18n) package for localization. if you want to contribute translations, please follow these steps:
* if you are updating existing translations (e.g.: typos, wrong translations), you can manually edit the corresponding `active.<lang>.toml` file in the `internal/localization/locales` folder.
* if you are adding a new language or new strings, you **must** use the `goi18n` CLI tool to extract and merge translations. refer to the [go-i18n documentation](https://github.com/nicksnyder/go-i18n#command-goi18n) for more information.
> [!NOTE]
> messages are stored inside the `internal/localization/messages.go` file. you **must** edit this file to add new messages, then use the `goi18n extract` command to update the localization files.

## note
we reserve the right to reject contributions that do not align with the project's goals or standards. please make sure to test your changes thoroughly before submitting a pull request.