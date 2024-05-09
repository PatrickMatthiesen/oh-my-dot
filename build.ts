import { $ } from "bun";
import { parseArgs } from "util";

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
