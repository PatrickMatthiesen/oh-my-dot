import { $ } from "bun";
import { parseArgs } from "util";
import os from "os";
import path from "path";

const { values } = parseArgs({
    args: Bun.argv,
    options: {
      out: {
        type: 'string',
        default: './build/',
      },
    },
    strict: true,
    allowPositionals: true,
});
console.log("Building to: " + values.out);

const ref = await $`git rev-parse --short HEAD`.text()
console.log("Commit hash: " + ref);

var version = process.env.ohmydot_version
if (!version || version.trim() === "") {
    console.log("Version not set, setting to new canary version");
    const versionTag = await $`git describe --tags --abbrev=0`.text()
    const versionNumbers = versionTag.split(".")
    const patch = parseInt(versionNumbers[2]) + 1
    const newVersion = `${versionNumbers[0]}.${versionNumbers[1]}.${patch}-canary`
    version = newVersion
}
console.log("Version: " + version);

await $`go build -ldflags "-X github.com/PatrickMatthiesen/oh-my-dot/cmd.Version=${version} -X github.com/PatrickMatthiesen/oh-my-dot/cmd.CommitHash=${ref}" -o ${values.out} .`

// set the oh-my-dot_version environment variable
if (os.platform() === "win32") {
    const fullPath = path.resolve(values.out);
    const currentDebugPath = (await $`powershell -NoProfile -Command "[Environment]::GetEnvironmentVariable('OhMyDot_Debug','User')"`.text()).trim();
    if (currentDebugPath !== fullPath) {
        console.log("Setting OhMyDot_Debug environment variable to built exe path");
        const escapedFullPath = fullPath.replace(/'/g, "''");
        await $`powershell -NoProfile -Command "[Environment]::SetEnvironmentVariable('OhMyDot_Debug', '${escapedFullPath}', 'User')"`;
    }
    
    // if not in PATH, add it
    const userPath = (await $`powershell -NoProfile -Command "[Environment]::GetEnvironmentVariable('Path','User')"`.text()).trim();
    if (!userPath.includes("%OhMyDot_Debug%")) {
        console.log("Adding %OhMyDot_Debug% to User PATH (works as a reference to the OhMyDot_Debug env var)");
        const newUserPath = userPath + ';%OhMyDot_Debug%';
        const escapedNewUserPath = newUserPath.replace(/'/g, "''");
        await $`powershell -NoProfile -Command "[Environment]::SetEnvironmentVariable('Path', '${escapedNewUserPath}', 'User')"`;
    }
}

console.log("Build complete.");