package main

import (
	"flag"
	"fmt"
	"log"
	"os/signal"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"os"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session

type User struct {
	Uuid  string `gorm:"primaryKey"`
	Token string `gorm:"unique"`
}

type BestFriend struct {
	UserUuid   string `gorm:"primaryKey"`
	FriendUuid string `gorm:"primaryKey"`
}

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "ping",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Basic command",
		},
		{
			Name: "add",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Basic command",
		},
		{
			Name:        "add",
			Description: "Add a user to the best friends list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user-option",
					Description: "User",
					Required:    true,
				},
				// {
				// 	Type:        discordgo.ApplicationCommandOptionRole,
				// 	Name:        "role-option",
				// 	Description: "Role option",
				// 	Required:    false,
				// },

				// Required options must be listed first since optional parameters
				// always come after when they're used.
				// The same concept applies to Discord's Slash-commands API

				// {
				// 	Type:        discordgo.ApplicationCommandOptionChannel,
				// 	Name:        "channel-option",
				// 	Description: "Channel option",
				// 	// Channel type mask
				// 	ChannelTypes: []discordgo.ChannelType{
				// 		// discordgo.ChannelTypeGuildText,
				// 		discordgo.ChannelTypeGuildVoice,
				// 	},
				// 	Required: false,
				// },
			},
		},
		{
			Name:        "remove",
			Description: "Remove a user from the best friends list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user-option",
					Description: "User",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"add": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			// Get the value from the option map.
			// When the option exists, ok = true
			// if option, ok := optionMap["string-option"]; ok {
			// 	// Option values must be type asserted from interface{}.
			// 	// Discordgo provides utility functions to make this simple.
			// 	margs = append(margs, option.StringValue())
			// 	msgformat += "> string-option: %s\n"
			// }

			// if opt, ok := optionMap["channel-option"]; ok {
			// 	margs = append(margs, opt.ChannelValue(nil).ID)
			// 	msgformat += "> channel-option: <#%s>\n"
			// }

			var bestFriendId string

			if opt, ok := optionMap["user-option"]; ok {
				// get creator id
				var userId string
				if i.User != nil {
					userId = i.User.ID
				} else if i.Member != nil {
					userId = i.Member.User.ID
				} else {
					log.Println("No user found???")
				}
				bestFriend := &BestFriend{
					UserUuid:   userId,
					FriendUuid: opt.UserValue(nil).ID,
				}
				err := db.Create(bestFriend)
				if err != nil {
					log.Printf("Error creating best friend: %v", err)
					s.ChannelMessageSend(i.ChannelID, "Error creating best friend")
				}
				bestFriendId = opt.UserValue(nil).ID
			}

			// if opt, ok := optionMap["role-option"]; ok {
			// 	margs = append(margs, opt.RoleValue(nil, "").ID)
			// 	msgformat += "> role-option: <@&%s>\n"
			// }

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"Added <@%s> to your best friends list",
						bestFriendId,
					),
				},
			})
		},
		"remove": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			var bestFriendId string

			if opt, ok := optionMap["user-option"]; ok {
				// get creator id
				var userId string
				if i.User != nil {
					userId = i.User.ID
				} else if i.Member != nil {
					userId = i.Member.User.ID
				} else {
					log.Println("No user found???")
				}
				bestFriend := &BestFriend{
					UserUuid:   userId,
					FriendUuid: opt.UserValue(nil).ID,
				}
				err := db.Delete(bestFriend)
				if err != nil {
					log.Printf("Error removing best friend: %v", err)
					s.ChannelMessageSend(i.ChannelID, "Error removing best friend")
				}
				bestFriendId = opt.UserValue(nil).ID
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"Removed <@%s> from your best friends list",
						bestFriendId,
					),
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

var db *gorm.DB

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal(err)
		panic("failed to migrate schema")
	}
	err = db.AutoMigrate(&BestFriend{})
	if err != nil {
		log.Fatal(err)
		panic("failed to migrate schema")
	}

	// Discord stuff

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		log.Println("v:", v)
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	// react to voice changes
	s.AddHandler(func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
		if v.ChannelID == "" {
			return
		}
		log.Printf("%v joined voice channel %v", v.UserID, v.ChannelID)
		// get user name
		user, err := s.User(v.UserID)
		if err != nil {
			log.Printf("failed to get user: %v", err)
			return
		}
		// get channel name
		channel, err := s.Channel(v.ChannelID)
		if err != nil {
			log.Printf("failed to get channel: %v", err)
			return
		}
		// get guild name
		guild, err := s.Guild(channel.GuildID)
		if err != nil {
			log.Printf("failed to get guild: %v", err)
			return
		}
		// get best friends
		var bestFriends []BestFriend
		if result := db.Find(&bestFriends, BestFriend{FriendUuid: v.UserID}); result.Error != nil {
			log.Fatal(result.Error)
		}
		// send dm to best friends
		for _, bestFriend := range bestFriends {
			bestFriendUser, err := s.User(bestFriend.UserUuid)
			if err != nil {
				log.Printf("failed to get user: %v", err)
				continue
			}
			// create dm
			dmChannel, err := s.UserChannelCreate(bestFriendUser.ID)
			if err != nil {
				log.Printf("failed to create channel: %v", err)
				continue
			}
			// get channel url
			channelURL := fmt.Sprintf("https://discordapp.com/channels/%v/%v", guild.ID, channel.ID)
			// send embed dm
			msg, err := s.ChannelMessageSendEmbed(dmChannel.ID, &discordgo.MessageEmbed{
				Title: guild.Name,
				Description: fmt.Sprintf(
					"%v joined %v",
					user.Username,
					channel.Name,
				),
				Author: &discordgo.MessageEmbedAuthor{
					Name:    user.Username,
					IconURL: user.AvatarURL(""),
				},
				URL: channelURL,
			})
			if err != nil {
				log.Printf("failed to send message: %v", msg)
				log.Printf("Because: %v", err)
				continue
			}
		}

	})

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		if err != nil {
			log.Fatalf("Could not fetch registered commands: %v", err)
		}

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}