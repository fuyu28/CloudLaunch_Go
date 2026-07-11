# Build Directory

The build directory is used to house all the build files and assets for your application. 

The structure is:

* bin - Output directory
* darwin - macOS specific files
* windows - Windows specific files

## Mac

The `darwin` directory holds files specific to Mac builds.
These may be customised and used as part of the build. To return these files to the default state, simply delete them
and
build with `wails build`.

The directory contains the following files:

- `Info.plist` - the main plist file used for Mac builds. It is used when building using `wails build`.
- `Info.dev.plist` - same as the main plist file but used when building using `wails dev`.

## Windows

The `windows` directory contains the manifest and rc files used when building with `wails build`.
These may be customised for your application. To return these files to the default state, simply delete them and
build with `wails build`.

- `icon.ico` - The icon used for the application. This is used when building using `wails build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer. These are used when building using `wails build`.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.

### Bundling screencap-cli.exe

The Windows installer bundles `screencap-cli.exe` — a non-interactive, WGC-based
screenshot CLI from https://github.com/fuyu28/screencap-rs.

On Windows, `wails build` and `wails dev` fetch and place the binary
automatically via the `preBuildHooks` entry in `wails.json`, which runs
`scripts/fetch-screencap-cli.ps1`. (On non-Windows hosts the hook is skipped
with "Non native build hook: Skipping.", so it does not break macOS
development.)

To fetch it manually:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/fetch-screencap-cli.ps1
```

The NSIS installer (`installer/project.nsi`) picks up
`build/windows/screencap-cli.exe` and installs it alongside the app. The fetch
script also copies the binary to `build/bin/screencap-cli.exe` so that
`wails dev` / `wails build` binaries (which live under `build/bin`) can resolve
the CLI next to the running executable — without this, the screenshot feature
fails in dev mode.

The script pins the download by SHA256 and skips re-downloading when the
existing binary already matches. To update the bundled version, change **both**
`$ScreencapVersion` and `$ScreencapSha256` in `scripts/fetch-screencap-cli.ps1`,
re-run it, and verify the binary reports the expected version:

```
> screencap-cli.exe --version
screencap-cli 0.4.0    # exit code 0
```

Note: `*.exe` is gitignored, so `screencap-cli.exe` is not committed — it is
fetched at build time.