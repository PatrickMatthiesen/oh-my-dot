Usage:
{{if .Runnable -}} {{.UseLine}} {{- end}}
{{if and .HasAvailableSubCommands (not (eq .Name "oh-my-dot"))}}
{{.CommandPath}} [command]
{{end -}}

{{ if .HasExample }}
{{blue}}Examples:{{reset}}
{{.Example}}
{{end -}}

{{- if .HasAvailableSubCommands}}{{ $cmds := .Commands }}

{{- if eq (len .Groups) 0}}
{{- /* If there are no groupe on the subcommand*/ -}}
Available Commands:
{{- range $cmds}}
	{{if (or .IsAvailableCommand (eq .Name "help")) -}}
		{{rpad .Name .NamePadding }} {{.Short}}
	{{end}}
{{end}}

{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .NameAndAliases .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .NameAndAliases .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}