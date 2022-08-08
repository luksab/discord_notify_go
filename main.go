package main

import (
	"flag"
	"fmt"
	"log"
	"os/signal"
	"time"

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
	if *BotToken == "" {
		// get bot token from environment variable
		*BotToken = os.Getenv("BOT_TOKEN")
	}
	if *GuildID == "" {
		// get guild ID from environment variable
		*GuildID = os.Getenv("GUILD_ID")
	}
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalln("Invalid bot parameters:", err)
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "best_friend",
			Description: "Modify your best friend list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a user to the best friends list to get notified when they join VC",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user-option",
							Description: "User",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a user from the best friends list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user-option",
							Description: "User",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "List best friends",
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"best_friend": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			global_options := i.ApplicationCommandData().Options
			global_optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(global_options))
			for _, opt := range global_options {
				log.Println("option:", opt.Name, "type:", opt.Type, "value:", opt.Options)
				global_optionMap[opt.Name] = opt
			}

			options := global_optionMap[global_options[0].Name].Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				log.Println("option:", opt.Name, "type:", opt.Type, "value:", opt.Options)
				optionMap[opt.Name] = opt
			}

			// As you can see, names of subcommands (nested, top-level)
			// and subcommand groups are provided through the arguments.
			switch global_options[0].Name {
			case "add":
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
					if err.Error != nil {
						log.Println("Error creating best friend:", err.Error.Error())
						if err.Error.Error() == "constraint failed: UNIQUE constraint failed: best_friends.user_uuid, best_friends.friend_uuid (1555)" {
							var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: "You already have that user in your best friends list.",
								},
							})
						} else {
							var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: "Error creating best friend",
								},
							})
						}
						return
					}
					bestFriendId = opt.UserValue(nil).ID
				}

				var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Addded best friend",
								Description: fmt.Sprintf(
									"Added <@%s> to your best friends list.",
									bestFriendId,
								),
							},
						},
					},
				})
			case "remove":
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
					bestFriendId = opt.UserValue(nil).ID
					bestFriend := &BestFriend{
						UserUuid:   userId,
						FriendUuid: bestFriendId,
					}

					err := db.Delete(bestFriend)
					if err.Error != nil {
						log.Println("Error removing best friend:", err.Error)
						var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "Error removing best friend",
							},
						})
						return
					}
					if err.RowsAffected == 0 {
						var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "You don't have that user in your best friends list.",
							},
						})
						return
					}
				}

				var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Removed best friend",
								Description: fmt.Sprintf(
									"Removed <@%s> from your best friends list :(",
									bestFriendId,
								),
							},
						},
					},
				})
			case "list":
				var userId string
				if i.User != nil {
					userId = i.User.ID
				} else if i.Member != nil {
					userId = i.Member.User.ID
				} else {
					log.Println("No user found???")
				}

				var bestFriends []*BestFriend
				err := db.Find(&bestFriends, BestFriend{UserUuid: userId}).Error
				if err != nil {
					log.Println("Error getting best friends:", err)
					var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Error getting best friends",
						},
					})
					return
				}
				if len(bestFriends) == 0 {
					var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You don't have any best friends. :'(",
						},
					})
					return
				}
				var bestFriendsString string = "Your best friends are:\n"
				for _, bestFriend := range bestFriends {
					bestFriendsString += fmt.Sprintf("<@%s>\n", bestFriend.FriendUuid)
				}
				var _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Your best friends",
								Description: bestFriendsString,
							},
						},
					},
				})
			}
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
		log.Fatalln("Cannot open the session:", err)
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
		if v.BeforeUpdate != nil && v.BeforeUpdate.ChannelID != "" {
			// user was already in a voice channel
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
		channel, err := s.State.Channel(v.ChannelID)
		if err != nil {
			log.Printf("failed to get channel: %v", err)
			return
		}
		// get guild
		guild, err := s.State.Guild(v.GuildID)
		if err != nil {
			log.Printf("failed to get guild: %v", err)
			return
		}
		time.Sleep(time.Second * 60)
		// check again, if user is in the same VC, then send dm
		var userIsInVC bool
		for _, member := range guild.VoiceStates {
			if member.UserID == user.ID {
				if member.ChannelID == v.ChannelID {
					userIsInVC = true
				}
				break
			}
		}
		if !userIsInVC {
			return
		}
		log.Printf("%v is in voice channel %v", user.Username, channel.Name)

		// get best friends
		var bestFriends []BestFriend
		if result := db.Find(&bestFriends, BestFriend{FriendUuid: v.UserID}); result.Error != nil {
			log.Fatal(result.Error)
		}
		// send dm to best friends
	outer:
		for _, bestFriend := range bestFriends {
			bestFriendUser, err := s.User(bestFriend.UserUuid)
			if err != nil {
				log.Printf("failed to get user: %v", err)
				continue
			}
			// check if best friend is in a VC already
			for _, guild := range s.State.Guilds {
				for _, vs := range guild.VoiceStates {
					if vs.UserID == bestFriendUser.ID {
						continue outer
					}
				}
			}
			_, err = s.GuildMember(guild.ID, bestFriendUser.ID)
			if err != nil {
				log.Printf("best friend %v is not in the guild %v", bestFriendUser.Username, guild.Name)
				continue
			}

			// create dm
			dmChannel, err := s.UserChannelCreate(bestFriendUser.ID)
			if err != nil {
				log.Printf("failed to create channel: %v", err)
				return
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
				return
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
