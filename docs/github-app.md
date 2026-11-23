# Use a GitHub App as the publisher

By default, the pilot publishes with your fine-grained token, so its PRs show your account. A
GitHub App is the alternative:
- each run gets a token that lasts one hour;
- the token can reach only the fixture repo;
- PRs show `<app-name>[bot]`.

It is recommended for teams, for approvals on GitHub ([github-loop.md](github-loop.md)), and
later for cluster mode. Setup takes about 15 minutes.

## 1. Create the App

On GitHub: *Settings → Developer settings → GitHub Apps → New GitHub App*.

- **Name:** anything unique, for example `<you>-chalkeus`.
- **Homepage URL:** your fixture repo's URL.
- **Webhook:** untick *Active*. The pilot doesn't use webhooks.
- **Repository permissions:** *Contents*, *Pull requests* and *Issues* each read and write.
  Leave every other permission at *No access*. *Metadata* read-only is added automatically.
- **Where can this App be installed:** *Only on this account*.

Click *Create GitHub App*. Note the **App ID** shown at the top.

## 2. Install it on the fixture repo only

On the App's page, open *Install App* and choose your account. Pick *Only select
repositories*, then select your fixture repo. Don't select any other repo. After installing,
the browser address ends in `/installations/<number>`: that number is the **installation ID**.

## 3. Generate and store the private key

On the App's page, under *Private keys*, click *Generate a private key*, and a `.pem` file
downloads. Add the two IDs to `~/.config/chalkeus/forge.env`:

```sh
FORGE_PUBLISHER_CREDENTIAL=app
FORGE_APP_ID=<app id>
FORGE_APP_INSTALLATION_ID=<installation id>
```

Then store the key. It's kept only if the check passes:

```sh
./chalkeus forge app-key < ~/Downloads/<app-name>.<date>.private-key.pem
rm ~/Downloads/<app-name>.*.private-key.pem
```

The key is saved as `~/.config/chalkeus/github-app.pem` (mode 600). It's read only on your
machine, to sign the request for a short-lived token, and is never copied into the cluster or the
evidence.

## 4. Check it

```sh
./chalkeus forge app-token --check
```

It mints a token, tests that it can write the fixture repo (code, PRs, issues) and can't write
this project's repository, then throws the token away. Expect `OK`.

## 5. Use it

`./chalkeus forge up` and every run now use a fresh App token. To go back to the token, set
`FORGE_PUBLISHER_CREDENTIAL=token`. If both a token and a key exist, you must set
`FORGE_PUBLISHER_CREDENTIAL`; the pilot won't guess.

If something fails, see [troubleshooting](troubleshooting.md#github-app).
