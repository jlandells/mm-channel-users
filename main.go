package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

var Version = "development" // Default value - overwritten during bild process

var debugMode bool = false

// LogLevel is used to refer to the type of message that will be written using the logging code.
type LogLevel string

type mmConnection struct {
	mmURL    string
	mmPort   string
	mmScheme string
	mmToken  string
}

// Channel represents a public channel within a team
type Channel struct {
	ID   string // Channel ID
	Name string // Channel name
}

// User represents a user in a specific channel
type User struct {
	ChannelName string // The name of the channel the user is associated with
	UserID      string // The unique ID of the user
	Username    string // The username of the user
	Email       string // The email of the user
	FirstName   string // The first name of the user
	LastName    string // The last name of the user
	Nickname    string // The nickname of the user
}

// Struct for JSON output
type JSONOutput struct {
	Channels []JSONChannel `json:"Channels"`
}

type JSONChannel struct {
	Channel ChannelWithUsers `json:"Channel"`
}

type ChannelWithUsers struct {
	ChannelName string `json:"ChannelName"`
	Users       []User `json:"Users"`
}

const (
	debugLevel   LogLevel = "DEBUG"
	infoLevel    LogLevel = "INFO"
	warningLevel LogLevel = "WARNING"
	errorLevel   LogLevel = "ERROR"
)

const (
	defaultPort   = "8065"
	defaultScheme = "http"
	pageSize      = 60
	maxErrors     = 3
)

// Logging functions

// LogMessage logs a formatted message to stdout or stderr
func LogMessage(level LogLevel, message string) {
	if level == errorLevel {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(os.Stdout)
	}
	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf("[%s] %s\n", level, message)
}

// DebugPrint allows us to add debug messages into our code, which are only printed if we're running in debug more.
// Note that the command line parameter '-debug' can be used to enable this at runtime.
func DebugPrint(message string) {
	if debugMode {
		LogMessage(debugLevel, message)
	}
}

// getEnvWithDefaults allows us to retrieve Environment variables, and to return either the current value or a supplied default
func getEnvWithDefault(key string, defaultValue interface{}) interface{} {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getTeamID(mmClient model.Client4, teamName string) (string, error) {
	DebugPrint("Getting team ID for : " + teamName)

	ctx := context.Background()
	etag := ""

	team, response, err := mmClient.GetTeamByName(ctx, teamName, etag)

	if err != nil {
		LogMessage(errorLevel, "Failed to retrieve team ID: "+err.Error())
		return "", err
	}
	if response.StatusCode != 200 {
		LogMessage(errorLevel, "Function call to GetTeamByName returned bad HTTP response")
		return "", errors.New("bad HTTP response")
	}

	teamID := team.Id

	return teamID, nil
}

func getPublicChannelsForTeam(mmClient model.Client4, teamID string) ([]Channel, error) {
	DebugPrint("Getting public channels for team ID: " + teamID)

	var allChannels []Channel
	perPage := 50
	page := 0

	ctx := context.Background()
	etag := ""

	for {
		channels, response, err := mmClient.GetPublicChannelsForTeam(ctx, teamID, page, perPage, etag)
		if err != nil {
			LogMessage(errorLevel, "Failed to retrieve public channels: "+err.Error())
			return nil, err
		}
		if response.StatusCode != 200 {
			LogMessage(errorLevel, "Function call to GetPublicChannelsForTeam returned bad HTTP response")
			return nil, errors.New("bad HTTP response")
		}

		// Exit the loop if we're not getting any more channels returned
		if len(channels) == 0 {
			break
		}

		// Add found channels to main slice
		for _, ch := range channels {
			allChannels = append(allChannels, Channel{
				ID:   ch.Id,
				Name: ch.Name,
			})
		}

		page++
	}

	return allChannels, nil
}

func getUsersInChannels(mmClient model.Client4, channels []Channel, includeBots bool) ([]User, error) {
	DebugPrint("Getting users")

	var allUsers []User
	perPage := 50

	ctx := context.Background()
	etag := ""

	for _, channel := range channels {
		page := 0

		for {
			// Fetch users in the current channel
			users, response, err := mmClient.GetUsersInChannel(ctx, channel.ID, page, perPage, etag)
			if err != nil {
				errorMessage := fmt.Sprintf("Failed to retrieve users from channel: %s.  Error: %s", channel.Name, err.Error())
				LogMessage(errorLevel, errorMessage)
				return nil, err
			}
			if response.StatusCode != 200 {
				LogMessage(errorLevel, "Function call to GetUsersInChannel returned bad HTTP response")
				return nil, errors.New("bad HTTP response")
			}

			// Break if we've run out of users to process in this channel
			if len(users) == 0 {
				break
			}

			// Append users to the main slice
			for _, user := range users {

				if user.DeleteAt != 0 {
					DebugPrint("Skipping account as it's deactivated")
					continue
				}
				if user.IsBot && !includeBots {
					DebugPrint("Skipping bot account")
					continue
				}

				allUsers = append(allUsers, User{
					ChannelName: channel.Name,
					UserID:      user.Id,
					Username:    user.Username,
					Email:       user.Email,
					FirstName:   user.FirstName,
					LastName:    user.LastName,
					Nickname:    user.Nickname,
				})
			}

			page++
		}
	}

	return allUsers, nil
}

// outputCSV writes the slice of users to a CSV file or stdout if filename is empty
func outputCSV(users []User, filename string) error {
	var file *os.File
	var err error

	// Open the appropriate output
	if filename != "" {
		file, err = os.Create(filename)
		if err != nil {
			errorMessage := fmt.Sprintf("failed to create file: %v", err)
			LogMessage(errorLevel, errorMessage)
			return err
		}
		defer file.Close()
	} else {
		file = os.Stdout
	}

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	header := []string{"ChannelName", "UserID", "Username", "Email", "FirstName", "LastName", "Nickname"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write each user as a row
	for _, user := range users {
		row := []string{
			user.ChannelName,
			user.UserID,
			user.Username,
			user.Email,
			user.FirstName,
			user.LastName,
			user.Nickname,
		}
		if err := writer.Write(row); err != nil {
			errorMessage := fmt.Sprintf("failed to write CSV row: %v", err)
			LogMessage(errorLevel, errorMessage)
			return err
		}
	}

	return nil
}

func outputJSON(users []User, filename string) error {
	// Group users by channel name
	channelMap := make(map[string][]User)
	for _, user := range users {
		channelMap[user.ChannelName] = append(channelMap[user.ChannelName], user)
	}

	// Build JSONOutput structure
	var output JSONOutput
	for channelName, usersInChannel := range channelMap {
		output.Channels = append(output.Channels, JSONChannel{
			Channel: ChannelWithUsers{
				ChannelName: channelName,
				Users:       usersInChannel,
			},
		})
	}

	// Open file or use stdout
	var file *os.File
	var err error
	if filename != "" {
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()
	} else {
		file = os.Stdout
	}

	// Encode JSON with pretty-printing
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func main() {

	// Parse Command Line
	DebugPrint("Parsing command line")

	var MattermostURL string
	var MattermostPort string
	var MattermostScheme string
	var MattermostToken string
	var MattermostTeam string
	var OutputFileType string
	var OutputFileName string
	var IncludeBotsFlag bool
	var DebugFlag bool
	var VersionFlag bool

	flag.StringVar(&MattermostURL, "url", "", "The URL of the Mattermost instance (without the HTTP scheme)")
	flag.StringVar(&MattermostPort, "port", "", "The TCP port used by Mattermost. [Default: "+defaultPort+"]")
	flag.StringVar(&MattermostScheme, "scheme", "", "The HTTP scheme to be used (http/https). [Default: "+defaultScheme+"]")
	flag.StringVar(&MattermostToken, "token", "", "The auth token used to connect to Mattermost")
	flag.StringVar(&MattermostTeam, "team", "", "The name of the Mattermost team")
	flag.StringVar(&OutputFileType, "type", "CSV", "The typ of export file to be produced (CSV/JSON). [Default: CSV]")
	flag.StringVar(&OutputFileName, "file", "", "The name of the output file (Required)")
	flag.BoolVar(&IncludeBotsFlag, "includebots", false, "Flag to add bot accounts to the output, where present.")
	flag.BoolVar(&DebugFlag, "debug", false, "Enable debug output")
	flag.BoolVar(&VersionFlag, "version", false, "Show version information and exit")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "This utility allows you to generate a CSV or JSON file listing all users in all public channels within a Team")
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if VersionFlag {
		fmt.Printf("mm-channel-users - Version: %s\n\n", Version)
		os.Exit(0)
	}

	// If information not supplied on the command line, check whether it's available as an envrionment variable
	if MattermostURL == "" {
		MattermostURL = getEnvWithDefault("MM_URL", "").(string)
	}
	if MattermostPort == "" {
		MattermostPort = getEnvWithDefault("MM_PORT", defaultPort).(string)
	}
	if MattermostScheme == "" {
		MattermostScheme = getEnvWithDefault("MM_SCHEME", defaultScheme).(string)
	}
	if MattermostToken == "" {
		MattermostToken = getEnvWithDefault("MM_TOKEN", "").(string)
	}
	if !DebugFlag {
		DebugFlag = getEnvWithDefault("MM_DEBUG", debugMode).(bool)
	}

	debugMode = DebugFlag

	DebugMessage := fmt.Sprintf("Parameters: \n  MattermostURL=%s\n  MattermostPort=%s\n  MattermostScheme=%s\n  MattermostToken=%s\n  Team=%s\n  FileType=%s\n  FileName=%s\n  IncludeBots=%t\n",
		MattermostURL,
		MattermostPort,
		MattermostScheme,
		MattermostToken,
		MattermostTeam,
		OutputFileType,
		OutputFileName,
		IncludeBotsFlag,
	)
	DebugPrint(DebugMessage)

	// Validate required parameters
	DebugPrint("Validating parameters")
	var cliErrors bool = false
	if MattermostURL == "" {
		LogMessage(errorLevel, "The Mattermost URL must be supplied either on the command line of vie the MM_URL environment variable")
		cliErrors = true
	}
	if MattermostScheme == "" {
		LogMessage(errorLevel, "The Mattermost HTTP scheme must be supplied either on the command line of vie the MM_SCHEME environment variable")
		cliErrors = true
	}
	if MattermostToken == "" {
		LogMessage(errorLevel, "The Mattermost auth token must be supplied either on the command line of vie the MM_TOKEN environment variable")
		cliErrors = true
	}
	if MattermostTeam == "" {
		LogMessage(errorLevel, "A Mattermost team name is required to use this utility.")
		cliErrors = true
	}

	fileType := strings.ToUpper(OutputFileType)
	if fileType != "CSV" && fileType != "JSON" {
		LogMessage(errorLevel, "Output file type can be one of CSV or JSON only.")
		cliErrors = true
	}

	if cliErrors {
		flag.Usage()
		os.Exit(1)
	}

	// Prepare the Mattermost connection
	mattermostConenction := mmConnection{
		mmURL:    MattermostURL,
		mmPort:   MattermostPort,
		mmScheme: MattermostScheme,
		mmToken:  MattermostToken,
	}

	mmTarget := fmt.Sprintf("%s://%s:%s", mattermostConenction.mmScheme, mattermostConenction.mmURL, mattermostConenction.mmPort)

	DebugPrint("Full target for Mattermost: " + mmTarget)
	mmClient := model.NewAPIv4Client(mmTarget)
	mmClient.SetToken(mattermostConenction.mmToken)
	DebugPrint("Connected to Mattermost")

	LogMessage(infoLevel, "Processing started - Version: "+Version)

	teamID, err := getTeamID(*mmClient, MattermostTeam)

	if err != nil {
		LogMessage(errorLevel, "Failed to retrieve team ID.  Error: "+err.Error())
		os.Exit(1)
	}

	DebugPrint("Team ID: " + teamID)

	channels, err := getPublicChannelsForTeam(*mmClient, teamID)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to retrieve public channels for team '%s'.  Error: %s", MattermostTeam, err.Error())
		LogMessage(errorLevel, errorMessage)
		os.Exit(2)
	}

	for _, ch := range channels {
		debugMessage := fmt.Sprintf("Channel ID: %s,  Name: %s", ch.ID, ch.Name)
		DebugPrint(debugMessage)
	}

	users, err := getUsersInChannels(*mmClient, channels, IncludeBotsFlag)

	if err != nil {
		LogMessage(errorLevel, "Failed to retrieve users")
		os.Exit(3)
	}

	if fileType == "CSV" {
		err := outputCSV(users, OutputFileName)
		if err != nil {
			LogMessage(errorLevel, "Error writing CSV")
			os.Exit(4)
		}
	}
	if fileType == "JSON" {
		err := outputJSON(users, OutputFileName)
		if err != nil {
			LogMessage(errorLevel, "Error writing JSON file")
			os.Exit(5)
		}
	}

	LogMessage(infoLevel, "Processing complete!")

	os.Exit(0)

}
