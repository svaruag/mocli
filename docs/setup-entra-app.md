# Setup Entra App (Required)

Mocli requires your own Microsoft Entra app registration. This is a one-time setup.

## 1. Open App Registrations

- Azure portal: https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
- Entra portal: https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade

If you only have a personal Microsoft account (`@outlook.com`, `@live.com`, `@hotmail.com`), first ensure you can access an Entra tenant via Azure signup.

## 2. Create App Registration

Recommended values:

- Name: `Mocli`
- Supported account types: `Accounts in any organizational directory and personal Microsoft accounts`
- Redirect URI: add `http://localhost`

After creation, copy:

- `Application (client) ID`

## 3. Authentication Settings

In the app's `Authentication` section:

- Ensure `Mobile and desktop applications` platform is present.
- Ensure `http://localhost` is listed as redirect URI.
- Enable `Allow public client flows`.

## 4. API Permissions

Add delegated Microsoft Graph permissions:

- `User.Read`
- `Mail.Read`
- `Mail.Send`
- `Calendars.ReadWrite`
- `Tasks.ReadWrite`
- `Files.ReadWrite`

OIDC scopes are requested by Mocli during login:

- `openid`
- `profile`
- `offline_access`

If your tenant requires admin consent, grant consent before login.

## 5. Create Mocli Credentials File

Create `entra-app.json`:

```json
{
  "client_id": "YOUR-APP-CLIENT-ID",
  "tenant": "common"
}
```

Tenant values:

- `common`: mixed org/personal accounts (recommended default)
- `consumers`: personal Microsoft accounts only
- tenant GUID: single-tenant setup

## 6. Save Credentials and Authorize

```bash
mo auth credentials ./entra-app.json
mo auth add you@outlook.com --device
mo auth status
```

If you add Graph permissions later, re-consent to mint a token with new scopes:

```bash
mo auth add you@outlook.com --device --force-consent
```

## 7. Common Tenant Errors

`AADSTS50020`:
- Account not present in targeted tenant; use `common` or `consumers` for personal accounts.

`AADSTS50059`:
- Tenant context missing/mismatched; set explicit `tenant` in credentials JSON.

`AADSTS700016`:
- Wrong `client_id`; verify app ID from App registration overview.
