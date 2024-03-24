{{blue}}Usage:{{reset}}
{{if .Runnable -}} {{.UseLine}} {{- end}}
{{if and .HasAvailableSubCommands (not (eq .Name "oh-my-dot"))}}
{{red}}{{.CommandPath}} [command]
{{end -}}

{{ if .HasExample }}
{{blue}}Examples:{{reset}}
{{.Example}}
{{end -}}

{{- if .HasAvailableSubCommands}}{{ $cmds := .Commands }}

{{- if eq (len .Groups) 0}}
{{- /* If there are no groupe on the subcommand*/ -}}
{{blue}}Available Commands:{{reset}}
{{- range $cmds}}
	{{if (or .IsAvailableCommand (eq .Name "help")) -}}
		{{rpad .Name .NamePadding }} {{.Short}}
	{{end}}
{{end}}

{{else}}{{range $group := .Groups}}

{{green}}{{.Title}}{{reset}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
{{cyan}}{{rpad .NameAndAliases .NamePadding }}{{reset}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{blue}}Additional Commands:{{reset}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
{{cyan}}{{rpad .NameAndAliases .NamePadding }}{{reset}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{blue}}Flags:{{reset}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{blue}}Global Flags:{{reset}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{blue}}Additional help topics:{{reset}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{purple}}{{.CommandPath}} {{cyan}}[command] {{weird}}--help{{reset}}" for more information about a command.{{end}}