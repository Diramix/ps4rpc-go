package bot

import "github.com/bwmarrin/discordgo"

func commandDefs() []*discordgo.ApplicationCommand {

	integrations := &[]discordgo.ApplicationIntegrationType{
		discordgo.ApplicationIntegrationGuildInstall,
		discordgo.ApplicationIntegrationUserInstall,
	}
	contexts := &[]discordgo.InteractionContextType{
		discordgo.InteractionContextGuild,
		discordgo.InteractionContextBotDM,
		discordgo.InteractionContextPrivateChannel,
	}
	sortChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "By size", Value: "size"},
		{Name: "Recent", Value: "recent"},
		{Name: "By name", Value: "name"},
	}
	filterChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "All", Value: "all"},
		{Name: "Unlocked", Value: "unlocked"},
		{Name: "Locked", Value: "locked"},
	}

	cmd := func(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommand {
		return &discordgo.ApplicationCommand{
			Name:             name,
			Description:      desc,
			IntegrationTypes: integrations,
			Contexts:         contexts,
			Options:          opts,
		}
	}

	return []*discordgo.ApplicationCommand{
		cmd("info", "Current console status"),
		cmd("library", "Installed games",
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "sort", Description: "Sort order", Choices: sortChoices}),
		cmd("game", "Game card",
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Game name", Required: true, Autocomplete: true}),
		cmd("recent", "Recently played games"),
		cmd("shot", "Latest screenshot"),
		cmd("trophy", "Trophies for a chosen game",
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "game", Description: "Game name", Required: true, Autocomplete: true},
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "filter", Description: "Trophy filter", Choices: filterChoices}),
	}
}
